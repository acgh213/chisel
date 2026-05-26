# CLAUDE.md

this file provides guidance to Claude Code (and other AI coding assistants) when working in this repository.

## project overview

chisel is a local-first markdown writing TUI. v0.1 is a brutal MVP: binder tree + markdown editor + save. That's it.

- **repo:** github.com/acgh213/chisel
- **language:** Go
- **TUI framework:** Bubble Tea + Lip Gloss + Bubbles (textarea, tree)
- **v0.1 scope:** open directory, navigate file tree, edit .md files, Ctrl+S saves. Single binary.

## architecture (v0.1)

```
Go binary
  ├── binder tree (filesystem navigation — folders + .md files)
  └── markdown editor (textarea)
```

No Python backend. No LLM. No manifest. No git. No themes. Filesystem IS the data model.

## key design decisions (v0.1)

1. **Filesystem is the manifest.** A project is a directory. Folders and .md files are the structure. No manifest.jsonl, no config.json.
2. **One scene per .md file.** Each .md file is a scene. The binder tree shows the directory structure.
3. **Modeless editor.** Standard shortcuts (Ctrl+S, Ctrl+Z, Ctrl+F). No vim mode in v0.1.
4. **Single binary.** `go build` produces one binary. No runtime dependencies.
5. **Peach theme only.** One color palette, hardcoded. No theme engine.

## project structure (v0.1)

```
chisel/
├── README.md
├── DESIGN.md              # full architecture vision (v1.2 reference)
├── CLAUDE.md              # this file
├── LICENSE
├── main.go                # entry point: chisel <directory>
├── go.mod / go.sum
└── tui/
    ├── model.go           # root bubbletea model (composes binder + editor)
    ├── binder.go          # file tree (bubbles/tree)
    ├── editor.go          # markdown editor (bubbles/textarea)
    └── styles.go          # color palette, base styles
```

## go dependencies (v0.1)

```go
import (
    "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/charmbracelet/bubbles/textarea"
    "github.com/charmbracelet/bubbles/tree"
)
```

No go-git in v0.1. No Python deps.

## file conventions

- Go files are lowercase: `editor.go`, `binder.go`
- All markdown is lowercase with hyphens: `DESIGN.md`, `CLAUDE.md`
- Use `filepath` package for all paths (Windows compatibility)
- package name is `tui` for all files in tui/

## styling

- Peach theme only. Colors defined as lipgloss.Color constants in `tui/styles.go`.
- Every component references these constants — no hardcoded hex values in component code.

## v0.1 keyboard shortcuts

| Key | Action |
|-----|--------|
| j/k or ↑/↓ | Navigate binder tree |
| Enter | Open file / toggle folder in binder |
| Space | Toggle folder in binder |
| Tab | Switch focus between binder and editor |
| Ctrl+N | New scene (prompt for name) |
| Ctrl+S | Save current file |
| Ctrl+Q / Esc | Quit |

## what's NOT in v0.1

- No LLM integration (Python backend, NDJSON protocol)
- No revision history (git / jj)
- No manifest.jsonl or config.json
- No export
- No themes, pomodoro, character sheets, corkboard
- No research, mirror, style analysis

## references

- [DESIGN.md](DESIGN.md) — full architecture vision (v1.2 reference, not v0.1)
- Archive branch: `archive/chisel-full` — full v1.2 code for design reference
