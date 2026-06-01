# CLAUDE.md

This file provides guidance to Claude Code (and other AI coding assistants) when working in this repository.

## project overview

chisel is a local-first markdown writing TUI targeting Scrivener-class features. The binary is built and usable today; it is actively growing toward a full writing environment.

- **repo:** github.com/acgh213/chisel
- **language:** Go
- **TUI framework:** Bubble Tea + Lip Gloss + Bubbles (textarea, tree, textinput)
- **usage:** `chisel <project-directory>` opens a directory as a writing project; `chisel init` scaffolds a new one

## current architecture

```
Go binary (chisel)
  ├── core/              — pure data layer (zero charmbracelet imports)
  │   ├── project.go     — Project, FileNode, BuildTree, Flatten
  │   ├── scene.go       — Scene (frontmatter + body), LoadScene, Save, CreateScene, ParseScene
  │   ├── metadata.go    — Metadata, splitFrontmatter, serializeFrontmatter (shared), serializeScene
  │   ├── revision.go    — RevisionBackend interface + GitBackend (go-git, pure Go)
  │   ├── outline.go     — SceneInfo, FolderScenes, SortScenesForReading, ReadSceneInfo
  │   ├── export.go      — Project.Export (compile manuscript.md + optional pandoc .docx)
  │   ├── crud.go        — CreateFolder, RenameNode (auto .md ext), DeleteNode (recursive)
  │   ├── scaffold.go    — ScaffoldProject, Slugify, ParseTemplate (3 templates)
  │   └── character.go   — Character, CharacterMeta, LoadCharacter, ListCharacters
  └── tui/               — Bubble Tea presentation layer
      ├── model.go       — root model; dispatches keys, owns layout, pandoc detection
      ├── binder.go      — file tree pane (bubbles/tree over core.FileNode)
      ├── editor.go      — markdown editor (bubbles/textarea over core.Scene)
      ├── history.go     — revision history browser (Ctrl+H)
      ├── corkboard.go    — index-card grid view (F2)
      ├── outliner.go     — collapsible outline view (F3)
      ├── rightpanel.go   — character inspector, binder-driven (F5)
      ├── prompt.go      — inline prompt bar for binder CRUD
      └── styles.go      — peach color palette, shared lipgloss styles
```

**Hard rule:** `core/` has zero charmbracelet imports. All `core` types are plain Go structs — `go list -deps ./core` returns stdlib + `yaml.v3` + `go-git` only. A future GUI reuses `core` without touching `tui`.

## data model

- **Filesystem is the project.** A directory of `.md` files is everything. No manifest, no config.json, no sidecar files.
- **YAML frontmatter** at the top of each `.md` is the metadata (title, status, synopsis, tags, draft_order, word_target, pov, word_count, created, modified). Files without frontmatter are plain markdown and still open/save cleanly.
- **Characters** live in `characters/` as `.md` files with their own frontmatter schema (name, role, description, tags). `ListCharacters` returns nil on missing dir (not error).
- **Revision history** is git-backed (`go-git`, pure Go). Every `Ctrl+S` triggers an automatic snapshot. The `.git` directory is created inside the project root on first save. `RevisionBackend` interface allows future jj swap.
- **Exports** go to `<project>/exports/manuscript.md` (and optionally `.docx` if pandoc is installed). Scenes are ordered by `draft_order` then filename. The `exports/` subdirectory is excluded from re-export.
- **Scene ordering:** scenes with an explicit `draft_order` sort first in ascending order, then the rest alphabetically. `SortScenesForReading` is the single authority.

## project structure

