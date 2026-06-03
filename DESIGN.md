# ✧ chisel — design document ✧

> **Status (June 2026):** This document is the north-star vision for chisel.
> Features shipped through Phase 19 are marked **[shipped]** inline below.
> The original v1.2 full implementation lives on `archive/chisel-full` (reference only).

**Shipped through Phase 19:**
- Stabilized TUI layout (border-aware sizing, no overflow) ✓
- `core/` package with zero charmbracelet imports (GUI-ready seam) ✓
- YAML frontmatter metadata per scene (status glyphs, word count, timestamps) ✓
- Git-backed revision history — auto-snapshot, browse/diff/restore (Ctrl+H) ✓
- Corkboard (F2) and outliner (F3) structural views ✓
- Timeline view — date-sorted scene list with cross-hop (F4) ✓
- Compile/export to manuscript.md + optional pandoc .docx (Ctrl+E) ✓
- Binder-side CRUD — create/rename/delete files and folders (n/N/r/d) ✓
- `chisel init` subcommand — 3 templates, interactive + non-interactive ✓
- Right panel — world panel: characters + locations, passive binder-driven inspector (F5) ✓
- Character YAML frontmatter (name, role, description, tags, arc, voice, relationships) ✓
- Location sheets — `locations/` directory, LocationMeta with Atmosphere/Significance (Phase 9) ✓
- Timeline view — F4 with dated/undated sections, cross-hop F2/F3/F4 (Phase 10) ✓
- Quick-note popup — backtick from any state, saves to `notes/scratch.md` (Phase 11) ✓
- Scene notes — `notes` frontmatter field, W toggles World Index/Scene Notes, e edits inline (Phase 12) ✓
- Full-text search overlay — Ctrl+F from any state, body-only search (Phase 13) ✓
- Reading mode — F6 full-screen centered column, word-wrapped prose (Phase 14) ✓
- Themes + session stats + sprint timer — 4 dark themes (Ctrl+T), word count, F7 pomodoro (Phase 19) ✓

**Pending:** LLM assist, light theme, tag browser, project statistics, GUI.

## architecture

The architecture has evolved through nineteen phases. What was originally designed as a Go TUI + Python LLM backend is now a pure-Go TUI with a strict `core`/`tui` split and seven overlay/structural views. The LLM layer (pending) will slot back in as a `core` package, keeping the same boundary.

```
┌──────────────────────────────────────────────────────────┐
│                   chisel binary (Go)                      │
│                                                           │
│  ┌───────────────────────────────────────────────────┐   │
│  │              tui/ (Bubble Tea + Lip Gloss)         │   │
│  │  ┌────────┐ ┌────────┐ ┌───────────────────┐      │   │
│  │  │ binder │ │ editor │ │  right panel      │      │   │
│  │  │ (tree) │ │(md txt)│ │  (world panel)    │      │   │
│  │  └────────┘ └────────┘ └───────────────────┘      │   │
│  │  ┌──────────────────────────────────────────┐     │   │
│  │  │  structural views                        │     │   │
│  │  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ │     │   │
│  │  │  │corkboard │ │ outliner │ │ timeline │ │     │   │
│  │  │  │(F2)      │ │(F3)      │ │(F4)      │ │     │   │
│  │  │  └──────────┘ └──────────┘ └──────────┘ │     │   │
│  │  └──────────────────────────────────────────┘     │   │
│  │  ┌──────────────────────────────────────────┐     │   │
│  │  │  overlays (global, from any state)       │     │   │
│  │  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ │     │   │
│  │  │  │quicknote │ │ search   │ │ reader   │ │     │   │
│  │  │  │(backtick)│ │(Ctrl+F)  │ │(F6)      │ │     │   │
│  │  │  └──────────┘ └──────────┘ └──────────┘ │     │   │
│  │  └──────────────────────────────────────────┘     │   │
│  │  ┌──────────────────────────────────────────┐     │   │
│  │  │  themes + session (Phase 19)             │     │   │
│  │  │  Ctrl+T cycles, persists .chisel.yaml    │     │   │
│  │  │  F7 25-min sprint timer w/ word count    │     │   │
│  │  └──────────────────────────────────────────┘     │   │
│  │                         │                          │   │
│  │                    core/ types                     │   │
│  │  ┌──────────┬──────────┬──────────┬──────────┐   │   │
│  │  │ project  │  scene   │ metadata │ revision │   │   │
│  │  │ outline  │  export  │  crud    │ scaffold │   │   │
│  │  │ character│ location │ timeline │  notes   │   │   │
│  │  │ search   │  config  │          │          │   │   │
│  │  └──────────┴──────────┴──────────┴──────────┘   │   │
│  └───────────────────────────────────────────────────┘   │
│                                                           │
│  ┌───────────────────────────────────────────────────┐   │
│  │  pending: llm/ (OpenAI-compatible HTTP API)        │   │
│  │  ┌────────┐ ┌──────────┐ ┌────────────────┐      │   │
│  │  │  llm   │ │ research │ │   analysis     │      │   │
│  │  │ calls  │ │ gather   │ │   (mirror)     │      │   │
│  │  └────────┘ └──────────┘ └────────────────┘      │   │
│  └───────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────┘
```

