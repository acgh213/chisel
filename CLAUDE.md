# CLAUDE.md

This file provides guidance to Claude Code (and other AI coding assistants) when working in this repository.

## project overview

chisel is a local-first markdown writing TUI for fiction writers. It is actively growing toward Scrivener-class features with a cross-platform Go binary.

- **repo:** github.com/acgh213/chisel
- **language:** Go
- **TUI framework:** Bubble Tea + Lip Gloss + Bubbles (textarea, tree)
- **usage:** `chisel <directory>` or `chisel init`

## current architecture

```
Go binary
  ├── core/              — pure data layer (zero charmbracelet imports)
  │   ├── project.go     — Project, FileNode, BuildTree, Flatten
  │   ├── scene.go       — Scene (frontmatter + body), LoadScene, Save, CreateScene
  │   ├── metadata.go    — Metadata, splitFrontmatter, serializeFrontmatter
  │   ├── character.go   — Character, CharacterMeta, ListCharacters
  │   ├── revision.go    — RevisionBackend interface + go-git implementation
  │   ├── outline.go     — SceneInfo, FolderScenes, SortScenesForReading
  │   ├── export.go      — Project.Export (manuscript.md + optional .docx)
  │   ├── crud.go        — CreateFolder, RenameNode, DeleteNode
  │   └── scaffold.go    — ScaffoldProject, Template, Slugify (chisel init)
  └── tui/               — Bubble Tea presentation layer
      ├── model.go       — root model; layout, key dispatch, pandoc detection
      ├── binder.go      — file tree pane
      ├── editor.go      — markdown editor (bubbles/textarea over core.Scene)
      ├── history.go     — revision history browser (Ctrl+H)
      ├── corkboard.go   — index-card grid view (F2)
      ├── outliner.go    — collapsible outline view (F3)
      ├── rightpanel.go  — binder-driven inspector panel (F5); character viewer
      ├── prompt.go      — inline bottom-bar text-input for CRUD prompts
      └── styles.go      — peach color palette, all lipgloss styles
```

**Hard rule:** `core/` has zero charmbracelet imports. All `core` types are plain Go structs. A future GUI reuses `core` without touching `tui`.

## data model

- **Filesystem is the project.** A directory of `.md` files is everything.
- **YAML frontmatter** at the top of each `.md` holds metadata (scene: title, status, synopsis, draft_order, etc.; character: name, role, description, tags). Files without frontmatter are plain markdown and round-trip cleanly.
- **Character files** live in `characters/` as `.md` files with `CharacterMeta` frontmatter.
- **No manifest.** No `config.json`. No sidecar files.
- **Revision history** is git-backed (`go-git`, pure Go). Every `Ctrl+S` snapshots. `.git` is created in the project root on first save.
- **Exports** go to `<project>/exports/manuscript.md` (optional `.docx` via pandoc).

## project structure

```
chisel/
├── README.md
├── DESIGN.md          # v1.2 north-star vision; shipped features marked
├── CLAUDE.md          # this file
├── LICENSE
├── main.go            # entry: chisel <dir> | chisel init [--template T] [dir]
├── go.mod / go.sum
├── core/              # (see architecture above)
└── tui/               # (see architecture above)
```

## go dependencies

```go
import (
    "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/charmbracelet/bubbles/textarea"
    "github.com/charmbracelet/bubbles/textinput"  // binder CRUD prompts
    "github.com/go-git/go-git/v5"                 // revision history (pure Go)
    "gopkg.in/yaml.v3"                             // frontmatter
)
```

No Python backend. No LLM. No manifest. No system git dependency.

## file conventions

- Go files are lowercase: `editor.go`, `binder.go`
- Markdown files use uppercase: `DESIGN.md`, `CLAUDE.md`
- Use `filepath` package for all paths (Windows compatibility)
- `package main` at repo root; `package core` in `core/`; `package tui` in `tui/`
- New `core` files must not import `charmbracelet/*` — enforce by grep

## styling

- Peach theme only. Colors defined as `lipgloss.Color` constants in `tui/styles.go`.
- Every component references these constants — no hardcoded hex values in component code.

## keyboard shortcuts

| Key | Action |
|-----|--------|
| j/k or ↑/↓ | Navigate binder tree |
| Enter | Open file / toggle folder in binder |
| Tab | Switch focus: binder ↔ editor |
| n | New scene (prompt, relative to binder selection) |
| N | New folder |
| r | Rename selected node |
| d | Delete selected node (y/n confirm) |
| Ctrl+N | New scene (same as n, works from editor too) |
| Ctrl+S | Save + revision snapshot |
| Ctrl+H | Revision history browser |
| Ctrl+E | Compile project → `exports/manuscript.md` |
| F2 | Corkboard view |
| F3 | Outliner view |
| F5 | Toggle right inspector panel (character viewer) |
| Ctrl+Q / Esc | Quit (second press if unsaved changes) |

**In history browser:** ↑/↓ navigate, Enter diff, `r` restore, Esc close  
**In corkboard/outliner:** ↑/↓/←/→ navigate, Enter open scene, Esc back  
**Right panel (F5):** passive — reflects binder selection; shows character detail or cast list

## error handling

- *Recoverable errors*: `core` returns `error`; TUI shows it in the **status bar** and keeps running.
- *Programmer errors*: may panic in dev; never for routine I/O.
- `core` never prints or touches the UI — errors propagate up.
- `serializeFrontmatter` in `core/metadata.go` is the single serializer for all frontmatter types.

## what's built

**Phases 0–8 (all shipped):**
- Stable layout (exact terminal sizing, three-pane support)
- `core/` package — zero TUI imports, GUI-ready seam
- YAML frontmatter metadata per scene + characters
- Git-backed revision history (Ctrl+H)
- Binder CRUD: create/rename/delete files and folders inline
- Corkboard (F2) and outliner (F3) views
- Compile/export to manuscript.md (Ctrl+E), optional .docx via pandoc
- `chisel init` — scaffold new projects from templates (minimal/novel/short-stories)
- Right inspector panel (F5) — binder-driven character viewer; cast list

**Coming next:**
- Left-pane mode switching (binder / character list / location list)
- Location sheets (`locations/`)
- Timeline view (F4) — scenes on a time axis, continuity helper
- Quick-note popup — global capture hotkey, scratch file
- LLM assist (research, rewrite, generate) via local or cloud model
- Name generator
- GUI alongside the TUI (Wails or Fyne), reusing `core`

## references

- [DESIGN.md](DESIGN.md) — full v1.2 vision
- Archive branch: `archive/chisel-full` — original v1.2 code (reference only)
- Plan file: `~/.claude/plans/hi-so-this-project-zippy-russell.md`
