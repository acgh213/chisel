# ✧ chisel — goals & roadmap ✧

## all goals achieved ✅

Every item in the original roadmap has been implemented. See [CHANGELOG.md](CHANGELOG.md) for the full version history.

## short-term (v0.1 — "write something") ✅

- [x] **project creation.** `chisel new my-project` scaffolds the directory structure, manifest, and config
- [x] **binder tree.** navigate a scene tree in the left panel, expand/collapse folders, see word counts and status inline
- [x] **markdown editor.** modeless text editing with standard shortcuts (save, find, undo, cut/copy/paste)
- [x] **scene CRUD.** create, rename, delete, and reorder scenes from the binder
- [x] **manifest tracking.** word count updates on save, status toggles (draft → revised → done), modified timestamps
- [x] **pane mode switching.** `ctrl+1` / `ctrl+2` to toggle between editor-only and binder+editor
- [x] **file-backed.** everything saves to `.md` files on disk. close chisel, open the folder in VS Code, your writing is there
- [x] **revision history.** every save creates an automatic snapshot (git-backed, jj-ready API). browse history, compare versions, restore passages — all from within chisel

## medium-term (v0.2 — "think with it") ✅

- [x] **Python backend.** `chisel.py` with NDJSON protocol, provider abstraction, and operation dispatch
- [x] **rewrite.** select text, `Ctrl+R`, get alternatives in the LLM panel
- [x] **generate.** `Ctrl+G` to continue from cursor with optional guidance
- [x] **summarize.** `Ctrl+Shift+S` to summarize selection or current scene
- [x] **ask.** `Ctrl+K` opens a prompt bar; ask questions about the text or research a topic
- [x] **mode 3.** `ctrl+3` adds the LLM panel as a third pane
- [x] **provider config.** `config.json` with separate slots for general LLM and mirror model
- [x] **streaming responses.** LLM output appears token by token in the panel, not all at once

## medium-term (v0.3 — "find the patterns") ✅

- [x] **style analysis.** `Ctrl+A` runs the current scene through the mirror model — surfaces overused words, rhythm issues, tics
- [x] **research gathering.** `Ctrl+F5` researches a topic and saves notes to `research/`, auto-tagged to current scene
- [x] **research panel.** view all notes tagged to the current scene in the LLM panel
- [x] **cross-scene analysis.** compare the rhythm of scene 1 and scene 4 — mirror model across multiple files
- [x] **tag system.** add/remove tags on scenes, filter binder by tag

## long-term (v1.0 — "a real tool") ✅

- [x] **export to manuscript.** concatenate scenes in draft order into one `.md` file in `exports/`
- [x] **export to docx.** optional MS Word output via pandoc
- [x] **vim bindings.** opt-in toggle in settings
- [x] **themes.** configurable color palettes (peach default, plus dark, light, forest, ocean)
- [x] **session word count.** "you wrote 847 words this session" — tracks words added since open
- [x] **daily goals.** configurable word count target, progress bar
- [x] **git integration.** auto-commit on save with meaningful messages
- [x] **corkboard view.** grid of scene cards showing title, word count, status, first line
- [x] **outline view.** collapsible outline of all scenes showing only titles and status, for structural editing
- [x] **distraction-free mode.** hide everything except the editor — one keystroke

## long-term (v1.1+ — "stretch goals") ✅

- [x] **character sheets.** a `characters/` folder with markdown files for each character — name, description, arc, relationships. automatically linked when a character is mentioned in a scene. editable alongside the text in a dedicated view
- [x] **scene notes.** per-scene planning notes and outlines, viewable alongside the editor. scrivener's "document notes" pane, but in a terminal
- [x] **timeline view.** visual timeline of scenes by in-world date, not just draft order
- [x] **reading mode.** full-screen, justified text, no UI chrome — like a kindle
- [x] **sprint timer.** pomodoro-style writing sprints with word count tracking per sprint
- [x] **typewriter mode.** center the current line, dim everything else, disable backspace
- [x] **standalone binary.** `go build` produces a single `.exe` — no Python dependency for the TUI itself (LLM features would still need the Python backend or an embedded HTTP client)

## remaining stretch goals (not yet implemented)

These features from the original stretch-goal list were deferred as non-blocking:

- [ ] **quarkdown rendering.** swap the markdown renderer for something that handles typographic features (small caps, ligatures, proper quotes) — for the editor display and export
- [ ] **image paste.** paste images from clipboard into a scene (saved to `scenes/assets/`)
- [ ] **cloud sync.** optional sync of the project folder via any file sync tool (dropbox, syncthing, etc.) — no built-in cloud, just works with whatever you use

## non-goals

things chisel explicitly does *not* try to be (unchanged):

- **not a code editor.** syntax highlighting for prose only. no language server protocol. no debugging.
- **not a cloud service.** no accounts, no sync, no telemetry. your writing is files on your disk.
- **not a publishing tool.** export is simple concatenation. for typesetting, use pandoc or a real layout tool.
- **not collaborative.** single-user. no real-time multiplayer editing. git handles version conflicts.
- **not a replacement for scrivener.** scrivener does a hundred things chisel won't. chisel does ten things, locally, in a terminal, and you can read the source in an afternoon.