## data model

### [shipped] project structure on disk

```
my-novel/
├── README.md
├── .chisel.yaml           # app prefs: theme, daily goal (gracefully ignored when missing)
├── scenes/                # your writing — one .md file per scene
│   ├── ch01-opening.md
│   ├── ch02-rising-action.md
│   └── notes.md
├── characters/            # character profiles — .md with YAML frontmatter
│   ├── protagonist.md
│   └── antagonist.md
├── locations/             # location descriptions — .md with YAML frontmatter
│   ├── the-docks.md
│   └── borderlands.md
├── notes/                 # quick-note scratchpad (auto-created on first use)
│   └── scratch.md
├── exports/               # compiled output
│   ├── manuscript.md
│   └── manuscript.docx   (if pandoc installed)
└── .git/                  # auto-created by go-git on first save
```

No `manifest.jsonl`. No sidecar files for scene content. The filesystem and YAML frontmatter are the data model. `.chisel.yaml` is the sole config file — it carries app preferences (theme, daily goal), not project structure. It is gracefully ignored when missing; chisel works with defaults.

### [shipped] scene format

Each scene is a `.md` file with optional YAML frontmatter:

```markdown
---
title: Chapter One — Arrival
status: revised
synopsis: She steps onto the platform alone.
tags:
  - opening
  - rain
draft_order: 1
word_target: 2000
pov: first
timeline_date: 1847-03-15
notes: The rain motif should echo the prologue's storm — maybe the same words.
word_count: 1247
created: 2026-05-31T09:50:24-04:00
modified: 2026-05-31T09:50:24-04:00
---
# Chapter One

The train pulled in at dusk.
```

Files without frontmatter are plain markdown and open/save cleanly. The `word_count`, `created`, and `modified` fields are auto-managed — the user never edits them. Status glyphs appear in the binder: ○ draft, ◐ revised, ● done. `timeline_date` is optional; scenes without it appear in the timeline's undated section. `notes` is free-form scene-level annotation, editable inline from the world panel (W → e).

### [shipped] character format

Character profiles in `characters/` follow the same pattern with a richer frontmatter schema:

```markdown
---
name: Elara Voss
role: Protagonist
description: A cartographer who maps the unmappable. Haunted by the cities she's erased.
arc: Learns that the map is not the territory — and that some places should stay uncharted.
voice: Precise, clipped sentences. Uses cartographic metaphors even in casual speech.
relationships:
  - Kael: Former apprentice, now her fiercest critic. Unresolved.
tags:
  - pov
  - haunted
---
## Background

Elara grew up in the borderlands between Ha'ren and the outer rings...
```

Arc, voice, and relationships are Phase 12 additions — they render in the right panel's entity detail view below the description.

### [shipped] location format

Location sheets in `locations/` mirror the character schema (Phase 9):

```markdown
---
name: The Docks
type: District
description: A labyrinth of salt-warped piers and half-sunk warehouses at the city's eastern edge.
atmosphere: Damp, briny, perpetually fogged. The sound of creaking wood and distant foghorns.
significance: Where Elara first meets Kael. The drowning motif originates here.
tags:
  - eastern-district
  - water
---
## Notable features

- The Sunken Bell tavern — neutral ground for smugglers
- Pier 7 — collapsed during the flood of '42, never rebuilt
```

### [pending] manifest format (v1.2 reference)

The original v1.2 design used a JSONL manifest. This has been superseded by YAML frontmatter in the files themselves. The manifest approach is preserved here as design history — it was the right call for the v1.2 architecture with a Python backend, but frontmatter-in-file eliminates the sync problem entirely.

### [pending] research notes

The original design included a `research/` directory with auto-tagged LLM-gathered notes. When the LLM layer returns, this pattern will be re-evaluated against the frontmatter-in-file approach.

## [shipped] revision history

Chisel tracks every save automatically. Every Ctrl+S creates a git snapshot via `go-git` (pure Go, no system git binary). The user browses history with Ctrl+H: a scrollable list of snapshots with timestamps, colored unified diffs, and one-key restore.

### backend

**Git** (shipped) — `go-git/v5` handles everything in-process. The `.git` directory lives inside the project root, initialized lazily on first save. Empty commits (save with no changes) are silently discarded.

