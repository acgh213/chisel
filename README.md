# ✧ chisel ✧

A local-first markdown writing TUI. Binder tree on the left, editor in the center, world panel on the right. Built for people who want Scrivener's structure without leaving the terminal — and without lock-in.

**Your writing lives in plain `.md` files.** Folders are your binder. YAML frontmatter is your metadata. If chisel disappears tomorrow, your project is still a directory of markdown. Nothing to export, nothing to recover.

## quick start

```bash
# build
go build -o chisel .

# create a project
./chisel init
./chisel init --template novel my-novel

# open a project
./chisel my-project/
./chisel .
```

Templates: `minimal` (README only), `novel` (scenes/ + characters/ + locations/ + sample chapters), `short-stories` (single story file).

## what it does

- **Binder** — navigate your project as a file tree. Create, rename, delete files and folders directly (`n`/`N`/`r`/`d`). Folders stay expanded across operations.
- **Editor** — write in a modeless markdown editor. Standard shortcuts: Ctrl+S saves (+ session word accumulation), Ctrl+F finds, Ctrl+Z undoes. Zero learning curve if you've used a text editor.
- **Metadata** — each scene carries YAML frontmatter (title, status, synopsis, tags, draft order, word target, POV, timeline date, notes). Word count and timestamps are automatic. Files without frontmatter stay plain — no forced metadata.
- **Revision history** — every Ctrl+S creates an automatic git snapshot. Browse, diff, and restore any saved version with Ctrl+H. Pure Go — no system git required.
- **Structural views** — three ways to see your project: corkboard (index-card grid, F2), outliner (collapsible outline, F3), and timeline (date-sorted scene list, F4). Cross-hop between them with F2/F3/F4.
- **World panel** — right panel shows character profiles and location sheets. Passive, binder-driven. Toggle with F5. W switches between world index and scene notes. e edits notes inline.
- **Scene notes** — per-scene scratch space in the `notes` frontmatter field. Edit inline without leaving the binder.
- **Quick-note** — backtick opens a floating popup from any state. Jot a thought, Enter saves it to `notes/scratch.md`.
- **Full-text search** — Ctrl+F searches all scenes (body text). Browse results, Enter opens the match.
- **Reading mode** — F6 opens the current scene full-screen, word-wrapped. No chrome. Just your words.
- **Themes** — four dark themes (peach, forest, ocean, midnight). F8 cycles, persists to `.chisel.yaml`.
- **Sprint timer** — F7 starts a 25-minute pomodoro. Status bar shows countdown and words written. Works from any view.
- **Session word count** — accumulated across saves and file switches. Shown as `+N today` (or `+N/G today` with a daily goal).
- **Export** — compile your project to `exports/manuscript.md` (Ctrl+E). Optional `.docx` via pandoc if installed.
- **Init** — scaffold new projects with templates. Interactive or non-interactive.

## keybindings

### everywhere

