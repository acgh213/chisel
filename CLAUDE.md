# CLAUDE.md

this file provides guidance to Claude Code (and other AI coding assistants) when working in this repository.

## project overview

chisel is a local-first, markdown-native writing tool with LLM augmentation. it's a TUI (terminal user interface) built with Go + Bubble Tea, with a Python backend for LLM calls. think "scrivener in a terminal, with a local AI that helps you think."

- **repo:** github.com/acgh213/chisel
- **language:** Go (TUI) + Python (LLM backend)
- **TUI framework:** [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lip Gloss](https://github.com/charmbracelet/lipgloss) + [Bubbles](https://github.com/charmbracelet/bubbles)

## architecture

```
Go TUI (bubbletea)
  ├── binder tree (scene navigation)
  ├── markdown editor (writing)
  └── LLM panel (responses, research, analysis)
        │
        │ NDJSON over stdin/stdout (subprocess)
        │
Python backend (chisel.py)
  ├── LLM calls (openai-compatible API)
  ├── research gathering
  └── stylistic analysis (mirror model)
```

the Go TUI handles everything the user touches. the Python backend handles everything the LLM touches. they communicate via NDJSON — the TUI spawns `chisel.py` as a subprocess and pipes requests/responses.

## key design decisions

1. **one scene per `.md` file.** no multi-scene files with delimiters. the binder tree provides organization.
2. **manifest-driven.** scene metadata (title, status, word count, tags, draft order) lives in `manifest.jsonl`, not in markdown frontmatter. the `.md` files are pure prose.
3. **research auto-tags to current scene.** when the LLM researches a topic, the resulting note is automatically linked to the active scene. toggle in settings to disable.
4. **revision history via git.** every save auto-commits. jj (Jujutsu) planned as optional backend in v1.2.
5. **modeless editor by default.** standard shortcuts (Ctrl+S, Ctrl+Z, Ctrl+F). vim bindings as opt-in toggle.
6. **LLM provider-flexible.** any OpenAI-compatible endpoint works (LM Studio, llama.cpp, OpenAI, Anthropic). configured in `config.json`.
7. **two model slots.** `llm` for general tasks (rewrite, generate, research, ask). `mirror` for stylistic analysis (fine-tuned model on the user's own writing).

## project structure (planned)

```
chisel/
├── README.md
├── DESIGN.md
├── GOALS.md
├── PLAN.md
├── CHANGELOG.md
├── CLAUDE.md              ← this file
├── chisel.py              # Python backend (LLM calls, research, analysis)
├── tui/                   # Go TUI
│   ├── main.go
│   ├── model.go           # root bubbletea model + state
│   ├── binder.go          # scene tree navigation
│   ├── editor.go          # markdown editor
│   ├── manifest.go        # JSONL read/write
│   ├── config.go          # config.json loading
│   ├── llm.go             # subprocess manager for chisel.py
│   ├── history.go         # revision history (git-backed)
│   ├── styles.go          # color tokens and common styles
│   └── themes/            # theme definitions
│       ├── peach.go
│       ├── dark.go
│       ├── light.go
│       ├── forest.go
│       └── ocean.go
└── scenes/                # user's writing (created by `chisel new`)
```

## file conventions

- **all markdown is lowercase with hyphens.** README.md, not ReadMe.md. design-doc.md, not DesignDoc.md.
- **Go files are lowercase.** `editor.go`, not `Editor.go`.
- **JSONL manifests are append-only during normal use.** reorder events rewrite the entire file.
- **config files are JSON.** `config.json`, not `.yaml` or `.toml`.

## development notes

### go dependencies

```go
import (
    "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/charmbracelet/bubbles/textarea"
    "github.com/charmbracelet/bubbles/table"
    "github.com/charmbracelet/bubbles/tree"
)
```

### git for revision history

use `go-git` (pure Go library) for auto-commit on save. no shelling out to `git` CLI:

```go
import "github.com/go-git/go-git"
```

the revision API should be abstract so jj can slot in later:

```go
type RevisionBackend interface {
    Save(path string, message string) error
    Log(path string) ([]Revision, error)
    Diff(path string, rev1, rev2 string) (string, error)
    Restore(path string, rev string) error
}
```

### ndjson protocol (go ↔ python)

all communication between the TUI and Python backend uses NDJSON over stdin/stdout:

**request:**
```json
{"op": "rewrite", "text": "selected text here", "context": "previous paragraph for context"}
```

**response:**
```json
{"op": "rewrite", "result": "suggested alternative text", "tokens": 127, "status": "ok"}
```

**streaming:** for long operations, responses may be chunked:
```json
{"op": "ask", "result": "partial response...", "status": "streaming"}
{"op": "ask", "result": "...rest of response", "status": "ok"}
```

### windows compatibility

- all file paths use `filepath` package, never hardcoded separators
- `os.Executable()` for locating `chisel.py` relative to the binary
- `shutil.move()` in Python for cross-drive file moves
- reserved Windows characters stripped from folder names: `\ / : * ? " < > |`

### styling

- color palette defined in `styles.go` as lipgloss.Color constants
- every component references these tokens — no hardcoded hex values in component code
- theme switching means swapping the token values, not rewriting components
- peach theme is the default

## phase implementation order

follow PLAN.md. the phases are ordered by dependency:

0. scaffolding (project creation, config, manifest I/O)
1. binder + editor (writing experience)
2. revision history (go-git auto-commit, history browser, diff/restore)
3. llm integration (Python backend, rewrite/generate/summarize/ask)
4. mirror + research (style analysis, research gathering)
5. export + polish (manuscript export, themes, corkboard)
6. character + scene notes (character sheets, scene notes)
7. jj backend (jj revision history)

do not skip ahead. each phase depends on the one before it.

## references

- [DESIGN.md](DESIGN.md) — architecture, data model, pane layouts, llm integration, decisions
- [GOALS.md](GOALS.md) — short-term through long-term feature roadmap, non-goals
- [PLAN.md](PLAN.md) — phased implementation breakdown with tasks
- [CHANGELOG.md](CHANGELOG.md) — version history