**jj** (pending) — the `RevisionBackend` interface was designed for this swap: `Snapshot`, `Log`, `Diff`, `Restore`. A `JjBackend` implementing the same interface can slot in without changing any TUI code.

## [shipped] structural views

### corkboard (F2)

Scrivener's corkboard: a scrollable grid of index cards for the scenes in the current binder folder. Each card shows title, status, word-count progress, and synopsis excerpt. Cards are fixed-width (26 chars) so the grid aligns regardless of content. Navigation with arrow keys; Enter opens the selected scene.

### outliner (F3)

A collapsible project-wide outline. Every file and folder appears as a tree row with indentation. Scene rows carry a right-aligned column with status glyph and word-count (or word-count/target). Folders expand/collapse independently of the binder. Word-count targets that have been met render in green.

### timeline (F4)

A date-sorted view of all project scenes (Phase 10). Scenes with a `timeline_date` frontmatter field appear first in ascending order; undated scenes follow alphabetically with a divider row. Columns: date, status glyph, title, word count. F2/F3/F4 cross-hop between all structural views — whichever you open last, the others are one key away.

### right panel (F5)

A passive, binder-driven inspector pane. No cursor, no focus, no key handling — it reflects whatever the binder has selected. Three modes, unified under a single world panel (Phases 9, 12):

- **Entity detail:** when a file in `characters/` or `locations/` is selected, shows the full profile — name, role/type, description, arc/atmosphere, voice/significance, relationships, tags, and body notes. Characters and locations share the same rendering code path.
- **World index:** when anything else is selected, lists all characters and locations with roles/types
- **Scene notes:** W toggles the panel to show the current scene's `notes` field. e opens an inline editor pre-filled with the note text. When the scene is also open in the editor, edits route through the in-memory scene to prevent clobbering unsaved body changes.
- **Empty hint:** when no `characters/` or `locations/` directory exists

Toggled with F5. The layout rebalances to three panes (binder shrinks, editor stays, right panel appears on the right).

## [shipped] overlays

Overlays are checked before all other key dispatch — they own the keyboard globally.

### quick-note (backtick)

A floating single-line popup (Phase 11). Backtick opens it from any state — binder, editor, structural views, history browser. Type a thought, Enter saves a timestamped entry to `notes/scratch.md`, Esc cancels. The popup overlays the current view so you never lose your place. The `notes/` directory and `scratch.md` are created on first use.

### search (Ctrl+F)

Full-text search overlay (Phase 13). Ctrl+F opens a query popup from any state; Enter runs the search across all `.md` files (body text only — frontmatter excluded). Results show the matching file, title, line number, and trimmed match line. j/k navigate results; Enter opens the selected scene. Esc returns to the query input; Esc again closes. Note: cursor positioning to the exact match line is not implemented — bubbles/textarea lacks a go-to-line API.

### reading mode (F6)

Full-screen centered reading view (Phase 14). F6 opens the current scene in a word-wrapped column — no binder, no right panel, no chrome. ↑/↓/j/k scroll line-by-line; Ctrl+D/U jump half a page. F6 or Esc exits. Typewriter centering and paragraph dim are deferred: bubbles/textarea has no scroll-offset setter and no per-line styling hook.

## [shipped] themes + session (Phase 19)

### themes

