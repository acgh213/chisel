# ✧ chisel — goals & roadmap ✧

## short-term (v0.1 — "write something")

the goal for v0.1 is a tool you can actually draft in. not feature-complete, but usable end-to-end.

- [ ] **project creation.** `chisel new my-project` scaffolds the directory structure, manifest, and config
- [ ] **binder tree.** navigate a scene tree in the left panel, expand/collapse folders, see word counts and status inline
- [ ] **markdown editor.** modeless text editing with standard shortcuts (save, find, undo, cut/copy/paste)
- [ ] **scene CRUD.** create, rename, delete, and reorder scenes from the binder
- [ ] **manifest tracking.** word count updates on save, status toggles (draft → revised → done), modified timestamps
- [ ] **pane mode switching.** `ctrl+1` / `ctrl+2` to toggle between editor-only and binder+editor
- [ ] **file-backed.** everything saves to `.md` files on disk. close chisel, open the folder in VS Code, your writing is there
- [ ] **revision history.** every save creates an automatic snapshot (git-backed, jj-ready API). browse history, compare versions, restore passages — all from within chisel

## medium-term (v0.2 — "think with it")

llm features arrive. the tool becomes a thinking partner.

- [ ] **Python backend.** `chisel.py` with NDJSON protocol, provider abstraction, and operation dispatch
- [ ] **rewrite.** select text, `Ctrl+R`, get alternatives in the LLM panel
- [ ] **generate.** `Ctrl+G` to continue from cursor with optional guidance
- [ ] **summarize.** `Ctrl+Shift+S` to summarize selection or current scene
- [ ] **ask.** `Ctrl+K` opens a prompt bar; ask questions about the text or research a topic
- [ ] **mode 3.** `ctrl+3` adds the LLM panel as a third pane
- [ ] **provider config.** `config.json` with separate slots for general LLM and mirror model
- [ ] **streaming responses.** LLM output appears token by token in the panel, not all at once

## medium-term (v0.3 — "find the patterns")

the mirror model gets integrated. research becomes structured.

- [ ] **style analysis.** `Ctrl+A` runs the current scene through the mirror model — surfaces overused words, rhythm issues, tics
- [ ] **research gathering.** `Ctrl+F5` researches a topic and saves notes to `research/`, auto-tagged to current scene
- [ ] **research panel.** view all notes tagged to the current scene in the LLM panel
- [ ] **cross-scene analysis.** "compare the rhythm of scene 1 and scene 4" — mirror model across multiple files
- [ ] **tag system.** add/remove tags on scenes, filter binder by tag

## long-term (v1.0 — "a real tool")

polish, export, and features that make chisel feel complete.

- [ ] **export to manuscript.** concatenate scenes in draft order into one `.md` file in `exports/`
- [ ] **export to docx.** optional MS Word output via pandoc
- [ ] **vim bindings.** opt-in toggle in settings
- [ ] **themes.** configurable color palettes (peach default, plus dark, light, forest, ocean)
- [ ] **session word count.** "you wrote 847 words this session" — tracks words added since open
- [ ] **daily goals.** configurable word count target, progress bar
- [ ] **git integration.** optional auto-commit on save with meaningful messages
- [ ] **corkboard view.** grid of scene cards showing title, word count, status, first line — scrivener's corkboard in a terminal
- [ ] **outline view.** collapsible outline of all scenes showing only titles and status, for structural editing
- [ ] **distraction-free mode.** hide everything except the editor — one keystroke

## long-term (v1.1+ — "stretch goals")

features that would be amazing but aren't blocking.

- [ ] **quarkdown rendering.** swap the markdown renderer for something that handles typographic features (small caps, ligatures, proper quotes) — for the editor display and export
- [ ] **image paste.** paste images from clipboard into a scene (saved to `scenes/assets/`)
- [ ] **character sheets.** a `characters/` folder with markdown files for each character — name, description, arc, relationships. automatically linked when a character is mentioned in a scene. editable alongside the text in a dedicated view
- [ ] **scene notes.** per-scene planning notes and outlines, viewable alongside the editor. scrivener's "document notes" pane, but in a terminal
- [ ] **timeline view.** visual timeline of scenes by in-world date, not just draft order
- [ ] **reading mode.** full-screen, justified text, no UI chrome — like a kindle
- [ ] **sprint timer.** pomodoro-style writing sprints with word count tracking per sprint
- [ ] **typewriter mode.** center the current line, dim everything else, disable backspace
- [ ] **cloud sync.** optional sync of the project folder via any file sync tool (dropbox, syncthing, etc.) — no built-in cloud, just works with whatever you use
- [ ] **standalone binary.** `go build` produces a single `.exe` — no Python dependency for the TUI itself (LLM features would still need the Python backend or an embedded HTTP client)

## non-goals

things chisel explicitly does *not* try to be:

- **not a code editor.** syntax highlighting for prose only. no language server protocol. no debugging.
- **not a cloud service.** no accounts, no sync, no telemetry. your writing is files on your disk.
- **not a publishing tool.** export is simple concatenation. for typesetting, use pandoc or a real layout tool.
- **not collaborative.** single-user. no real-time multiplayer editing. git handles version conflicts.
- **not a replacement for scrivener.** scrivener does a hundred things chisel won't. chisel does ten things, locally, in a terminal, and you can read the source in an afternoon.