```
chisel/
├── README.md
├── DESIGN.md          # v1.2 north-star vision; features marked shipped/pending
├── CLAUDE.md          # this file
├── LICENSE
├── main.go            # entry point: subcommand dispatch (init vs TUI open)
├── go.mod / go.sum
├── core/
│   ├── project.go / project_test.go
│   ├── scene.go / scene_test.go
│   ├── metadata.go / metadata_test.go
│   ├── revision.go / revision_test.go
│   ├── outline.go / outline_test.go
│   ├── export.go / export_test.go
│   ├── crud.go / crud_test.go
│   ├── scaffold.go / scaffold_test.go
│   └── character.go / character_test.go
└── tui/
    ├── model.go / model_test.go
    ├── binder.go / binder_test.go
    ├── editor.go
    ├── history.go / history_test.go
    ├── corkboard.go
    ├── outliner.go
    ├── rightpanel.go
    ├── prompt.go
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
    "github.com/charmbracelet/bubbles/textinput"  // prompt bar
    "github.com/go-git/go-git/v5"                 // revision history (pure Go, no system git)
    "gopkg.in/yaml.v3"                             // frontmatter
)
```

No Python backend. No LLM. No manifest files. No system git dependency. No `os/exec` for git — `go-git` does everything in-process. `os/exec` is used only for optional pandoc (compile export).

## file conventions

- Go files are lowercase: `editor.go`, `binder.go`, `rightpanel.go`
- Markdown files use uppercase: `DESIGN.md`, `CLAUDE.md`, `README.md`
- Use `filepath` package for all paths (Windows compatibility)
- `package main` in `main.go` at repo root; `package core` in `core/`; `package tui` in `tui/`
- New `core` files must not import `charmbracelet/*` — verify with `go list -deps ./core | grep charmbracelet`
- Tests live alongside source: `*_test.go` in the same package directories

## styling

- Peach theme only. Colors defined as `lipgloss.Color` constants in `tui/styles.go`.
- Every component references these constants — no hardcoded hex values in component code.
- Key style vars: `StatusBarStyle`, `PromptBarStyle`, `HistoryStyle`, `RightPanelStyle`, `CardStyle`, `CardSelectedStyle`, `ViewHeaderStyle`, `DiffAddStyle`/`DiffDelStyle`/`DiffMetaStyle`, `MetTargetStyle`

## keyboard shortcuts

| Key | Context | Action |
|-----|---------|--------|
| j/k or ↑/↓ | Binder | Navigate tree |
| Enter | Binder | Open file / toggle folder |
| Space | Binder | Toggle folder |
| Tab | Any | Switch binder ↔ editor |
| Ctrl+S | Editor | Save current file + create revision snapshot |
| Ctrl+H | Editor | Open revision history browser |
| Ctrl+E | Editor | Compile project to exports/manuscript.md |
| Ctrl+N | Any | New scene (prompt for name) |
| n | Binder | New scene (same as Ctrl+N) |
| N | Binder | New folder |
| r | Binder | Rename selected node |
| d | Binder | Delete selected (y=confirm) |
| F2 | Any | Open corkboard view |
| F3 | Any | Open outliner view |
| F5 | Any | Toggle right panel (character inspector) |
| Ctrl+Q / Esc | Any | Quit (second press confirms if unsaved) |

**In history browser:** ↑/↓ navigate snapshots, Enter show diff, `r` restore, Esc close
**In corkboard/outliner:** ←/→/↑/↓ navigate, Enter open scene, Esc/F1 return to main, F2/F3 switch views
**In prompt bar:** type name then Enter to confirm, Esc to cancel (delete: y=confirm, any other key cancels)

## design patterns

### view ownership

When a sub-view is open, it owns all keys — the root `Update()` checks in priority order: history → structural views (corkboard/outliner) → prompt → normal dispatch. This avoids key collision bugs where Esc quits the app instead of closing the overlay.

### action-return pattern

Sub-views that can't act on the root model return an action enum:
- `historyModel.update()` returns `historyAction` (none/close/restore)
- `corkboardModel.update()` and `outlinerModel.update()` return `viewAction` (none/close/open)

The root model applies these actions — the sub-view never touches the root's state directly.