Four dark themes cycle with Ctrl+T: peach (warm amber), forest (green-gold), ocean (blue-steel), midnight (deep violet). The active theme persists to `.chisel.yaml` and is loaded on project open. All `Style*` vars are bare declarations; `rebuildStyles()` is the single source of truth, called from `init()` and on every theme change. Light theme is deferred — it requires full-screen background painting (chisel relies on the terminal's default background).

### session word count

Words written this session are accumulated on every save across file switches. The delta between `fileLoadWords` (baseline at load/save) and current word count is added to `sessionWords` — only positive deltas count, so saving the same file twice doesn't double-count. Shown in the status bar as `+N today` or `+N/G today` when a daily goal is configured in `.chisel.yaml`.

### sprint timer

F7 starts a 25-minute countdown from any view. The status bar shows `Sprint MM:SS +N words` throughout. Second F7 stops early; the timer auto-stops at 0:00. Word gain is shown on stop. The sprint tick runs on Bubble Tea's `tea.Tick` — no goroutines.

## [pending] LLM integration

### provider abstraction

The LLM layer will talk to any OpenAI-compatible endpoint. Configuration will live in per-scene or per-project settings (exact format TBD — likely YAML frontmatter on a project-level config or environment variables). The design from v1.2 with separate `llm` and `mirror` model slots is still the target:

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

Two model slots: `llm` for general-purpose tasks (rewrite, generate, research, ask), `mirror` for stylistic analysis — a fine-tuned model that surfaces patterns in your writing.

### operations

Each LLM operation is a typed request, triggered by keystroke:

| operation | keystroke | uses mirror? | description |
|-----------|-----------|:---:|-------------|
| `rewrite` | Ctrl+R | no | suggest alternatives for selected text |
| `generate` | Ctrl+G | no | continue from cursor |
| `summarize` | Ctrl+Shift+S | no | summarize selection or scene |
| `ask` | Ctrl+K | no | answer question or research topic |
| `analyze` | Ctrl+A | yes | surface tics, rhythm, overused words |
| `research` | Ctrl+F5 | no | gather notes and tag to current scene |

### interaction pattern

Operations are triggered by keystrokes, not a chat interface. The user selects text (or doesn't — some operations work on the whole scene), hits a key, and the response appears in a panel. No back-and-forth conversation.

## [shipped] editor

The editor is modeless. Standard shortcuts: Ctrl+S saves (+ snapshots + session word accumulation), Ctrl+F finds, Ctrl+Z undoes. No vim mode. The goal is that someone who hasn't used a CLI editor in years can sit down and write.

Scenes open from the binder (Enter) or from structural views (Enter in corkboard/outliner/timeline) or from search results (Enter). The editor preserves unsaved changes when switching scenes — a modified indicator (●) appears in the status bar.

## [shipped] binder CRUD

Create, rename, and delete files and folders directly in the binder:

- `n` — new scene (prompt for name, creates `<name>.md` in current folder)
- `N` — new folder
- `r` — rename selected (auto-preserves `.md` extension, pre-fills current name)
- `d` — delete selected (y=confirm, recursive for folders)

These keys only fire when the binder is focused. When the editor has focus, they insert literal characters. The prompt bar occupies the status-bar row during CRUD and is dismissed on Enter (confirm) or Esc (cancel).

## [shipped] project scaffolding

`chisel init` scaffolds new projects from three templates:

- **minimal** — README.md only
- **novel** — `scenes/`, `characters/`, `locations/` with two seeded chapters (draft_order 1 and 2, status: draft)
- **short-stories** — single `story-01.md`

Interactive mode prompts for name and template choice. Non-interactive: `--template <tmpl>` + optional positional directory + `--no-open`.

## decisions

- **One scene per file.** Each `.md` file is one scene. No delimiters, no multi-scene files.
- **YAML frontmatter over JSONL manifest.** Frontmatter lives in the file itself — no sync problem, no desync possible. The v1.2 manifest approach was correct for its architecture but wrong for a local-first TUI where the filesystem is the API.
- **`core` stays charmbracelet-free.** This is the GUI-ready seam. Everything that touches the terminal lives in `tui/`. A Wails or Fyne frontend can import `core` directly.
- **Passive right panel.** The world panel has no cursor and handles no keys. It's a pure view — the binder drives it. This keeps the mode count low and avoids focus-management complexity. The scene notes editor (e) is a prompted text input in the status bar, not a second cursor in the panel.
- **Binder CRUD is synchronous.** The old async `newSceneMsg` flow was replaced with a modal prompt bar. The user types a name, presses Enter, and the operation completes immediately.
- **`RevisionBackend` is trigger-agnostic.** The backend knows how to snapshot, log, diff, and restore. It does not know *when* to do those things. The caller (Ctrl+S handler, autosave timer, structural edit) decides timing.
- **Export is a core operation.** `Project.Export()` lives in `core`, not `tui`. A GUI or CLI can compile a manuscript without running the TUI.
- **Overlays own keys first.** The view-ownership priority chain is `quickNote → search → reader → history → structural views → prompt → normal`. This prevents Esc from quitting the app when an overlay is open. Overlays that need text input (quick-note, search query) block F7 to avoid key collision.
- **`.chisel.yaml` is app prefs, not a manifest.** The "filesystem is the project" rule applies to scene content. Theme, daily goal, and future preferences live in `.chisel.yaml` — a single config file that is gracefully ignored when missing. This is a deliberate carve-out, documented in CLAUDE.md.

## pane layout evolution

The original design had three pane configurations (editor-only, binder+editor, binder+editor+LLM) toggled with Ctrl+1/2/3. The current implementation uses a more flexible approach:

- **Main view:** binder + editor (always) + optional right panel (F5)
- **Structural views:** corkboard (F2), outliner (F3), or timeline (F4) replace the main view full-width. Cross-hop between them with F2/F3/F4.
- **Overlay views:** quick-note (backtick), search (Ctrl+F), history browser (Ctrl+H) overlay the current view. Reading mode (F6) goes full-screen with no chrome.
- **Sprint timer (F7):** toggles from any view (except text-input overlays); status bar shows countdown and word gain.

This is simpler than the original 3-mode design and avoids the complexity of resizing three panes with independent content types. When the LLM panel returns, it can slot in as either a structural view or a third pane in the main layout — the `viewMode` system already supports both patterns.