| key | action |
|-----|--------|
| j/k or ↑/↓ | Navigate |
| Enter | Open file / toggle folder |
| Space | Toggle folder |
| Tab | Switch binder ↔ editor |
| F2 | Corkboard view |
| F3 | Outliner view |
| F4 | Timeline view |
| F5 | Toggle world panel |
| F6 | Reading mode (scene must be open) |
| F7 | Start/stop sprint timer |
| F8 | Cycle theme |
| Ctrl+F | Full-text search |
| ` (backtick) | Quick-note popup |
| ? | Help / full keymap (non-text-input states) |
| Ctrl+Q / Esc | Quit (second press confirms if unsaved) |

### binder (focused)

| key | action |
|-----|--------|
| n | New scene (prompt for name) |
| N | New folder |
| r | Rename selected |
| d | Delete selected (y=confirm) |
| W | Toggle world index / scene notes (right panel open) |
| e | Edit scene note inline (note mode) |
| Ctrl+N | New scene (same as `n`) |

### editor (focused)

| key | action |
|-----|--------|
| Ctrl+S | Save + snapshot |
| Ctrl+H | Revision history browser |
| Ctrl+E | Compile to manuscript.md |
| Ctrl+N | New scene |

### structural views (corkboard / outliner / timeline)

| key | action |
|-----|--------|
| ←/→/↑/↓ | Navigate cards / rows |
| Enter | Open selected scene |
| Esc / F1 | Return to binder+editor |
| F2 / F3 / F4 | Switch between structural views |

### history browser

| key | action |
|-----|--------|
| ↑/↓ | Navigate snapshots |
| Enter | Show diff for selected |
| r | Restore selected revision |
| Esc / Ctrl+H | Close browser |

### search overlay

| key | action |
|-----|--------|
| (type query) | Enter search terms |
| Enter | Run search |
| ↑/↓ | Navigate results |
| Enter | Open selected scene |
| Esc | Refine query / close |

### reading mode

| key | action |
|-----|--------|
| ↑/↓ or j/k | Scroll |
| Ctrl+D / Ctrl+U | Jump half-page |
| F6 / Esc | Exit reading mode |

## architecture

```
Go binary
  ├── main.go           — entry point + init subcommand
  ├── core/             — pure data layer (zero TUI imports, GUI-ready)
  │   ├── project.go    — Project, FileNode, BuildTree
  │   ├── scene.go      — Scene, Load/Save/Create, WordCount
  │   ├── metadata.go   — YAML frontmatter, parse/serialize (shared)
  │   ├── revision.go   — RevisionBackend + go-git implementation
  │   ├── outline.go    — SceneInfo, FolderScenes, sort
  │   ├── export.go     — Compile manuscript.md + pandoc .docx
  │   ├── crud.go       — CreateFolder, RenameNode, DeleteNode
  │   ├── scaffold.go   — Project templates (init)
  │   ├── character.go  — Character, CharacterMeta, ListCharacters
  │   ├── location.go   — Location, LocationMeta, ListLocations
  │   ├── timeline.go   — TimelineEntry, BuildTimeline (date-sorted)
  │   ├── notes.go      — AppendScratch (quick-note journal)
  │   ├── search.go     — SearchResult, SearchScenes (full-text)
  │   └── config.go     — ChiselConfig, Load/Save (.chisel.yaml)
  └── tui/              — Bubble Tea presentation
      ├── model.go      — root model, layout, key dispatch, sprint timer
      ├── binder.go     — file tree pane
      ├── editor.go     — markdown editor
      ├── history.go    — revision browser
      ├── corkboard.go  — index-card grid view (F2)
      ├── outliner.go   — collapsible outline view (F3)
      ├── timeline.go   — date-sorted scene list (F4)
      ├── quicknote.go  — floating quick-note popup (backtick)
      ├── search.go     — full-text search overlay (Ctrl+F)
      ├── reader.go     — full-screen reading mode (F6)
      ├── rightpanel.go — world panel: characters + locations (F5)
      ├── prompt.go     — inline prompt bar for CRUD
      └── styles.go     — 4 dark themes, Color* vars + rebuildStyles()
```

Single binary. No Python backend. No LLM. No manifest files. No system git dependency. Filesystem is the data model.

## on-disk format

A chisel project is a directory of `.md` files:

```
my-novel/
├── README.md
├── .chisel.yaml          # app prefs (theme, daily goal) — optional
├── scenes/
│   ├── ch01-opening.md
│   └── ch02-rising-action.md
├── characters/
│   ├── protagonist.md
│   └── antagonist.md
├── locations/
│   ├── the-docks.md
│   └── borderlands.md
├── notes/
│   └── scratch.md         # quick-note catch-all (auto-created)
└── exports/
    ├── manuscript.md
    └── manuscript.docx    (if pandoc installed)
```

Scene files carry optional YAML frontmatter:

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
notes: The rain motif should echo the prologue's storm.
word_count: 1247
created: 2026-05-31T09:50:24-04:00
modified: 2026-05-31T09:50:24-04:00
---
# Chapter One

The train pulled in at dusk.
```

Files without frontmatter are plain markdown and work fine — chisel never forces metadata on you.

## building

```bash
go build -o chisel .
```

Requires Go 1.22+. Optional: [pandoc](https://pandoc.org/) for `.docx` export.

## philosophy

Plain markdown files in folders. Your writing, your filesystem, your git history. Everything chisel touches is transparent — open your project in VS Code, Obsidian, or any text editor and it's exactly what you'd expect. No lock-in, no proprietary format, no migration path needed.

## license

MIT — see [LICENSE](LICENSE).