### prompt bar

`binderPrompt` is a self-contained struct that occupies the status-bar row during CRUD. It has its own `update()` and `view()` methods. Built as a reusable frame — future quick-note and right-panel inputs plug into the same infrastructure.

### right panel (passive)

`rightPanelModel` has no cursor, no focus, no key handling. It's purely reactive — `SyncToSelection()` is called on every binder navigation and after every CRUD refresh. Three display modes: character detail (when a file in `characters/` is selected), cast list (any other selection), empty hint (no characters yet).

### binder CRUD

`n`/`N`/`r`/`d` fire only when binder is focused. When editor is focused, those keys forward to the textarea (insert literal characters). This avoids mode-switching confusion. The prompt early-returns before esc/quit handling so Esc cancels the prompt instead of quitting.

### revision backend

`RevisionBackend` is a thin interface: `Snapshot`, `Log`, `Diff`, `Restore`. The backend is trigger-agnostic — `Snapshot` means "snapshot now," and *when* to call it is the caller's decision (Ctrl+S now, autosave later). `GitBackend` inits the repo lazily (on first save, not startup) and treats `ErrEmptyCommit` as a no-op.

## error handling

- *Recoverable errors* (YAML parse failure, save permission denied, git issues, pandoc failure): `core` returns a normal `error`; the TUI shows it in the **status bar** and keeps running.
- *Programmer errors* (impossible states): may panic in dev; never for routine I/O.
- `core` never prints or touches the UI — errors propagate up; the TUI decides how to show them.
- Frontmatter parse failures degrade gracefully to body-only — a bad header never fails the load.
- `ListCharacters` and `ReadSceneInfo` are best-effort: unreadable files are skipped, not errored.

## test conventions

- `core` tests use `t.TempDir()` for filesystem isolation
- `tui` tests drive the full Bubble Tea loop with simulated keys — no mocking of `Update()`
- Load-back assertions verify that seeded scenes survive the full save→parse round-trip
- Layout tests assert pane widths sum to terminal width across multiple sizes
- Key test files: `core/revision_test.go` (full snapshot→log→diff→restore), `tui/views_test.go` (corkboard/outliner flow + fit-terminal), `tui/history_test.go` (full restore flow), `core/scaffold_test.go` (load-back assertions)

## what's built (Phases 0–8)

- **Phase 0:** Stabilized TUI layout — border-aware sizing, no overflow, blink storm fix, quit guard
- **Phase 1:** `core/` package extraction — zero charmbracelet imports, GUI-ready seam
- **Phase 2:** YAML frontmatter metadata per scene — status glyphs, word count, timestamps
- **Phase 3:** Git-backed revision history — auto-snapshot on save, browse/diff/restore (Ctrl+H)
- **Phase 4:** Corkboard (F2) and outliner (F3) structural views
- **Phase 5:** Compile/export to manuscript.md, optional .docx via pandoc (Ctrl+E)
- **Phase 6:** Binder-side CRUD — create/rename/delete files and folders (n/N/r/d)
- **Phase 7:** `chisel init` subcommand — 3 project templates (minimal/novel/short-stories), interactive + non-interactive
- **Phase 8:** Right panel + character viewer — passive binder-driven inspector, character YAML frontmatter, cast list (F5)

## what's coming

- LLM assist (rewrite, generate, summarize, research) via local or cloud model
- Per-scene notes and richer character sheets (arc, relationships)
- Themes (dark, light, forest, ocean), session word count, pomodoro sprint timer
- Typewriter / focus mode, reading mode
- GUI alongside the TUI (Wails or Fyne), reusing `core`

## references

- [DESIGN.md](DESIGN.md) — full v1.2 vision; features marked shipped/pending
- Archive branch: `archive/chisel-full` — original v1.2 code (reference only)
- Plan file: `~/.claude/plans/hi-so-this-project-zippy-russell.md`
