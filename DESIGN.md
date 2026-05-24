# ✧ chisel — design document ✧

## architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Go TUI (bubbletea)                    │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────────┐  │
│  │  binder  │  │  editor  │  │     llm panel        │  │
│  │ (tree)   │  │ (markdown│  │ (responses, research, │  │
│  │          │  │  text)   │  │  questions, analysis) │  │
│  └──────────┘  └──────────┘  └──────────────────────┘  │
│                         │                               │
│            NDJSON over stdin/stdout (subprocess)        │
│                         │                               │
│  ┌──────────────────────────────────────────────────┐   │
│  │              Python backend (chisel.py)           │   │
│  │  ┌────────┐  ┌──────────┐  ┌────────────────┐   │   │
│  │  │  llm   │  │ research │  │   analysis     │   │   │
│  │  │ calls  │  │ gather   │  │   (mirror)     │   │   │
│  │  └────────┘  └──────────┘  └────────────────┘   │   │
│  └──────────────────────────────────────────────────┘   │
│                         │                               │
│              OpenAI-compatible HTTP API                 │
│                         │                               │
│  ┌──────────┐  ┌──────────┐  ┌────────────────────┐    │
│  │ LM Studio│  │llama.cpp │  │ cloud (openai/     │    │
│  │ (local)  │  │ (local)  │  │ anthropic)         │    │
│  └──────────┘  └──────────┘  └────────────────────┘    │
└─────────────────────────────────────────────────────────┘
```

**Go TUI** handles everything the user touches: the binder tree, the text editor, the LLM panel, keyboard shortcuts, pane layout switching. **Python backend** handles everything the LLM touches: model calls, image encoding (if vision features arrive later), research gathering, stylistic analysis. They communicate via NDJSON over stdin/stdout — same pattern as the screenshot cataloger. This split means the TUI stays responsive during LLM calls and the Python side can be swapped out or replaced without touching the UI.

## data model

### project structure on disk

```
my-project/
├── manifest.jsonl        # scene metadata, one JSON object per line
├── config.json           # project-level settings
├── scenes/               # your writing — one .md file per scene
│   ├── ch01-arrival.md
│   ├── ch02-the-garden.md
│   └── notes_2026-05.md
├── research/             # LLM-gathered notes tagged to scenes
│   ├── roman-architecture.md
│   └── color-symbolism.md
└── exports/              # compiled output (future)
    └── manuscript.md
