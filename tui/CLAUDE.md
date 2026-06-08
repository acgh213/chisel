# tui/ — Bubble Tea presentation layer

All user-facing rendering and input handling. Depends on `core/` for all data operations.
No filesystem writes happen here except through `core` functions.

## key invariants

- Sub-views never mutate root model state directly — they return action enums and the root applies them.
- Key dispatch priority in `model.Update()`: quickNote → search → reader → history → structural views → prompt → normal binder/editor dispatch.
- Layout: binder + editor widths must always sum to terminal width. Verify with layout tests in `views_test.go`.
- No hardcoded hex colors — all styles reference `Color*` vars from `styles.go`.

---

## file reference

### model.go

**Types:** `Pane` (enum), `viewMode` (enum), `Model`

**Pane constants:** `PaneBinder`, `PaneEditor`

**viewMode constants:** `viewMain`, `viewCorkboard`, `viewOutliner`, `viewTimeline`

`Model` owns: `Project *core.Project`, `Binder BinderModel`, `Editor EditorModel`, `History historyModel`, `Corkboard corkboardModel`, `Outliner outlinerModel`, `Timeline timelineModel`, `RightPanel rightPanelModel`, `Search searchModel`, `QuickNote quickNoteModel`, `Reader readerModel`, `Prompt binderPrompt`, plus layout/theme/sprint-timer state.

**Key exports:**
- `NewModel(root string) (Model, error)` — bootstrap all sub-models, load config, detect pandoc
- `Model.Update(msg tea.Msg) (tea.Model, tea.Cmd)` — root dispatcher; enforces sub-view priority
- `Model.View() string` — compose final screen string from active sub-view + status bar

Sprint timer: started/stopped by F7; `timerStart time.Time`, `timerRunning bool`, `sprintWords int` (words at timer start). Status bar renders countdown while active.

---

### binder.go

**Types:** `BinderModel`

**Key exports:**
- `NewBinder(root string) BinderModel`
- `BinderModel.Update(msg tea.Msg) (BinderModel, tea.Cmd)`
- `BinderModel.Init() tea.Cmd`
- `BinderModel.SelectedNode() *core.FileNode` — nil if nothing selected
- `BinderModel.Refresh() error` — rebuild tree from disk (called after CRUD)
- `BinderModel.View() string`

Navigation: j/k or arrows move cursor; Enter opens file or toggles folder; Space toggles folder.
CRUD keys (n/N/r/d) only fire when binder has focus; when editor is focused those keys go to the textarea.

---

### editor.go

**Types:** `EditorModel`

**Key exports:**
- `NewEditor() EditorModel`
- `EditorModel.Update(msg tea.Msg) (EditorModel, tea.Cmd)`
- `EditorModel.Init() tea.Cmd`
- `EditorModel.OpenScene(s *core.Scene)` — load scene into textarea
- `EditorModel.CurrentScene() *core.Scene` — nil if no scene open
- `EditorModel.IsModified() bool`
- `EditorModel.Save() error` — calls `scene.Save()` + triggers revision snapshot
- `EditorModel.View() string`

Tracks `modified bool` and `original string` (body at open time) to detect unsaved changes.
Ctrl+S saves and snapshots; Ctrl+H opens history; Ctrl+E triggers export.

---

### history.go

**Types:** `historyAction` (enum), `historyMode` (enum), `historyModel`

**historyAction constants:** `historyNone`, `historyClose`, `historyRestore`

**historyMode constants:** `historyList`, `historyDiff`

Internal to `model.go` — created as a field of `Model`. Accessed via `model.History`.

`historyModel.update(msg) (historyModel, historyAction, tea.Cmd)` — returns an action the root applies.

In list mode: ↑/↓ navigate snapshots, Enter shows diff, `r` restores. In diff mode: ↑/↓ scroll, Esc returns to list. Esc in list closes the browser.

---

### corkboard.go

**Types:** `corkboardModel`, `viewAction` (enum)

**viewAction constants:** `viewNone`, `viewClose`, `viewOpen`

Index-card grid layout: cards are fixed-width columns; row count adjusts to terminal height.
Navigation: ←/→/↑/↓ move grid cursor; Enter opens scene; Esc/F1 returns to main; F3/F4 cross-hop to outliner/timeline.

`corkboardModel.update(msg) (corkboardModel, viewAction, *core.Scene)` — the root applies the returned action.

---

### outliner.go

**Types:** `outlinerModel`

