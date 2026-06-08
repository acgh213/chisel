# changelog

all notable changes to chisel will be documented in this file.

format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
chisel uses [Semantic Versioning](https://semver.org/).

---

## [unreleased]

### planned
- tag browser + binder filtering (#20)
- split reference pane — Ctrl+\ to view two scenes side by side (#25)
- character mention detection in right panel (#29)
- collections — cross-folder scene groupings (#33)
- custom metadata columns in outliner (#34)
- content blocks — transclude/embed one scene inside another (#42)
- light mode / theme toggle (#52)
- kitty keyboard protocol support (#56)
- declarative cursor control (#57)
- vim mode (Ctrl+Shift+V) (#30)
- project statistics — word count history chart with trend lines (#21, basic version shipped)

---

## [0.2.0] — 2026-06-08

### added

**status chrome and help layer (#64, #66):**
- two-row bottom shelf — stable state row + hint row that never changes height
- state row shows: file name, word count, reading time, modified marker, sprint state, session words, streak badge, project target progress
- hint row shows compact context-sensitive key hints; shrinks (not hides) during sprints
- `?` opens a full help layer from any non-text-input state (binder, structural views, history)
- `?` inserts a literal question mark in editor, search, prompt, and quick-note
- help layer shows global, binder, editor, structural view, and overlay keys
- structural views (corkboard, outliner, timeline, history) show context in state row (e.g. "Corkboard — Acts (12 scenes)")

**git history features (#44, #36, #21, #70):**
- writing streak badge — 🔥 N days in state row, computed from git commit history
- writing calendar heatmap (F9) — 52-week grid with month labels, day headers, commit drilldown overlay
- word-count history chart (F10) — scrollable bar chart of daily word deltas parsed from commit messages; negative deltas highlighted in red
- `AllLog(since)` — date-range-limited git log, no longer loads entire repo history
- `FloorUTC` exported from core/ for shared use

**project word target (#31):**
- `project_target` field in `.chisel.yaml` — set your novel's target word count
- `📊 23,412/80,000 (29%)` progress indicator in state row
- full-width progress bar in stats view (F10)
- `core.ProjectWordCount` walks all scene files and sums word counts

**document links (#26):**
- `[[scene-name]]` syntax detected in editor
- Ctrl+Enter on a link navigates to the linked scene
- resolves by filename, title, or partial match (case-insensitive)
- searches scenes, characters, and locations
- editor hint row shows `^Enter link`

**reading time estimate (#27):**
- `~X min read` shown next to word count in state row (200 WPM)

**project bookmarks (#41):**
- Ctrl+B toggles bookmark on current scene, persists to `.chisel.yaml`
- ★ prefix shown on bookmarked scenes in binder
- F11 opens bookmark list view — Enter to navigate, Esc to close
- empty state shows "No bookmarks — press Ctrl+B on a scene to bookmark it"

**outline-only export (#37):**
- Ctrl+O exports `exports/outline.md` — scene structure without prose
- each scene shows: title, status, word count/target, tags, POV, timeline date, synopsis (blockquote), notes

**bubble tea v2 migration (#54, #62):**
- migrated from Bubble Tea v1 to v2 (charm.land/bubbletea/v2)
- Lipgloss v2, Bubbles v2
- all key messages use `tea.KeyPressMsg`
- sprint timer uses v2 native terminal progress bar (#55)

**documentation (#65):**
- per-folder CLAUDE.md files for core/ and tui/
- API reference for every type and function
- architecture invariants and file conventions

### changed
- theme cycling moved from Ctrl+T to F8 (consistent with F-key view toggles)
- status bar replaced by two-row bottom shelf (breaking change for status bar customization)
- inline 12-cell sprint progress bar removed in favor of v2 native terminal progress bar
- `AllLog` now takes a `since time.Time` parameter for date-range limiting

### fixed
- sprint shelf no longer competes with key hints for the same row
- help layer prevents `?` from interfering with text input states
- quick-note popup positioned correctly above two-row shelf

---

## [0.1.0] — 2026-05-25 to 2026-06-07

complete rewrite from scratch in Go with Bubble Tea TUI framework.
the previous version (0.0.x–1.2.x) used a Python backend with NDJSON protocol and LLM integration.
the rewrite is pure Go — no Python, no system git, no LLM dependency.

### phases shipped

**phase 0 — stabilize TUI:** basic binder + editor layout, sizing, blink, quit, editor Enter key

**phase 1 — extract core:** `core/` package separated from `tui/` — zero Charm imports, plain Go structs only

**phase 2 — scene metadata:** YAML frontmatter parsing, `Metadata` struct, `LoadScene`/`Save`, round-trip serialization

**phase 3 — revision history:** go-git backend (`GitBackend`), `RevisionBackend` interface, auto-snapshot on Ctrl+S, history browser (Ctrl+H), diff view, restore

**phase 4 — corkboard + outliner:** index-card grid (F2), collapsible outline (F3), cross-hop navigation

**phase 5 — compile + export:** `Project.Export()` → `exports/manuscript.md`, optional `.docx` via pandoc

**phase 6 — binder CRUD:** create scene/folder (n/N), rename (r), delete (d), inline prompt bar

**phase 7 — chisel init:** `chisel init` scaffolds new projects, templates (minimal, novel, short-stories)

**phase 8 — right panel:** character/location inspector (F5), world index, scene notes

**phase 9 — location sheets:** `locations/` directory, `ListLocations`, location metadata

**phase 10 — timeline:** date-sorted scene list (F4), `BuildTimeline`, cross-hop navigation

**phase 11 — quick note:** floating popup (backtick), any state, saves to `notes/scratch.md`

**phase 12 — scene notes:** `notes` frontmatter field, richer entity sheets, right panel integration

**phase 13 — search:** full-text search overlay (Ctrl+F), case-insensitive, browse results

**phase 14 — reading mode:** full-screen reading (F6), word-wrapped, no chrome, scroll with j/k

**phase 15–18 — polish:** layout fixes, edge cases, test coverage, documentation

**phase 19 — themes + session stats + sprint timer:** four dark themes (peach/forest/ocean/midnight, F8), session word count accumulation, 25-minute pomodoro sprint timer (F7), daily goal tracking

### architecture
- `core/` — pure Go data layer (zero Charm imports)
- `tui/` — Bubble Tea presentation layer
- filesystem is the project — `.md` files with YAML frontmatter
- git-backed revision history (go-git, pure Go)
- four dark themes driven by `tui/styles.go` color tokens
- structural views follow a consistent pattern: `update()` returns `viewAction`, sized via `layout()`, integrated with bottom shelf

---

## [0.0.x–1.2.x] — 2026-05-24 (legacy)

the original version with Python backend, NDJSON protocol, and LLM integration.
this version was completely replaced by the Bubble Tea rewrite (0.1.0+).

### included
- project scaffolding, config/manifest I/O
- binder tree, markdown editor, revision history
- LLM integration via Python subprocess (rewrite, generate, summarize, ask)
- mirror analysis, research gathering, auto-tag
- export to manuscript, corkboard, outline, themes
- character sheets, scene notes, timeline
- jj backend for revision history

### why it was rewritten
- Python dependency was fragile (subprocess management, environment setup)
- LLM integration added complexity without clear value for a writing tool
- the TUI framework (original) was replaced by Bubble Tea for better component reuse
- the rewrite is simpler, faster, and has zero external dependencies beyond go-git