```

### manifest format (JSONL)

Each line is a self-contained JSON object. The `id` is the filename stem. The manifest is append-only during normal use; reorder events rewrite the entire file. Same pattern as the screenshot cataloger.

```json
{
  "id": "ch01-arrival",
  "file": "scenes/ch01-arrival.md",
  "title": "Chapter 1 — Arrival",
  "status": "revised",
  "word_count": 1247,
  "pov": "first",
  "draft_order": 1,
  "tags": ["opening", "establishing"],
  "created": "2026-05-24T14:00:00",
  "modified": "2026-05-24T22:00:00",
  "research_refs": ["roman-architecture"],
  "notes": "needs a tighter ending — the last paragraph drifts"
}
```

### research notes

Research files live in `research/` and are plain markdown. The LLM populates them on request. **By default, research notes are auto-tagged to the current scene** — the scene's `research_refs` array updates automatically. A toggle in settings (`research.auto_tag`) disables this, leaving new notes untagged until the user assigns them manually. The research panel shows all notes tagged to the current scene.

## pane configurations

The user switches between three layouts with a single keystroke. No window management, no mouse. Each layout is a *mode*, not a fixed split — the TUI reflows on toggle.

### mode 1: editor only (`ctrl+1`)
```
┌────────────────────────────────────────────────┐
│                  editor                        │
│                                                │
│        (full-screen writing)                   │
│                                                │
└────────────────────────────────────────────────┘
```
For focused drafting. Nothing else on screen.

### mode 2: binder + editor (`ctrl+2`)
```
┌──────────────┬─────────────────────────────────┐
│   binder     │           editor                │
│   (tree)     │                                 │
│              │      (writing pane)             │
│ ▸ ch01       │                                 │
│   ▸ arrival  │                                 │
│   ▸ garden   │                                 │
│ ▸ ch02       │                                 │
│   ▸ escape   │                                 │
│ ▸ notes      │                                 │
└──────────────┴─────────────────────────────────┘
```
For navigating between scenes while writing.

### mode 3: binder + editor + llm (`ctrl+3`)
```
┌──────────┬──────────────┬──────────────────────┐
│  binder  │   editor     │     llm panel        │
│  (tree)  │              │                      │
│          │  (writing)   │  rewrite suggestions  │
│ ▸ ch01   │              │  research notes       │
│   ▸ arr… │              │  analysis output      │
│   ▸ gar… │              │  ask responses        │
│ ▸ ch02   │              │                      │
└──────────┴──────────────┴──────────────────────┘
```
For working with the LLM alongside the text.

## llm integration

### provider abstraction

The Python backend talks to any OpenAI-compatible endpoint. Configuration lives in `config.json`:

```json
{
  "llm": {
    "api_base": "http://localhost:1234/v1",
    "model": "gemma-4-e4b",
    "max_tokens": 2048,
    "temperature": 0.7
  },
  "mirror": {
    "api_base": "http://localhost:1234/v1",
    "model": "cass/gemma-4-e4b-it",
    "max_tokens": 1024,
    "temperature": 0.3
  }
}
```

Two model slots: `llm` for general-purpose tasks (rewrite, generate, research, ask), `mirror` for stylistic analysis — your fine-tuned model that surfaces patterns in your writing. They can point at the same endpoint with different model names, or at entirely different servers.

### operations

Each LLM operation is a typed request from the TUI to the Python backend:

| operation | what it does | uses mirror? |
|-----------|-------------|:---:|
| `rewrite` | suggest alternatives for selected text | no |
| `generate` | continue from cursor with optional guidance | no |
| `summarize` | summarize selection, scene, or chapter | no |
| `ask` | answer a question about the text or research a topic | no |
| `analyze` | surface tics, rhythm issues, overused words | yes |
| `research` | gather notes on a topic and tag to current scene | no |

### interaction pattern

Operations are triggered by keystrokes, not a chat interface. The user selects text (or doesn't — some operations work on the whole scene), hits a key, and the response appears in the LLM panel. No back-and-forth conversation. The panel is a *viewer*, not a chatbot.

```
Keystroke map:
  Ctrl+R     → rewrite selected text
  Ctrl+G     → generate from cursor
  Ctrl+Shift+S → summarize selection/scene
  Ctrl+K     → ask a question (prompt bar opens at bottom)
  Ctrl+A     → analyze style of current scene (mirror)
  Ctrl+F5    → research topic (prompt bar, results in research/)
```

## editor

The editor is modeless by default. Standard shortcuts (Ctrl+S saves, Ctrl+F finds, Ctrl+Z undoes). Vim bindings as an opt-in toggle in settings, not the default. The goal is that someone who hasn't used a CLI editor in years can sit down and write.

### file format

Plain markdown. No frontmatter required. The manifest holds metadata. A scene file might look like:

```markdown
# Chapter 1 — Arrival

The train pulled in at dusk. Rain had been falling for three hours
and showed no sign of stopping.

She stepped onto the platform alone.
```

If the file has a `# Title` on the first line, the editor can pull the title from there as a fallback if the manifest entry is missing. But the manifest is the source of truth.

## decisions

- **One scene per file.** Each `.md` file in `scenes/` is one scene. No delimiters, no multi-scene files. The binder tree provides organization; the filesystem stays simple.
- **Research auto-tags to current scene by default.** A toggle in settings disables this.
- **Built-in revision history.** Every save is tracked. See below.
- **Export is a v1 feature.** Concatenating scenes in draft order into `exports/manuscript.md` is essential for reading the whole work.

## revision history

Chisel tracks every save automatically. No manual commits, no staging area — every Ctrl+S creates a snapshot. The user browses history from within the editor: jump back to any saved version of the current scene, compare side-by-side, restore a passage.

### backend options

Two candidates for the storage layer:

**git** — ubiquitous, already installed on most systems. The tool initializes a `.git` inside the project directory on creation. Every save triggers a commit with a structured message (`scene: ch01-arrival — 1,247 words`). History browsing reads from `git log` and `git diff`. The downside: git's staging area is conceptually wrong for a writing tool where "save = snapshot" should be a single atomic operation.

**jj (Jujutsu)** — Google's git-compatible VCS. No staging area, automatic commits on every change, rebase-first workflow. A save in chisel maps directly to `jj new` + `jj describe`. History browsing is cleaner because jj's log shows all snapshots as first-class commits, not a messy reflog. The downside: jj is newer and not yet universally installed.

**Recommendation:** target git for v1 (zero dependency friction), but keep the revision API abstract so jj can slot in later as a configurable backend. The user-facing behavior is identical either way — Ctrl+S creates a snapshot, the history browser shows a timeline.
