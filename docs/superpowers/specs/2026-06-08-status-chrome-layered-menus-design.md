# Status Chrome and Layered Menus Design

## Purpose

Issue #64 identifies a real UI pressure point: Chisel's single-row status bar now carries temporary messages, file state, keyboard hints, sprint state, and session progress. The immediate problem is crowding, but the larger opportunity is to make Chisel feel more like a capable modern TUI with layered help and menu surfaces instead of one long row of key text.

This design defines the next UX direction before implementation:

- Reserve two bottom rows for stable application chrome.
- Move the complete keymap into a `?` help layer.
- Shrink context hints during sprints instead of hiding them.
- Treat the Bubble Tea v2 native terminal progress bar as the primary sprint progress indicator.
- Prepare for future layered menu screens without building the full menu system in the first pass.

## Goals

- Keep the writing surface calm and predictable across resizes, sprints, prompts, and overlays.
- Preserve keyboard discoverability without forcing the full keymap into the status row.
- Make sprint state visible without letting it dominate the whole interface.
- Establish a root-owned layer model for help and future menus.
- Keep styling theme-driven through `tui/styles.go`, including future light-theme readiness.

## Non-Goals

- Do not implement a full top menu bar in the first pass.
- Do not convert the passive right panel into the primary stats dashboard yet.
- Do not change core data models.
- Do not add Charm dependencies to `core/`.
- Do not replace existing overlays such as search, quick note, reader, history, corkboard, outliner, or timeline.

## Current Context

`tui/model.go` currently builds one `statusParts` slice in `View()`. It may contain:

- temporary status messages
- open file name, word count, and modified marker
- full focus-specific keyboard hints
- sprint countdown, inline progress bar, and sprint word gain
- session word count and daily goal progress

The resulting string is joined and truncated into one status row. This behaves predictably from a code perspective, but it makes unrelated information compete for the same terminal cells.

The recent Bubble Tea v2 migration and sprint work changed the shape of the problem. Chisel now has access to the v2 native terminal progress indicator, so the inline sprint bar no longer needs to be the main progress representation.

## UX Model

Chisel should have three levels of interface chrome:

1. **Bottom shelf**
   Always reserved as two rows at the bottom of the main app.

2. **Help layer**
   Opened with `?`. Shows complete keymap and context help. It is an overlay/layer, not normal status text.

3. **Future menu layers**
   Root-owned layered screens for command/menu flows. These may eventually be reached from the help layer, a top menu pattern, a bottom command palette, or a dedicated menu key.

The first implementation should deliver levels 1 and 2 while shaping the code so level 3 can be added cleanly later.

## Bottom Shelf

The bottom shelf reserves exactly two rows in normal app modes:

- **State row:** application state and transient feedback.
- **Hint row:** compact context hints.

The shelf should be stable. Opening a sprint should not change pane height. Changing focus from binder to editor should not cause the body to gain or lose a row.

### State Row

The state row is the place for information about what is happening now:

- temporary status message when present
- file name
- word count
- modified marker
- active structural view context
- sprint time remaining and word gain
- session words or daily goal progress

When a temporary status message is present, it should lead the row. It may suppress less important file state if width is tight, but it should not erase sprint state while a sprint is active.

In normal binder/editor mode, the state row should prioritize the open file context. In structural views, it should identify the active view and its current scope, preserving the context that existing view headers/status text already provide. Examples include `Corkboard - Acts (12 scenes)`, `Outliner - Novel (42 items)`, `Timeline - Novel (19 scenes)`, and `History - chapter-01.md (6 snapshots)`.

Suggested compact examples:

```text
Saved chapter-01.md (842 words)        Sprint 18:42 +12        Today +144/1000
chapter-01.md 842w *                  Sprint 18:42 +12        Today +144/1000
Corkboard - Acts (12 scenes)          Sprint 18:42 +12        Today +144/1000
```

The exact separators can be chosen during implementation, but the row should favor compact state over prose.

### Hint Row

The hint row is for a small number of context-sensitive hints:

```text
Binder: Tab switch  n new  r rename  d delete  ? help
Editor: Tab switch  ^S save  ^E export
Sprint: F7 stop  ? help
```

During an active sprint, the hint row should shrink, not disappear. It should show only the most relevant controls:

```text
F7 stop sprint  ? help
```

This keeps discoverability alive without letting hints compete with sprint state.

## Help Layer

Pressing `?` should open a root-owned help layer whenever a non-text-input state owns the keyboard. The layer owns keys while open and closes with `Esc` or `?`.

The first version should show:

- global keys
- binder keys
- editor keys
- structural view keys
- overlay keys
- sprint and theme keys

The help layer should use the same root dispatch priority model as existing overlays: quick note, search, reader, history, structural views, prompt, and normal dispatch. Its priority must prevent `?` from interfering with text input overlays or editor prose entry.

Recommended behavior:

