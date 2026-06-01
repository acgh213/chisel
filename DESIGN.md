# ✧ chisel — design document ✧

> **Status (June 2026):** This document is the north-star vision for chisel.
> Features shipped through Phase 8 are marked **[shipped]** inline below.
> The original v1.2 full implementation lives on `archive/chisel-full` (reference only).

**Shipped through Phase 8:**
- Stabilized TUI layout (border-aware sizing, no overflow) ✓
- `core/` package with zero charmbracelet imports (GUI-ready seam) ✓
- YAML frontmatter metadata per scene (status glyphs, word count, timestamps) ✓
- Git-backed revision history — auto-snapshot, browse/diff/restore (Ctrl+H) ✓
- Corkboard (F2) and outliner (F3) structural views ✓
- Compile/export to manuscript.md + optional pandoc .docx (Ctrl+E) ✓
- Binder-side CRUD — create/rename/delete files and folders (n/N/r/d) ✓
- `chisel init` subcommand — 3 templates, interactive + non-interactive ✓
- Right panel + character viewer — passive binder-driven inspector (F5) ✓
- Character YAML frontmatter (name, role, description, tags) ✓

**Pending:** LLM assist, scene notes, richer character sheets, themes/goals, GUI.

## architecture

The architecture has evolved through eight phases. What was originally designed as a Go TUI + Python LLM backend is now a pure-Go TUI with a strict `core`/`tui` split. The LLM layer (pending) will slot back in as a `core` package, keeping the same boundary.

```
┌──────────────────────────────────────────────────────────┐
│                   chisel binary (Go)                      │
│                                                           │
│  ┌───────────────────────────────────────────────────┐   │
│  │              tui/ (Bubble Tea + Lip Gloss)         │   │
│  │  ┌────────┐ ┌────────┐ ┌───────────────────┐      │   │
│  │  │ binder │ │ editor │ │  right panel      │      │   │
│  │  │ (tree) │ │(md txt)│ │  (character view) │      │   │
│  │  └────────┘ └────────┘ └───────────────────┘      │   │
│  │  ┌──────────────────────────────────────────┐     │   │
│  │  │  structural views                        │     │   │
│  │  │  ┌──────────┐ ┌──────────┐               │     │   │
│  │  │  │corkboard │ │ outliner │               │     │   │
│  │  │  │(F2)      │ │(F3)      │               │     │   │
│  │  │  └──────────┘ └──────────┘               │     │   │
│  │  └──────────────────────────────────────────┘     │   │
│  │                         │                          │   │
│  │                    core/ types                     │   │
│  │  ┌──────────┬──────────┬──────────┬──────────┐   │   │
│  │  │ project  │  scene   │ metadata │ revision │   │   │
│  │  │ outline  │  export  │  crud    │ scaffold │   │   │
│  │  │          │          │          │ character│   │   │
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
├── scenes/               # your writing — one .md file per scene
│   ├── ch01-opening.md
│   ├── ch02-rising-action.md
│   └── notes.md
├── characters/           # character profiles — .md with YAML frontmatter
│   ├── protagonist.md
│   └── antagonist.md
├── locations/            # location descriptions (scaffolded, manual editing)
├── exports/              # compiled output
│   ├── manuscript.md
│   └── manuscript.docx   (if pandoc installed)
└── .git/                 # auto-created by go-git on first save
```

No `manifest.jsonl`. No `config.json`. No sidecar files. The filesystem and YAML frontmatter are the data model.

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
word_count: 1247
created: 2026-05-31T09:50:24-04:00
modified: 2026-05-31T09:50:24-04:00
---
# Chapter One

The train pulled in at dusk.
```

Files without frontmatter are plain markdown and open/save cleanly. The `word_count`, `created`, and `modified` fields are auto-managed — the user never edits them. Status glyphs appear in the binder: ○ draft, ◐ revised, ● done.

### [shipped] character format

Character profiles in `characters/` follow the same pattern with a different frontmatter schema:

```markdown
---
name: Elara Voss
role: Protagonist
description: A cartographer who maps the unmappable. Haunted by the cities she's erased.
tags:
  - pov
  - haunted
---
## Background

Elara grew up in the borderlands between Ha'ren and the outer rings...
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

### right panel (F5)

A passive, binder-driven inspector pane. No cursor, no focus, no key handling — it reflects whatever the binder has selected. Three modes:
- **Character detail:** when a file in `characters/` is selected, shows name, role, description, tags, and body notes
- **Cast list:** when anything else is selected, lists all characters with roles
- **Empty hint:** when no `characters/` directory exists

Toggled with F5. The layout rebalances to three panes (binder shrinks, editor stays, right panel appears on the right).

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

The editor is modeless. Standard shortcuts: Ctrl+S saves (+ snapshots), Ctrl+F finds, Ctrl+Z undoes. No vim mode. The goal is that someone who hasn't used a CLI editor in years can sit down and write.

Scenes open from the binder (Enter) or from structural views (Enter in corkboard/outliner). The editor preserves unsaved changes when switching scenes — a modified indicator (●) appears in the status bar.

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
- **Passive right panel.** The character inspector has no cursor and handles no keys. It's a pure view. The binder drives it. This keeps the mode count low and avoids focus-management complexity.
- **Binder CRUD is synchronous.** The old async `newSceneMsg` flow was replaced with a modal prompt bar. The user types a name, presses Enter, and the operation completes immediately.
- **`RevisionBackend` is trigger-agnostic.** The backend knows how to snapshot, log, diff, and restore. It does not know *when* to do those things. The caller (Ctrl+S handler, autosave timer, structural edit) decides timing.
- **Export is a core operation.** `Project.Export()` lives in `core`, not `tui`. A GUI or CLI can compile a manuscript without running the TUI.

## pane layout evolution

The original design had three pane configurations (editor-only, binder+editor, binder+editor+LLM) toggled with Ctrl+1/2/3. The current implementation uses a different approach:

- **Main view:** binder + editor (always) + optional right panel (F5)
- **Structural views:** corkboard (F2) or outliner (F3) replace the main view full-width
- **Overlay views:** history browser (Ctrl+H) overlays the main view full-width

This is simpler than the original 3-mode design and avoids the complexity of resizing three panes with independent content types. When the LLM panel returns, it can slot in as either a structural view or a third pane in the main layout — the `viewMode` system already supports both patterns.
