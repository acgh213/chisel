# ✧ chisel — implementation plan ✧

## all phases complete ✅

Every phase has been implemented, tested, and verified. See [CHANGELOG.md](CHANGELOG.md) for the full version history.

| phase | version | what ships | status |
|-------|---------|-----------|--------|
| 0 — scaffolding | v0.0.1 | project creation, config, manifest I/O | ✅ |
| 1 — binder + editor | v0.1.0 | writing experience, scene CRUD, pane modes | ✅ |
| 2 — revision history | v0.1.0 | go-git auto-commit, history browser, diff, restore | ✅ |
| 3 — llm integration | v0.2.0 | Python backend, NDJSON protocol, rewrite/generate/summarize/ask | ✅ |
| 4 — mirror + research | v0.3.0 | style analysis, research gathering, auto-tag, tag system | ✅ |
| 5 — export + polish | v1.0.0 | manuscript/docx export, themes, corkboard, outline, sprints | ✅ |
| 6 — character + scene notes | v1.1.0 | character sheets, scene notes, timeline view | ✅ |
| 7 — jj backend | v1.2.0 | jj revision history, git→jj migration | ✅ |

## archived phase breakdown

### phase 0: scaffolding (v0.0.1)

- [x] **project creation.** `chisel new my-project` scaffolds the full directory structure
- [x] **config loading.** read/write `config.json` with llm/mirror/history/editor/goals/theme slots
- [x] **manifest I/O.** JSONL read/write/append — append-only during normal use
- [x] **Go module init.** bubbletea, lipgloss, bubbles, go-git dependencies

### phase 1: binder + editor (v0.1.0)

- [x] **binder tree.** recursive `scenes/` scan, expand/collapse folders, status inline
- [x] **markdown editor.** modeless, Ctrl+S/Ctrl+Z/Ctrl+F via bubbles/textarea
- [x] **scene CRUD.** `n` (inline prompt), `d` (confirm dialog), `F2` (inline prompt), `K`/`J` reorder
- [x] **manifest tracking.** word count, modified timestamp on save; draft_order on reorder
- [x] **pane modes.** Ctrl+1 (editor), Ctrl+2 (binder+editor), Ctrl+3 (binder+editor+LLM)
- [x] **status bar.** scene name, word count, modified indicator, session timer, daily goal
- [x] **file-backed.** all writes go direct to `.md` files

### phase 2: revision history (v0.1.0 cont.)

- [x] **RevisionBackend interface.** Save/Log/Diff/Restore abstracted for git or jj
- [x] **go-git backend.** auto-commit on every Ctrl+S, structured commit messages
- [x] **history browser.** Ctrl+H — revision list with timestamps and messages
- [x] **diff view.** enter — unified diff between selected snapshots
- [x] **restore.** r — restore full scene from any snapshot

### phase 3: llm integration (v0.2.0)

- [x] **chisel.py backend.** NDJSON protocol, subprocess manager (tui/llm.go)
- [x] **provider config.** separate `llm` and `mirror` slots in config.json
- [x] **rewrite.** Ctrl+R — alternatives in LLM panel
- [x] **generate.** Ctrl+G — continue from cursor
- [x] **summarize.** Ctrl+Shift+S — summary of selection or scene
- [x] **ask.** Ctrl+K — inline prompt, streaming response
- [x] **pane mode 3.** binder + editor + LLM panel
- [x] **streaming.** tokens appear as they arrive
- [x] **error handling.** graceful degradation when backend is unavailable

### phase 4: mirror + research (v0.3.0)

- [x] **style analysis.** Ctrl+A — mirror model analyses current scene
- [x] **research gathering.** Ctrl+F5 — prompt for topic, saves to `research/{slug}.md`
- [x] **auto-tag.** research notes auto-linked to current scene via `research_refs`
- [x] **tag system.** add/remove tags, filter binder by tag (t/T)

### phase 5: export + polish (v1.0.0)

- [x] **export to manuscript.** concatenate scenes in draft order → `exports/manuscript.md`
- [x] **export to docx.** pandoc wrapper → `exports/manuscript.docx`
- [x] **themes.** 5 colour palettes — peach, dark, light, forest, ocean
- [x] **session word count.** tracked in status bar
- [x] **daily goals.** configurable target, progress percentage in status bar
- [x] **corkboard view.** grid of scene cards
- [x] **outline view.** collapsible titles and status
- [x] **timeline view.** tree markers with dates
- [x] **writing sprints.** 25-min pomodoro (Ctrl+Shift+P)
- [x] **typewriter mode.** toggle (Ctrl+Shift+T)
- [x] **reading mode.** full-screen, no UI chrome (Ctrl+Shift+R)
- [x] **vim bindings.** opt-in toggle (Ctrl+Shift+V)

### phase 6: character + scene notes (v1.1.0)

- [x] **character sheets.** `characters/` directory, `.md` profiles with name/description/arc/relationships
- [x] **character view.** Ctrl+Shift+C — browse character sheet cards
- [x] **scene notes.** per-scene planning notes in manifest `notes` field (Ctrl+Shift+N)

### phase 7: jj backend (v1.2.0)

- [x] **jj detection.** auto-detected from config (`history.backend: "jj"`)
- [x] **jj impl.** uses `jj describe` + `jj new` on save, `jj log`/`jj diff`/`jj file show` for history
- [x] **history browser.** same UI, different backend — swapped transparently

## original plan (reference)

The full original implementation plan with task breakdown is preserved in git history at commit `5d92243`. See [CHANGELOG.md](CHANGELOG.md) for the complete version history.

## dependencies

**Go** (TUI):
- `github.com/charmbracelet/bubbletea`
- `github.com/charmbracelet/lipgloss`
- `github.com/charmbracelet/bubbles` (textarea, table, tree)
- `github.com/go-git/go-git/v5` (pure Go git for auto-commit, no shelling out)

**Python** (LLM backend):
- `openai` (API client — works with LM Studio, llama.cpp, OpenAI, Anthropic)

**System:**
- `pandoc` (optional, for .docx export)
- `jj` (optional, v1.2 for jj revision backend)

## platform notes

- **Windows compatibility:** all file paths use Go's `filepath` package (never hardcoded separators). `os.Executable()` locates `chisel.py` relative to the binary at runtime. Reserved Windows characters (`\ / : * ? " < > |`) stripped from folder and scene names.
- **Go module structure:** root module is `github.com/acgh213/chisel`. Go source lives in `tui/` package under root. `chisel.py` lives at repo root, found at runtime via `os.Executable()`.
