# ✧ chisel — implementation plan ✧

## phase 0: scaffolding (v0.0.1)

the foundation. nothing visible yet, but everything after builds on it.

- [ ] **project creation.** `chisel new my-project` scaffolds:
  - `manifest.jsonl` (empty, valid)
  - `config.json` (defaults)
  - `scenes/` directory
  - `research/` directory
  - `exports/` directory
  - `.gitignore` (ignore exports, keep everything else)
  - `git init` + initial commit
- [ ] **config loading.** read `config.json` into Go structs. default values for missing fields
- [ ] **manifest I/O.** read/write JSONL manifest. load into memory on open, flush on change
- [ ] **Go module init.** `go mod init github.com/acgh213/chisel`, bubbletea dependency, project structure:
  ```
  tui/
  ├── main.go
  ├── model.go
  ├── binder.go
  ├── editor.go
  ├── manifest.go
  ├── config.go
  └── styles.go
  ```
- [ ] **verify.** `go build` produces `chisel.exe`. `chisel new test-project` creates a valid directory

## phase 1: binder + editor (v0.1.0)

the core writing experience. you can create scenes, write in them, and navigate between them.

- [ ] **binder tree.** left panel showing `scenes/` directory tree. expand/collapse folders. scene status shown inline (`draft` · `revised` · `done`)
- [ ] **markdown editor.** center panel. modeless editing with standard shortcuts (Ctrl+S save, Ctrl+Z undo, Ctrl+F find)
- [ ] **scene CRUD.** `n` new scene under current folder, `d` delete with confirm, `F2` rename, drag-to-reorder (binder moves)
- [ ] **manifest tracking.** on save: update word count, modified timestamp. on binder reorder: update `draft_order`, rewrite manifest
- [ ] **pane mode 1.** `ctrl+1` — editor only, full screen
- [ ] **pane mode 2.** `ctrl+2` — binder + editor, default layout
- [ ] **status bar.** bottom bar showing current scene, word count, session timer
- [ ] **file-backed.** all writes go direct to `.md` files. save and quit — it's on disk

## phase 2: revision history (v0.1.0 cont.)

every save is a snapshot. no manual commits.

- [ ] **auto-commit on save.** `git add -A && git commit -m "scene: {id} — {word_count} words"`
- [ ] **history browser.** `Ctrl+H` opens history sidebar for current scene — list of save points with timestamps and word counts
- [ ] **diff view.** select two snapshots, see side-by-side diff of the scene
- [ ] **restore.** restore selected passage or full scene from any snapshot
- [ ] **config.** `history.backend: "git"` in config.json. jj slot empty but API-abstracted for v1.2

## phase 3: llm integration (v0.2.0)

the Python backend wakes up. models become reachable.

- [ ] **chisel.py backend.** subprocess manager in Go, NDJSON protocol:
  - Request: `{"op": "rewrite", "text": "...", "context": "..."}`
  - Response: `{"op": "rewrite", "result": "...", "tokens": N}`
- [ ] **provider config.** separate `llm` and `mirror` slots in `config.json`
- [ ] **rewrite.** select text, `Ctrl+R` → alternatives in LLM panel
- [ ] **generate.** `Ctrl+G` → continue from cursor with optional guidance
- [ ] **summarize.** `Ctrl+S` → summary of selection or current scene
- [ ] **ask.** `Ctrl+K` → prompt bar opens, response streams to LLM panel
- [ ] **pane mode 3.** `ctrl+3` — binder + editor + LLM panel
- [ ] **streaming.** tokens appear in LLM panel as they arrive, not all at once
- [ ] **error handling.** backend crash, timeout, model unavailable — graceful messages, no TUI freeze

## phase 4: mirror + research (v0.3.0)

the mirror model finds patterns. research becomes structured.

- [ ] **style analysis.** `Ctrl+A` → mirror model analyzes current scene. output: overused words, rhythm notes, tic flags
- [ ] **research gathering.** `Ctrl+F5` → prompt for topic, backend researches, saves to `research/{slug}.md`
- [ ] **auto-tag.** new research note auto-tagged to current scene's `research_refs`
- [ ] **research panel.** in mode 3, show all research notes tagged to current scene
- [ ] **cross-scene analysis.** compare two scenes' style profiles
- [ ] **tag system.** add/remove tags on scenes via manifest edit. filter binder by tag

## phase 5: export + polish (v1.0.0)

the tool feels complete.

- [ ] **export to manuscript.** concatenate scenes in draft order → `exports/manuscript.md`
- [ ] **export to docx.** pandoc wrapper → `exports/manuscript.docx`
- [ ] **vim bindings.** opt-in toggle in settings
- [ ] **themes.** peach (default), dark, light, forest, ocean
- [ ] **session word count.** "you wrote 847 words this session"
- [ ] **daily goals.** configurable target, progress bar
- [ ] **corkboard view.** grid of scene cards — title, word count, status, first line
- [ ] **outline view.** collapsible titles and status only

## phase 6: character + scene notes (v1.1.0)

the scrivener features that matter most.

- [ ] **character sheets.** `characters/` folder, one `.md` per character. name, description, arc, relationships
- [ ] **auto-linking.** character names in scenes detected and linked to character sheets
- [ ] **character view.** browse and edit character sheets alongside the editor
- [ ] **scene notes.** per-scene planning/outline notes stored in manifest `notes` field. editable in a popover or side panel

## phase 7: jj backend (v1.2.0)

jj replaces git for revision history.

- [ ] **jj detection.** if `jj` is in PATH, offer as backend option
- [ ] **jj impl.** `jj new` + `jj describe` on save instead of `git commit`
- [ ] **history browser.** same UI, different backend — swapped transparently
- [ ] **migration.** `chisel migrate-history` converts git repo to jj repo

## dependencies

**Go** (TUI):
- `github.com/charmbracelet/bubbletea`
- `github.com/charmbracelet/lipgloss`
- `github.com/charmbracelet/bubbles` (textarea, table, tree)
- `github.com/go-git/go-git` (pure Go git for auto-commit, no shelling out)

**Python** (LLM backend):
- `openai` (API client — works with LM Studio, llama.cpp, OpenAI, Anthropic)
- `pillow` (image handling, future-proofing)

**System:**
- `git` (in PATH, for revision history v0.1-v1.1)
- `jj` (optional, v1.2)
- `pandoc` (optional, for .docx export)
