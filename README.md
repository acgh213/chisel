# ✧ chisel ✧

a local-first, markdown-native writing tool with LLM augmentation. for essays, blog posts, and fiction — the kind of writing where you need to move scenes around at 2am without the tool getting in your way.

**Scrivener in a terminal. With a local AI that helps you think.**

## what

chisel is a TUI writing environment that does what scrivener does for structure (binder tree, scene organization, corkboard, outline, character sheets) but in a terminal, with plain markdown files, and with a local LLM that helps you revise, research, and analyse your writing.

it runs entirely locally. your writing lives in `.md` files on your filesystem. the LLM features are optional, configurable, and private — no cloud, no subscription, no uploading your half-finished novel to a server somewhere.

## quick start

```bash
chisel new my-novel         # scaffold a project
cd my-novel && chisel       # launch the TUI
```

**Requires:** Go 1.22+ (or grab the binary from dist/). Python 3 + `pip install openai` for LLM features (optional).

## keybindings

| shortcut | action |
|----------|--------|
| `Ctrl+1/2/3` | pane modes (editor / binder+editor / +LLM panel) |
| `Ctrl+S` | save scene (auto-commits to git) |
| `Ctrl+H` | revision history browser |
| `Ctrl+R` | rewrite selection (LLM) |
| `Ctrl+G` | generate from cursor (LLM) |
| `Ctrl+Shift+S` | summarize scene (LLM) |
| `Ctrl+K` | ask a question (LLM) |
| `Ctrl+A` | style analysis (mirror model) |
| `Ctrl+F5` | research topic (LLM) |
| `Ctrl+E` | export manuscript |
| `Ctrl+Shift+E` | export to docx (via pandoc) |
| `Ctrl+B` | corkboard view |
| `Ctrl+O` | outline view |
| `Ctrl+L` | timeline view |
| `Ctrl+T` | cycle theme (peach/dark/light/forest/ocean) |
| `Ctrl+Shift+P` | writing sprint (25 min pomodoro) |
| `Ctrl+Shift+R` | reading mode |
| `Ctrl+Shift+C` | character sheets |
| `Ctrl+Shift+N` | scene notes |
| `n/d/F2` | new/delete/rename scene |
| `K/J` | reorder scenes |
| `t` / `T` | tag / filter by tag |
| `Tab` | switch focus (binder ↔ editor) |

## features

### phase 0 — scaffolding
- `chisel new` creates a complete project with `scenes/`, `research/`, `exports/`, `characters/`, `config.json`, `manifest.jsonl`, `.gitignore`
- Git repo initialised via go-git — no shelling out to the git CLI

### phase 1 — binder + editor
- **Binder tree** — recursive directory scan of `scenes/`, expand/collapse folders, status indicators (draft · revised · done)
- **Markdown editor** — modeless editing with `bubbles/textarea`, Ctrl+S/Ctrl+Z/Ctrl+F
- **Scene CRUD** — `n` inline prompt for new scene, `d` delete with confirm, `F2` inline rename, `K`/`J` reorder
- **Pane modes** — Ctrl+1 (editor-only), Ctrl+2 (binder+editor), Ctrl+3 (binder+editor+LLM)
- **Status bar** — scene name, word count, modified indicator, session timer, daily goal progress

### phase 2 — revision history
- Every Ctrl+S auto-commits to git via go-git
- `RevisionBackend` interface — abstract so jj can slot in later
- Ctrl+H opens history browser: revision list, diff view (enter), restore (r)
- Diff shows unified diff between any two snapshots

### phase 3 — LLM integration
- `chisel.py` Python backend communicates via NDJSON over stdin/stdout
- Provider-flexible: works with LM Studio, llama.cpp, OpenAI, Anthropic — any OpenAI-compatible API
- Two model slots: `llm` (general) and `mirror` (stylistic analysis)
- **Rewrite** (Ctrl+R) — alternatives in LLM panel
- **Generate** (Ctrl+G) — continue from cursor
- **Summarize** (Ctrl+Shift+S) — summary of selection or scene
- **Ask** (Ctrl+K) — inline prompt, response streams token by token
- Backend auto-detected; TUI runs without it — graceful degradation

### phase 4 — mirror + research
- **Style analysis** (Ctrl+A) — mirror model surfaces overused words, rhythm issues, tics
- **Research** (Ctrl+F5) — prompt for topic, LLM researches, saves to `research/{slug}.md`, auto-tags current scene via `research_refs`
- **Tag system** — `t` adds tag, `T` filters binder by tag

