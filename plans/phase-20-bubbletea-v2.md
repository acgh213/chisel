# Phase 20: Bubble Tea v2 Migration

Closes #54. Unlocks #55, #56, #57.

## Context

chisel is on Bubble Tea v1.3.10. v2.0.7 was released February 2026 and ships under a new vanity import domain (`charm.land`). The headline wins for chisel are faster rendering (the v2 renderer is ncurses-based), automatic synchronized frame delivery (no tearing on rapid typing), native keyboard protocol support via Kitty, and a progress bar primitive that unblocks the sprint timer visualization in #55. The v2 upgrade is a prerequisite for all three follow-on issues.

This plan is scoped to the mechanical migration only — get chisel building and passing tests on v2. Follow-on features (#55 progress bar, #56 Kitty bindings, #57 declarative cursor) ship as separate PRs after this merges.

## Package version targets

| Package | Current | Target |
|---|---|---|
| `github.com/charmbracelet/bubbletea` | v1.3.10 | `charm.land/bubbletea/v2` v2.0.7 |
| `github.com/charmbracelet/bubbles` | v1.0.0 | `charm.land/bubbles/v2` v2.1.0 |
| `github.com/charmbracelet/lipgloss` | v1.1.0 | `charm.land/lipgloss/v2` v2.0.3 |

Note: chisel's binder is a custom implementation (`tui/binder.go`) — it does **not** use `bubbles/tree`. Actual bubbles dependencies are only `textarea` (editor) and `textinput` (prompt, quicknote, search). Both exist in `charm.land/bubbles/v2`.

## Breaking changes and their fixes

### 1. `View()` return type — every component

Every `View()` method changes from returning `string` to returning `tea.View`.

Pattern across all files:
```go
// Before
func (m FooModel) View() string {
    return s
}

// After
func (m FooModel) View() tea.View {
    return tea.NewView(s)
}
```

Files: `tui/model.go`, `tui/binder.go`, `tui/editor.go`, `tui/corkboard.go`, `tui/outliner.go`, `tui/timeline.go`, `tui/history.go`, `tui/search.go`, `tui/quicknote.go`, `tui/reader.go`, `tui/rightpanel.go`, `tui/prompt.go`.

Sub-views that return `string` from internal `view()` helpers (e.g. `promptModel.view()`, `binderModel.View()` called from `model.go`) are fine — only the Bubble Tea interface method `View()` needs updating.

### 2. `tea.WithAltScreen()` removed — `main.go` + `model.go`

`tea.WithAltScreen()` is no longer a program option. Alt screen is declared per-frame in `View()`.

```go
// main.go — Before
p := tea.NewProgram(model, tea.WithAltScreen())

// main.go — After
p := tea.NewProgram(model)
```

```go
// tui/model.go — After, inside Model.View()
func (m Model) View() tea.View {
    // ... build content string ...
    v := tea.NewView(content)
    v.AltScreen = true
    return v
}
```

### 3. Key message type — `tui/model.go`

`tea.KeyMsg` becomes an interface; the concrete press type is `tea.KeyPressMsg`. The `.String()` method is preserved, so all existing `switch msg.String()` / `case "ctrl+s":` dispatch continues to work unchanged.

```go
// Before
case tea.KeyMsg:

// After
case tea.KeyPressMsg:
```

One change in `model.go`. All helper methods (`updateQuickNote`, `updateReader`, etc.) that accept `tea.KeyMsg` also change their parameter type to `tea.KeyPressMsg`.

### 4. `textarea` style fields — `tui/editor.go`

The textarea `Styles` struct is reorganized. Old top-level fields move into a nested `Styles` struct:

```go
// Before
ta.FocusedStyle.Base = lipgloss.NewStyle()
ta.BlurredStyle.Base = lipgloss.NewStyle()
ta.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(ColorHighlight)
ta.BlurredStyle.CursorLine = lipgloss.NewStyle().Background(ColorHighlight)
ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(ColorDim)
ta.BlurredStyle.Placeholder = lipgloss.NewStyle().Foreground(ColorDim)
ta.FocusedStyle.Text = lipgloss.NewStyle().Foreground(ColorFg)
ta.BlurredStyle.Text = lipgloss.NewStyle().Foreground(ColorFg)
ta.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(ColorAccent)
ta.BlurredStyle.Prompt = lipgloss.NewStyle().Foreground(ColorMuted)

// After
s := ta.Styles()
s.Focused.Base = lipgloss.NewStyle()
s.Blurred.Base = lipgloss.NewStyle()
s.Focused.CursorLine = lipgloss.NewStyle().Background(ColorHighlight)
s.Blurred.CursorLine = lipgloss.NewStyle().Background(ColorHighlight)
s.Focused.Placeholder = lipgloss.NewStyle().Foreground(ColorDim)
s.Blurred.Placeholder = lipgloss.NewStyle().Foreground(ColorDim)
s.Focused.Text = lipgloss.NewStyle().Foreground(ColorFg)
s.Blurred.Text = lipgloss.NewStyle().Foreground(ColorFg)
s.Focused.Prompt = lipgloss.NewStyle().Foreground(ColorAccent)
s.Blurred.Prompt = lipgloss.NewStyle().Foreground(ColorMuted)
ta.SetStyles(s)
```

The same getter/setter pattern applies wherever `rebuildStyles()` in `tui/styles.go` touches textarea style fields (check `tui/styles.go` for any direct textarea references).

### 5. `textinput` width and styles — `tui/quicknote.go`, `tui/search.go`

Width is now a setter method rather than a field:

```go
// Before
ti.Width = 50
ti.Width = searchPopupW - 4

// After
ti.SetWidth(50)
ti.SetWidth(searchPopupW - 4)
```

Style fields follow the same nested struct pattern as textarea above (check after compilation errors — only touch what the compiler flags).

### 6. Import path sweep — all TUI files + `main.go`

Every file that imports a Charm package needs its import path updated:

```go
// Before
tea "github.com/charmbracelet/bubbletea"
"github.com/charmbracelet/lipgloss"
"github.com/charmbracelet/bubbles/textarea"
"github.com/charmbracelet/bubbles/textinput"

// After
tea "charm.land/bubbletea/v2"
"charm.land/lipgloss/v2"
"charm.land/bubbles/v2/textarea"
"charm.land/bubbles/v2/textinput"
```

Files: `main.go`, `tui/model.go`, `tui/binder.go`, `tui/editor.go`, `tui/corkboard.go`, `tui/outliner.go`, `tui/timeline.go`, `tui/history.go`, `tui/search.go`, `tui/quicknote.go`, `tui/reader.go`, `tui/rightpanel.go`, `tui/prompt.go`, `tui/styles.go`, and test files.

### 7. `go.mod` update

Run in order:

```sh
go get charm.land/bubbletea/v2@v2.0.7
go get charm.land/lipgloss/v2@v2.0.3
go get charm.land/bubbles/v2@v2.1.0
go mod tidy
```

Then remove the old `github.com/charmbracelet/*` entries manually if `go mod tidy` doesn't drop them automatically (they will have no importers after the import sweep).

## What does NOT change

- `tea.WindowSizeMsg` — still exists in v2, received the same way. All test helpers that send `tea.WindowSizeMsg{Width: ..., Height: ...}` continue to work.
- `lipgloss` layout API — `JoinHorizontal`, `JoinVertical`, `lipgloss.Color`, `lipgloss.NewStyle()` are unchanged in v2. The import path changes but the API does not.
- Key dispatch logic — all `switch msg.String()` / `case "ctrl+s":` chains continue to work because `KeyPressMsg.String()` works identically to the old `KeyMsg.String()`.
- `core/` — zero changes. `core/` has no Charm imports; the hard rule is preserved.
- Sprint timer tick — `tea.Tick(time.Second, ...)` is unchanged.

## Test impact

Tests that call `m.View()` and check the result as a string will need updating: `View()` now returns `tea.View` not `string`. The rendered content is accessed via `tea.NewView(s)` — check whether test assertions should call `.String()` on the returned value or whether there's a v2 idiom for this. Compile errors will surface these directly; no need to preemptively audit.

## Implementation order

1. Update `go.mod` — run `go get` for all three packages, then `go mod tidy`
2. Import path sweep — find-replace across all `.go` files in `tui/` and `main.go`
3. `View()` signatures — change return types and wrap with `tea.NewView()`
4. `main.go` — remove `tea.WithAltScreen()`, add `v.AltScreen = true` in `Model.View()`
5. `model.go` — `tea.KeyMsg` → `tea.KeyPressMsg` (switch case + all helper signatures)
6. `editor.go` — textarea style field renames
7. `quicknote.go`, `search.go` — textinput `SetWidth()`
8. `go build ./...` — fix any remaining compilation errors
9. `go test ./...` — fix test callsites that check `View()` return type

## Follow-on phases (separate PRs, after this merges)

- **#55**: Sprint timer progress bar — `v.ProgressBar = tea.NewProgressBar(tea.ProgressBarDefault, elapsed/sprintDuration)` in `Model.View()` when sprint is active
- **#56**: Kitty keyboard protocol — `v.KeyboardEnhancements = tea.KeyboardEnhancementsAll` in `Model.View()`; add hold-to-repeat in reader mode using `key.IsRepeat`
- **#57** (P4/later): Declarative cursor shapes — `v.Cursor = &tea.Cursor{Shape: tea.CursorBar}` in editor mode vs block in binder

## Verification

```sh
go build ./...          # zero compilation errors
go test ./...           # all tests pass
chisel <test-project>   # TUI launches in alt screen, typing is responsive, themes work
```

Manual smoke test:
- Open a project, type in editor (no tearing, good latency)
- Ctrl+S saves and commits revision
- F2/F3/F4 structural views open and close cleanly
- F7 sprint timer starts and shows countdown
- F8 cycles themes, persists to .chisel.yaml
- Ctrl+Q quits cleanly