Collapsible tree of all scenes with status glyph, word count, and word target columns.
Navigation mirrors corkboard (↑/↓, Enter, Esc/F1, F2/F4 cross-hop).
Maintains its own expand/collapse state independently of the binder.

Returns `viewAction` to the root on Enter/Esc/cross-hop.

---

### timeline.go

**Types:** `timelineModel`

Flat sorted list of all scenes with `timeline_date`. Dated scenes first (ascending), then undated alphabetically. Section headers separate the two groups.

Navigation: ↑/↓, Enter opens scene, Esc/F1 returns to main, F2/F3 cross-hop.
Returns `viewAction` to the root.

---

### quicknote.go

**Types:** `quickNoteAction` (enum), `quickNoteModel`

**quickNoteAction constants:** `quickNoteNone`, `quickNoteConfirmed`, `quickNoteCancelled`

Floating single-line input overlay, highest render priority (drawn on top of any view).
Backtick opens from any state. Enter confirms (calls `core.AppendScratch`), Esc cancels.
Returns `quickNoteAction` so the root can clear the overlay.

---

### search.go

**Types:** `searchAction` (enum), `searchInputMode` (enum), `searchModel`

**searchAction constants:** `searchNone`, `searchOpen`, `searchClose`

**searchInputMode constants:** `searchInputting`, `searchBrowsing`

Two-phase UX:
1. **Inputting** — type query, Enter triggers `core.SearchScenes`, switches to browsing mode
2. **Browsing** — ↑/↓ navigate results, Enter opens scene (returns `searchOpen`), Esc resets to inputting

Overlay drawn at the same z-level as reader/history; root priority ensures Ctrl+F closes any other overlay first.

---

### reader.go

**Types:** `readerModel`

Full-screen reading mode. Word-wraps scene body into a centered 72-char column.
Scroll state: `offset int` (line index). Navigation: ↑/↓/j/k = ±1 line, Ctrl+D/U = half page.
F6 or Esc exits; the root restores the previous pane focus.

No edit capability — read-only view of the current scene body.

---

### rightpanel.go

**Types:** `rpContent` (enum), `entityDetail`, `rightPanelModel`

**rpContent constants:** `rpEmpty`, `rpWorld`, `rpEntity`, `rpError`

Passive reactive panel — no cursor, no focus, no key handling.
`rightPanelModel.SyncToSelection(node *core.FileNode, root string)` — called by the root on every binder navigation and after every CRUD refresh; loads character or location detail if a entity file is selected, otherwise shows cast list.

Two display modes toggled by W:
- **World Index** — alphabetical list of all characters + locations
- **Scene Notes** — `notes` frontmatter field of the currently selected scene

---

### prompt.go

**Types:** `promptMode` (enum), `binderPrompt`

**promptMode constants:** `promptNone`, `promptNewFile`, `promptNewFolder`, `promptRename`, `promptDelete`, `promptNote`

Occupies the status-bar row during CRUD. Has its own `update()` and `view()`.
Delete mode shows "y to confirm" and accepts a single keypress.
All other modes use a `textinput.Model` for text entry.

The root checks `prompt.mode != promptNone` before any Esc/quit handling so Esc cancels the prompt instead of quitting.

---

### styles.go

**Color variables** (package-level, rebound by `ApplyTheme`):
`ColorBg`, `ColorFg`, `ColorAccent`, `ColorMuted`, `ColorBorder`, `ColorHighlight`, `ColorDim`, `ColorGreen`, `ColorRed`

**Style variables** (rebuilt by `rebuildStyles()`):
`BinderStyle`, `EditorStyle`, `StatusBarStyle`, `PromptBarStyle`, `HistoryStyle`, `RightPanelStyle`, `CardStyle`, `CardSelectedStyle`, `ViewHeaderStyle`, `DiffAddStyle`, `DiffDelStyle`, `DiffMetaStyle`, `MetTargetStyle`, `FocusedBorderColor`

**Key exports:**
- `ApplyTheme(name string)` — rebind `Color*` vars for named theme, then call `rebuildStyles()`
- `NextTheme(current string) string` — cycle: peach → forest → ocean → midnight → peach
- `FocusedStyle(base lipgloss.Style) lipgloss.Style` — apply focused-border color to any style

**Themes:** `peach` (warm default), `forest` (green), `ocean` (blue), `midnight` (purple).
Adding a new theme: add a case in `ApplyTheme`, no other changes needed.
