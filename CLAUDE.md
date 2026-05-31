# CLAUDE.md

This file provides guidance to Claude Code (and other AI coding assistants) when working in this repository.

## project overview

chisel is a local-first markdown writing TUI targeting Scrivener-class features. The binary is built and usable today; it is actively growing toward a full writing environment.

- **repo:** github.com/acgh213/chisel
- **language:** Go
- **TUI framework:** Bubble Tea + Lip Gloss + Bubbles (textarea, tree)
- **usage:** `chisel <project-directory>` — a directory of `.md` files is the project

## current architecture

```
Go binary
  ├── core/           — pure data layer (zero charmbracelet imports)
  │   ├── project.go  — Project, FileNode, BuildTree, Flatten
  │   ├── scene.go    — Scene (frontmatter + body), LoadScene, Save, CreateScene
  │   ├── metadata.go — Metadata (YAML frontmatter), parseFrontmatter, serializeScene
  │   ├── revision.go — RevisionBackend interface + go-git implementation
  │   ├── outline.go  — SceneInfo, FolderScenes, SortScenesForReading
  │   └── export.go   — Project.Export (compile manuscript.md + optional .docx)
  └── tui/            — Bubble Tea presentation layer
      ├── model.go    — root model; dispatches keys, owns layout, pandoc detection
      ├── binder.go   — file tree pane (bubbles/tree over core.FileNode)
      ├── editor.go   — markdown editor (bubbles/textarea over core.Scene)
      ├── history.go  — revision history browser (Ctrl+H)
      ├── corkboard.go — index-card grid view (F2)
      ├── outliner.go  — collapsible outline view (F3)
      └── styles.go   — peach color palette, shared lipgloss styles
```

**Hard rule:** `core/` has zero charmbracelet imports. All `core` types are plain Go structs. A future GUI reuses `core` without touching `tui`.

## data model

- **Filesystem is the project.** A directory of `.md` files is everything.
- **YAML frontmatter** at the top of each `.md` is the metadata (title, status, synopsis, draft_order, word_target, pov, tags, word_count, created, modified). Files without frontmatter are plain markdown and still open/save cleanly.
- **No manifest.** No `config.json`. No sidecar files. The `.md` file is the single source of truth.
- **Revision history** is git-backed (`go-git`, pure Go). Every `Ctrl+S` triggers an automatic snapshot. The `.git` directory is created inside the project root on first save.
- **Exports** go to `<project>/exports/manuscript.md` (and optionally `.docx` if pandoc is installed).

## project structure

```
chisel/
├── README.md
├── DESIGN.md          # v1.2 north-star vision; sections marked shipped/pending
├── CLAUDE.md          # this file
├── LICENSE
├── main.go            # entry point: chisel <directory>
├── go.mod / go.sum
├── core/
│   ├── project.go
│   ├── project_test.go
│   ├── scene.go
│   ├── scene_test.go
│   ├── metadata.go
│   ├── metadata_test.go
│   ├── revision.go
│   ├── revision_test.go
│   ├── outline.go
│   ├── outline_test.go
│   ├── export.go
│   └── export_test.go
└── tui/
    ├── model.go
    ├── model_test.go
    ├── binder.go
    ├── binder_test.go
    ├── editor.go
    ├── history.go
    ├── history_test.go
    ├── corkboard.go
    ├── outliner.go
    ├── views_test.go
    └── styles.go
```

## go dependencies

```go
import (
    "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/charmbracelet/bubbles/textarea"
    "github.com/charmbracelet/bubbles/tree"
    "github.com/go-git/go-git/v5"   // revision history (pure Go, no system git)
    "gopkg.in/yaml.v3"               // frontmatter
)
```

No Python backend. No LLM. No manifest. No system git dependency.

## file conventions

- Go files are lowercase: `editor.go`, `binder.go`
- Markdown files use uppercase: `DESIGN.md`, `CLAUDE.md`
- Use `filepath` package for all paths (Windows compatibility)
- `package main` in `main.go` at repo root; `package core` in `core/`; `package tui` in `tui/`
- New `core` files must not import `charmbracelet/*` — enforce by grep

## styling

- Peach theme only. Colors defined as `lipgloss.Color` constants in `tui/styles.go`.
- Every component references these constants — no hardcoded hex values in component code.

## keyboard shortcuts

| Key | Action |
|-----|--------|
| j/k or ↑/↓ | Navigate binder tree |
| Enter | Open file / toggle folder in binder |
| Space | Toggle folder in binder |
| Tab | Switch focus between binder and editor |
| Ctrl+N | New scene (prompt for name) |
| Ctrl+S | Save current file + create revision snapshot |
| Ctrl+H | Open revision history browser |
| Ctrl+E | Compile project to `exports/manuscript.md` |
| F2 | Open corkboard view |
| F3 | Open outliner view |
| Ctrl+Q / Esc | Quit (second press confirms if unsaved changes) |

**In history browser:** ↑/↓ navigate, Enter show diff, `r` restore, Esc close  
**In corkboard/outliner:** ↑/↓/←/→ navigate, Enter open scene, Esc return to binder+editor

## error handling

- *Recoverable errors* (YAML parse failure, save permission denied, git issues, pandoc failure): `core` returns a normal `error`; the TUI shows it in the **status bar** and keeps running.
- *Programmer errors* (impossible states): may panic in dev; never for routine I/O.
- `core` never prints or touches the UI — errors propagate up; the TUI decides how to show them.

## what's built vs what's coming

**Built (Phases 0–5):**
- Stabilized TUI layout (exact terminal sizing, no overflow)
- core package with zero TUI dependencies (GUI-ready seam)
- YAML frontmatter metadata per scene
- Git-backed revision history (Ctrl+H)
- Corkboard (F2) and outliner (F3) views
- Compile / export to manuscript.md (Ctrl+E), optional .docx via pandoc

**Coming next:**
- Binder-side CRUD (create/rename/delete files and folders relative to selection; folder creation)
- `chisel init` command to scaffold a new project
- LLM assist (rewrite, generate, summarize, research) via local or cloud model
- Character sheets and per-scene notes
- Themes, session word count, sprint timer, typewriter / focus modes
- GUI alongside the TUI (Wails or Fyne), reusing `core`

## references

- [DESIGN.md](DESIGN.md) — full v1.2 vision; features marked shipped/pending
- Archive branch: `archive/chisel-full` — original v1.2 code (reference only)
- Plan file: `~/.claude/plans/hi-so-this-project-zippy-russell.md`