### phase 5 — export + polish
- **Manuscript export** (Ctrl+E) — concatenates scenes in draft order to `exports/manuscript.md`
- **Docx export** (Ctrl+Shift+E) — via pandoc wrapper
- **Corkboard** (Ctrl+B) — grid of scene cards (title, word count, status, first line)
- **Outline** (Ctrl+O) — collapsible titles with status icons (○ ◑ ●)
- **Timeline** (Ctrl+L) — tree view with dates
- **5 themes** — peach, dark, light, forest, ocean (Ctrl+T to cycle)
- **Vim bindings** — opt-in toggle (Ctrl+Shift+V, saved to config)
- **Daily word goals** — configurable target in `config.json`, progress in status bar
- **Writing sprints** — 25 min pomodoro (Ctrl+Shift+P)
- **Reading mode** — full-screen, no chrome (Ctrl+Shift+R, any key exits)
- **Typewriter mode** — toggle with Ctrl+Shift+T

### phase 6 — character sheets + scene notes
- `characters/` directory auto-created; `.md` profiles with name, description, arc, relationships
- Character browser (Ctrl+Shift+C)
- Per-scene planning notes (Ctrl+Shift+N)

### phase 7 — jj backend
- `JJBackend` implements `RevisionBackend` using the `jj` CLI
- Enabled by setting `history.backend: "jj"` in `config.json`
- Works as a drop-in replacement for the default git backend

## architecture

```
Go TUI (bubbletea)  ─── NDJSON over stdin/stdout ─── Python backend (chisel.py)
  ├── binder         │                                  ├── LLM calls
  ├── editor         │                                  ├── research
  ├── LLM panel      │                                  └── mirror analysis
  ├── history        │
  └── export         │
                     └──  OpenAI-compatible API (LM Studio, llama.cpp, OpenAI, Anthropic)
```

The Go TUI handles everything the user touches. The Python backend handles everything the LLM touches. They communicate via NDJSON — the TUI spawns `chisel.py` as a subprocess and pipes requests/responses.

## configuration

`config.json` in the project root:

```json
{
  "llm": {
    "api_base": "http://localhost:1234/v1",
    "model": "",
    "max_tokens": 2048,
    "temperature": 0.7
  },
  "mirror": {
    "api_base": "http://localhost:1234/v1",
    "model": "",
    "max_tokens": 1024,
    "temperature": 0.3
  },
  "history": { "backend": "git" },
  "editor": { "vim_mode": false },
  "goals": { "daily_word_target": 500 },
  "theme": "peach"
}
```

## building

```bash
# Windows
go build -o chisel.exe .

# macOS / Linux
go build -o chisel .
```

Or run the build script: `.\build.ps1` (Windows) produces a distributable `dist\chisel\` with the binary + Python backend.

## prerequisites

- **TUI only (no LLM):** nothing beyond the chisel binary
- **LLM features:** Python 3 + `pip install openai`
- **Docx export:** [pandoc](https://pandoc.org/)
- **jj backend (optional):** [Jujutsu](https://github.com/jj-vcs/jj) in PATH

## status

**v1.2.0** — all 7 phases implemented and tested. ready for everyday writing.

## core philosophy

- **plain markdown files in folders.** if chisel disappears tomorrow, your writing is still there. no proprietary formats, no lock-in.
- **scene-level organization.** a binder tree that tracks scenes, chapters, drafts, word counts, status — the structural stuff scrivener gets right.
- **LLM as a tool, not a co-author.** rewrite suggestions, research gathering, stylistic analysis, summarization. the model helps you think. it doesn't replace your voice.
- **local-first, provider-flexible.** runs with LM Studio, llama.cpp, or any OpenAI/Anthropic-compatible API. you choose where the model lives.
- **pane configurations as modes.** toggle with a keystroke, not window management.
- **auto-committing revision history.** every save is a snapshot. browse, diff, restore from within the editor.

## non-goals

chisel is not a code editor, not a cloud service, not a publishing tool, not a replacement for scrivener, and not collaborative. it does ten things, locally, in a terminal, and you can read the source in an afternoon.

## inspiration

- [Scrivener](https://www.literatureandlatte.com/scrivener/overview) — the gold standard for scene-level writing organization
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — the TUI framework that makes terminal apps beautiful
- [go-git](https://github.com/go-git/go-git) — pure Go git implementation for revision history
- [Jujutsu](https://github.com/jj-vcs/jj) — the git-compatible VCS with first-class snapshots

## license

MIT — see [LICENSE](LICENSE).