- `?` opens help from binder focus, structural views, history, and other non-text-input states.
- `?` inserts a literal question mark when the editor textarea owns text input.
- `?` inserts a literal question mark while prompt, search query, or quick-note text input owns the keyboard.
- `Esc` closes help.
- `?` also closes help as a quick toggle.

This preserves the user's ability to write normal prose. In editor focus, users can press `Tab` to return to binder focus and then press `?` for help.

## Menu and Layer Direction

The issue comment calls out TUI menus and the possibilities opened by Bubble Tea v2. The design direction is to treat menus as layers, not as inline status-bar content.

Future menu surfaces may include:

- a top menu bar for persistent categories such as File, Edit, View, Project, Help
- a bottom command palette for command search and quick actions
- a full-screen or centered command/help layer

The first pass should not choose all of these. It should create enough structure that future menu screens can be added without rewriting status rendering or root key dispatch.

The help layer is the first menu-like layer. It should be built as an independent TUI model with its own view/update methods and root-returned actions, matching existing patterns such as history and structural views.

## Prompt and Overlay Behavior

The CRUD prompt and scene-note editor currently occupy the same one-row slot as the status bar. With a two-row shelf:

- Active prompts should replace the shelf area, not add extra height.
- A prompt may use one or both bottom rows depending on content.
- The body height should remain based on the reserved shelf height.
- Quick note and search overlays should render over the full normal view including the two-row shelf.
- Reading mode remains a full-screen takeover with no bottom shelf.

This keeps layout stable while preserving existing overlay ownership rules.

## Layout Rules

The layout system should move from one reserved bottom row to two reserved bottom rows for normal app modes.

Rules:

- Body height is `terminal height - 2` in normal app modes.
- Body height remains at least 1.
- History, corkboard, outliner, and timeline use the body height above the shelf.
- Reader mode still owns the full screen.
- Prompt mode still uses the reserved bottom shelf area.
- All shelf text is width-aware and truncated or compacted before rendering.

## Styling

New shelf styles should live in `tui/styles.go` and be rebuilt by `rebuildStyles()`.

Suggested style split:

- `StatusBarStyle` may remain the state-row style.
- Add a hint-row style if needed, using theme tokens only.
- Avoid hardcoded hex colors outside `styles.go`.
- Keep future light theme in mind by relying on explicit foreground and background colors for both shelf rows.

## Components

### Status Shelf Model

A small renderer/helper should own bottom shelf composition.

Responsibilities:

- accept current root state as inputs
- build state row parts
- build hint row parts
- compact parts based on width and sprint state
- render one or two rows with theme styles

This does not need to become a full Bubble Tea submodel unless the implementation naturally benefits from it. A focused helper is enough for the first pass.

### Help Layer Model

A `helpModel` should own the `?` help screen.

Responsibilities:

- know whether it is active
- render the keymap for the current terminal size
- handle close/toggle keys
- return a root action such as close

The help model should not mutate the root model directly.

## Data Flow

No `core` changes are required.

The root `Model` remains the source of truth for:

- focus
- current view mode
- current file path and word count
- modified state
- status message
- sprint state
- session words and daily goal
- prompt state
- help-layer activity

The shelf renderer receives those values and emits rendered rows. The help model receives size and context and emits a layered view.

## Error Handling

No new I/O is needed for the first pass, so error handling is mostly unchanged.

Possible errors remain existing ones:

- save/export/load failures in status messages
- prompt validation errors
- config save failures on theme cycle

The shelf should make temporary errors prominent in the state row. Help-layer rendering should degrade gracefully at small widths by truncating text and keeping close instructions visible when possible.

## Testing

Tests should cover behavior rather than exact cosmetic strings where possible.

Recommended tests:

- layout reserves two bottom rows in normal modes
- body pane heights sum correctly after the two-row shelf change
- reader mode still uses full-screen behavior without the shelf
- prompt mode uses the reserved shelf area without changing body height
- sprint active state shrinks hint content and keeps sprint state visible
- `?` opens and closes the help layer in non-text-input states
- help layer owns `Esc` and closes without quitting the app
- editor-mode `?` inserts a literal question mark instead of opening help
- existing sprint native progress bar tests continue to pass

## Implementation Boundaries

The first implementation plan should include:

- two-row bottom shelf
- compact state and hint composition
- sprint compacting behavior
- removal or demotion of the inline 12-cell sprint bar
- `?` help layer
- tests for layout, sprint shelf behavior, and help open/close

The first implementation plan should not include:

- full top menu bar
- command palette
- right-panel stats dashboard
- light theme implementation
- broader visual redesign

## Acceptance Criteria

- The app reserves two bottom rows in normal modes.
- During sprint, the native terminal progress bar remains active and the inline status text stays compact.
- During sprint, the hint row shrinks instead of disappearing.
- The full keymap is available through `?` from non-text-input states.
- Existing overlays and structural views still own their keys correctly.
- Reading mode remains a full-screen takeover.
- `core/` remains free of Charm imports.
- Tests demonstrate the new layout and help-layer behavior.
