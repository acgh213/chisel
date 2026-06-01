# ✧ chisel ✧

A local-first markdown writing TUI. Binder tree on the left, editor in the center, inspector on the right. Built for people who want Scrivener's structure without leaving the terminal — and without lock-in.

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
- **Editor** — write in a modeless markdown editor. Standard shortcuts: Ctrl+S saves, Ctrl+F finds, Ctrl+Z undoes. Zero learning curve if you've used a text editor.
- **Metadata** — each scene carries YAML frontmatter (title, status, synopsis, tags, draft order, word target, POV). Word count and timestamps are automatic. Files without frontmatter stay plain — no forced metadata.
- **Revision history** — every Ctrl+S creates an automatic git snapshot. Browse, diff, and restore any saved version with Ctrl+H. Pure Go — no system git required.
- **Corkboard** — index-card grid view of scenes in a folder. See status, synopsis, and word-count progress at a glance (F2).
- **Outliner** — collapsible project-wide outline with status glyphs and word-count columns (F3).
- **Character viewer** — right panel shows character details or cast list. Passive, binder-driven. Toggle with F5.
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
| Ctrl+Q / Esc | Quit (second press confirms if unsaved) |

### binder (focused)

| key | action |
|-----|--------|
| n | New scene (prompt for name) |
| N | New folder |
| r | Rename selected |
| d | Delete selected (y=confirm) |
| F2 | Corkboard view |
| F3 | Outliner view |
| F5 | Toggle right panel |
| Ctrl+N | New scene (same as `n`) |

### editor (focused)

| key | action |
|-----|--------|
| Ctrl+S | Save + snapshot |
| Ctrl+H | Revision history browser |
| Ctrl+E | Compile to manuscript.md |
| Ctrl+N | New scene |
| F2 | Corkboard view |
| F3 | Outliner view |
| F5 | Toggle right panel |

### history browser

| key | action |
|-----|--------|
| ↑/↓ | Navigate snapshots |
| Enter | Show diff for selected |
| r | Restore selected revision |
| Esc / Ctrl+H | Close browser |

### corkboard / outliner

| key | action |
|-----|--------|
| ←/→/↑/↓ | Navigate cards / rows |
| Enter | Open selected scene |
| Esc / F1 | Return to binder+editor |
| F2 | Switch to corkboard |
| F3 | Switch to outliner |

## architecture

```
Go binary
  ├── main.go           — entry point + init subcommand
  ├── core/             — pure data layer (zero TUI imports, GUI-ready)
  │   ├── project.go    — Project, FileNode, BuildTree
  │   ├── scene.go      — Scene, Load/Save/Create, WordCount
  │   ├── metadata.go   — YAML frontmatter, parse/serialize
  │   ├── revision.go   — RevisionBackend + go-git implementation
  │   ├── outline.go    — SceneInfo, FolderScenes, sort
  │   ├── export.go     — Compile manuscript.md + pandoc .docx
  │   ├── crud.go       — CreateFolder, RenameNode, DeleteNode
  │   ├── scaffold.go   — Project templates (init)
  │   └── character.go  — Character, CharacterMeta, ListCharacters
  └── tui/              — Bubble Tea presentation
      ├── model.go      — root model, layout, key dispatch
      ├── binder.go     — file tree pane
      ├── editor.go     — markdown editor
      ├── history.go    — revision browser
      ├── corkboard.go   — index-card grid view
      ├── outliner.go    — collapsible outline view
      ├── rightpanel.go  — character inspector (F5)
      ├── prompt.go     — inline prompt bar for CRUD
      └── styles.go     — peach color palette
```

Single binary. No Python backend. No LLM. No manifest files. No system git dependency. Filesystem is the data model.

## on-disk format

A chisel project is a directory of `.md` files:

```
my-novel/
├── README.md
├── scenes/
│   ├── ch01-opening.md
│   └── ch02-rising-action.md
├── characters/
│   ├── protagonist.md
│   └── antagonist.md
├── locations/
└── exports/
    ├── manuscript.md
    └── manuscript.docx   (if pandoc installed)
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
