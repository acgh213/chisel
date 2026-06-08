package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// TestComputeLayoutSumsToWidth is the "panes tile the terminal exactly"
// guarantee, asserted rather than eyeballed. For widths at or above
// minBinderWidth+1 (21), the binder keeps its minimum and the editor takes the
// rest, so the outer widths sum to the terminal width, the panes leave two rows
// for the bottom shelf, and the binder never drops below minBinderWidth. (Below
// that threshold the binder yields columns to the editor — covered by
// TestComputeLayoutSumInvariant.)
func TestComputeLayoutSumsToWidth(t *testing.T) {
	cases := []struct{ w, h int }{
		{80, 24},
		{120, 40},
		{200, 50},
		{61, 30},
		{60, 20},
		{45, 20},
		{30, 20},
		{21, 10}, // exactly minBinderWidth+1, the threshold
	}
	for _, c := range cases {
		l := computeLayout(c.w, c.h, false)
		if l.binderW+l.editorW != c.w {
			t.Errorf("computeLayout(%d,%d): binderW(%d)+editorW(%d)=%d, want %d",
				c.w, c.h, l.binderW, l.editorW, l.binderW+l.editorW, c.w)
		}
		if l.paneH != c.h-2 {
			t.Errorf("computeLayout(%d,%d): paneH=%d, want %d", c.w, c.h, l.paneH, c.h-2)
		}
		if l.binderW < minBinderWidth {
			t.Errorf("computeLayout(%d,%d): binderW=%d below minBinderWidth=%d",
				c.w, c.h, l.binderW, minBinderWidth)
		}
	}
}

// TestComputeLayoutSumInvariant proves the panes tile the full width for every
// width >= 2 — including widths below minBinderWidth, where the editorW<1
// fallback shrinks the binder. (Two 1-wide panes need at least 2 columns;
// widths 0 and 1 are degenerate and only require positive dimensions, which
// TestComputeLayoutClampsTinyTerminal covers.)
func TestComputeLayoutSumInvariant(t *testing.T) {
	for w := 2; w <= 300; w++ {
		l := computeLayout(w, 24, false)
		if l.binderW+l.editorW != w {
			t.Errorf("computeLayout(%d,24): binderW(%d)+editorW(%d)=%d, want %d",
				w, l.binderW, l.editorW, l.binderW+l.editorW, w)
		}
		if l.binderW < 1 || l.editorW < 1 {
			t.Errorf("computeLayout(%d,24): non-positive pane: binderW=%d editorW=%d",
				w, l.binderW, l.editorW)
		}
	}
}

// TestComputeLayoutClampsTinyTerminal makes sure absurdly small (or zero)
// terminal sizes never yield a zero/negative dimension, which is what makes
// lipgloss and the textarea misbehave or panic.
func TestComputeLayoutClampsTinyTerminal(t *testing.T) {
	cases := []struct{ w, h int }{
		{20, 5},
		{10, 3},
		{2, 2},
		{1, 1},
		{0, 0},
	}
	for _, c := range cases {
		l := computeLayout(c.w, c.h, false)
		if l.binderW < 1 || l.editorW < 1 || l.paneH < 1 {
			t.Errorf("computeLayout(%d,%d) produced non-positive dimension: %+v", c.w, c.h, l)
		}
		// Also verify no panics for three-pane at tiny sizes.
		l3 := computeLayout(c.w, c.h, true)
		if l3.binderW < 1 || l3.editorW < 1 || l3.rightPanelW < 1 || l3.paneH < 1 {
			t.Errorf("3-pane computeLayout(%d,%d) produced non-positive dimension: %+v", c.w, c.h, l3)
		}
	}
}

// TestEnterInsertsNewlineInEditor guards the bug where the root model's "enter"
// case swallowed the key while the editor was focused, so you couldn't make a
// new line while writing.
func TestEnterInsertsNewlineInEditor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scene.md")
	if err := os.WriteFile(path, []byte("line one"), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	// Open the scene and focus the editor (cursor lands at start of text).
	if err := m.editor.LoadFile(path); err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	m.editor.Focus(true)
	m.focus = PaneEditor

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got := updated.(Model).editor.Content()
	if !strings.Contains(got, "\n") {
		t.Errorf("Enter in editor did not insert a newline; content = %q", got)
	}
}

// TestViewFitsTerminal renders the real composed View at several terminal
// sizes and asserts the output never overflows: no line wider than the
// terminal, no more lines than its height. This is the regression guard for
// the bordered-pane overflow bug.
func TestViewFitsTerminal(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "scene.md"),
		[]byte("# A Scene\n\nSome prose with several words on a line.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	base, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}

	sizes := []struct{ w, h int }{
		{80, 24},
		{120, 40},
		{200, 50},
		{60, 20},
		{40, 15},
	}
	for _, s := range sizes {
		updated, _ := base.Update(tea.WindowSizeMsg{Width: s.w, Height: s.h})
		view := updated.View().Content
		lines := strings.Split(view, "\n")

		if len(lines) > s.h {
			t.Errorf("%dx%d: view rendered %d lines, exceeds height %d", s.w, s.h, len(lines), s.h)
		}
		for i, ln := range lines {
			if w := lipgloss.Width(ln); w > s.w {
				t.Errorf("%dx%d: line %d width %d exceeds terminal width %d", s.w, s.h, i, w, s.w)
			}
		}
	}
}

func TestBottomShelfRendersStateAndHintRows(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "scene.md"),
		[]byte("# A Scene\n\nSome prose with several words on a line.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	view := m.View().Content
	lines := strings.Split(view, "\n")
	if len(lines) != 24 {
		t.Fatalf("view rendered %d lines, want exactly terminal height 24", len(lines))
	}
	stateRow := lines[len(lines)-2]
	hintRow := lines[len(lines)-1]
	if !strings.Contains(stateRow, "scene.md") || !strings.Contains(stateRow, "words") {
		t.Errorf("state row should show file and word context, got %q", stateRow)
	}
	if !strings.Contains(hintRow, "Editor:") || !strings.Contains(hintRow, "^S save") {
		t.Errorf("hint row should show compact editor hints, got %q", hintRow)
	}
}

func TestSprintShelfUsesCompactStateAndShrunkHints(t *testing.T) {
	dir := twoSceneProject(t)
	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF7})

	view := m.View().Content
	if !strings.Contains(view, "Sprint") {
		t.Fatalf("sprint shelf should include compact sprint state, got:\n%s", view)
	}
	if strings.Contains(view, "█") || strings.Contains(view, "░") {
		t.Errorf("sprint shelf should not include the old inline block progress bar, got:\n%s", view)
	}
	if !strings.Contains(view, "F7 stop sprint") || !strings.Contains(view, "? help") {
		t.Errorf("sprint hint row should shrink to stop/help controls, got:\n%s", view)
	}
}

// TestComputeLayoutThreePaneSumsToWidth asserts the three-pane (right panel
// open) split tiles the terminal exactly and all dimensions are positive for
// typical terminal sizes.
func TestComputeLayoutThreePaneSumsToWidth(t *testing.T) {
	cases := []struct{ w, h int }{
		{80, 24},
		{120, 40},
		{160, 50},
		{200, 50},
	}
	for _, c := range cases {
		l := computeLayout(c.w, c.h, true)
		sum := l.binderW + l.editorW + l.rightPanelW
		if sum != c.w {
			t.Errorf("3-pane computeLayout(%d,%d): %d+%d+%d=%d, want %d",
				c.w, c.h, l.binderW, l.editorW, l.rightPanelW, sum, c.w)
		}
		if l.binderW < 1 || l.editorW < 1 || l.rightPanelW < 1 {
			t.Errorf("3-pane computeLayout(%d,%d): non-positive pane: %+v", c.w, c.h, l)
		}
	}
}

// TestQuickNoteFlow opens the quick-note popup with backtick, types a note,
// confirms with Enter, and checks the popup is dismissed.
func TestQuickNoteFlow(t *testing.T) {
	dir := twoSceneProject(t)
	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// Backtick should open the popup.
	m, _ = m.Update(tea.KeyPressMsg{Code: '`', Text: "`"})
	if mm := m.(Model); !mm.quickNote.active() {
		t.Fatal("backtick should activate the quick-note popup")
	}

	// Type some text.
	for _, r := range "hello world" {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}

	// Enter should save and close.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	mm := m.(Model)
	if mm.quickNote.active() {
		t.Error("quick-note popup should be closed after Enter")
	}
	if !strings.Contains(mm.statusMsg, "saved") {
		t.Errorf("expected status message about saved note, got %q", mm.statusMsg)
	}
}

// TestQuickNoteEscCancels confirms Esc closes without saving.
func TestQuickNoteEscCancels(t *testing.T) {
	dir := twoSceneProject(t)
	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	m, _ = m.Update(tea.KeyPressMsg{Code: '`', Text: "`"})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})

	if mm := m.(Model); mm.quickNote.active() {
		t.Error("quick-note popup should be closed after Esc")
	}
}

func TestQuestionMarkOpensAndClosesHelpFromBinder(t *testing.T) {
	dir := twoSceneProject(t)
	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	m, _ = m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	mm := m.(Model)
	if !mm.help.active() {
		t.Fatal("? should open help when binder focus owns the keyboard")
	}
	if view := m.View().Content; !strings.Contains(view, "Chisel Help") || !strings.Contains(view, "Global") {
		t.Errorf("help view should render keymap content, got:\n%s", view)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	mm = m.(Model)
	if mm.help.active() {
		t.Error("Esc should close help")
	}
	if mm.quitting {
		t.Error("Esc from help must not quit the app")
	}
}

func TestQuestionMarkOpensHelpFromStructuralView(t *testing.T) {
	dir := twoSceneProject(t)
	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF2})
	m, _ = m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})

	mm := m.(Model)
	if !mm.help.active() {
		t.Fatal("? should open help from structural views")
	}
	if mm.viewMode != viewCorkboard {
		t.Errorf("opening help should preserve structural view mode, got %v", mm.viewMode)
	}
}

func TestQuestionMarkInEditorInsertsLiteralCharacter(t *testing.T) {
	dir := twoSceneProject(t)
	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	mm := m.(Model)
	if mm.focus != PaneEditor {
		t.Fatalf("opening a scene should focus editor, got %v", mm.focus)
	}
	before := mm.editor.Content()

	m, _ = m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	mm = m.(Model)
	if mm.help.active() {
		t.Fatal("? in editor focus should not open help")
	}
	if got := mm.editor.Content(); got == before || !strings.Contains(got, "?") {
		t.Errorf("? in editor focus should insert literal question mark; before=%q after=%q", before, got)
	}
}

// TestQuickNoteOpensFromStructuralView confirms backtick works even when a
// structural view (corkboard, outliner, timeline) is the active mode.
func TestQuickNoteOpensFromStructuralView(t *testing.T) {
	dir := twoSceneProject(t)
	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// Open corkboard first.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF2})
	if mm := m.(Model); mm.viewMode != viewCorkboard {
		t.Fatal("expected corkboard view")
	}

	// Backtick should open the quick-note popup even from corkboard.
	m, _ = m.Update(tea.KeyPressMsg{Code: '`', Text: "`"})
	if mm := m.(Model); !mm.quickNote.active() {
		t.Error("backtick should open quick-note from structural view")
	}
}

// TestComputeLayoutThreePaneSumInvariant checks the three-pane sum holds for
// all widths >= 3. Below 3 columns the sum exceeds width by design: three
// panes each floored to 1 must sum to at least 3, so the invariant cannot
// hold on a < 3-column terminal. The clamp test covers those widths separately.
func TestComputeLayoutThreePaneSumInvariant(t *testing.T) {
	for w := 3; w <= 300; w++ {
		l := computeLayout(w, 24, true)
		sum := l.binderW + l.editorW + l.rightPanelW
		if sum != w {
			t.Errorf("3-pane computeLayout(%d,24): sum %d+%d+%d=%d, want %d",
				w, l.binderW, l.editorW, l.rightPanelW, sum, w)
		}
		if l.binderW < 1 || l.editorW < 1 || l.rightPanelW < 1 {
			t.Errorf("3-pane computeLayout(%d,24): non-positive: binderW=%d editorW=%d rightW=%d",
				w, l.binderW, l.editorW, l.rightPanelW)
		}
	}
}

// TestWTogglesNoteModeInRightPanel opens the right panel, presses W from binder
// focus, and checks that the panel enters Scene Notes mode then returns to World
// Index on a second W press.
func TestWTogglesNoteModeInRightPanel(t *testing.T) {
	dir := twoSceneProject(t)
	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	// Open right panel with F5.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF5})
	mm := m.(Model)
	if !mm.showRightPanel {
		t.Fatal("F5 should open right panel")
	}
	if mm.rightPanel.noteMode {
		t.Fatal("note mode should be off initially")
	}

	// W from binder focus should enter note mode.
	m, _ = m.Update(tea.KeyPressMsg{Code: 'W', Text: "W"})
	mm = m.(Model)
	if !mm.rightPanel.noteMode {
		t.Error("W should toggle note mode on")
	}

	// Second W should toggle back.
	m, _ = m.Update(tea.KeyPressMsg{Code: 'W', Text: "W"})
	mm = m.(Model)
	if mm.rightPanel.noteMode {
		t.Error("second W should toggle note mode off")
	}
}

// TestWDoesNotToggleWhenEditorFocused verifies that W in the editor inserts a
// literal 'W' rather than toggling the right panel.
func TestWDoesNotToggleWhenEditorFocused(t *testing.T) {
	dir := twoSceneProject(t)
	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}

	// Open the first scene and switch focus to editor.
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF5})  // open panel
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab}) // focus editor
	mm := m.(Model)
	if mm.focus != PaneEditor {
		t.Skip("could not move focus to editor — skipping")
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: 'W', Text: "W"})
	mm = m.(Model)
	if mm.rightPanel.noteMode {
		t.Error("W in editor focus should not toggle note mode")
	}
}

// TestEOpensNotePromptInNoteMode checks that pressing e while in note mode and
// binder focus opens the note edit prompt.
func TestEOpensNotePromptInNoteMode(t *testing.T) {
	dir := twoSceneProject(t)
	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF5})      // open panel
	m, _ = m.Update(tea.KeyPressMsg{Code: 'W', Text: "W"}) // enter note mode

	// e should open the note prompt.
	m, _ = m.Update(tea.KeyPressMsg{Code: 'e', Text: "e"})
	mm := m.(Model)
	if mm.prompt.mode != promptNote {
		t.Errorf("e in note mode should open promptNote, got mode %d", mm.prompt.mode)
	}
}

// TestNoteRoutesThroughEditorWhenFileOpen verifies that saving a note for the
// currently open editor scene updates the in-memory scene rather than writing
// to disk, preventing clobber of unsaved body edits.
func TestNoteRoutesThroughEditorWhenFileOpen(t *testing.T) {
	dir := twoSceneProject(t)
	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	// Select the first scene in the binder and open it.
	mm := m.(Model)
	path := mm.binder.SelectedFile()
	if path == "" {
		t.Skip("no file selected — skipping")
	}
	if err := mm.editor.LoadFile(path); err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	m = mm

	// Open panel, toggle note mode, open edit prompt.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF5})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'W', Text: "W"})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'e', Text: "e"})

	// Type a note and confirm.
	for _, ch := range "my inline note" {
		m, _ = m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	mm = m.(Model)
	if mm.editor.Notes() != "my inline note" {
		t.Errorf("editor.Notes() = %q, want %q", mm.editor.Notes(), "my inline note")
	}
	if !mm.editor.IsModified() {
		t.Error("editor should be marked modified after SetNotes")
	}
}

// TestSearchOpenClose opens the search overlay with Ctrl+F and closes it with Esc.
func TestSearchOpenClose(t *testing.T) {
	dir := twoSceneProject(t)
	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	// Ctrl+F should open the search overlay.
	m, _ = m.Update(tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl})
	if mm := m.(Model); !mm.search.active() {
		t.Fatal("Ctrl+F should activate the search overlay")
	}

	// Esc should close it.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if mm := m.(Model); mm.search.active() {
		t.Error("Esc should close the search overlay")
	}
}

// TestSearchFlowOpensScene types a query, runs the search, and opens a result.
func TestSearchFlowOpensScene(t *testing.T) {
	dir := twoSceneProject(t) // a.md body: "one two three"; b.md body: "just four words here"
	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	// Open search.
	m, _ = m.Update(tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl})

	// Type "three" — only a.md body contains it.
	for _, r := range "three" {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}

	// Enter runs the search and should switch to browse mode.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	mm := m.(Model)
	if !mm.search.active() {
		t.Fatal("search should still be active after running a search with results")
	}
	if mm.search.mode != searchBrowsing {
		t.Fatalf("search mode should be searchBrowsing after Enter with results, got %v", mm.search.mode)
	}
	if len(mm.search.results) == 0 {
		t.Fatal("expected at least one search result for 'three'")
	}

	// Enter again opens the selected scene.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	mm = m.(Model)
	if mm.search.active() {
		t.Error("search overlay should be closed after opening a result")
	}
	if mm.editor.FilePath() == "" {
		t.Error("editor should have a file open after selecting a search result")
	}
}

// TestReaderOpenCloseNoScene confirms F6 with no scene open shows a status
// message instead of activating reading mode.
func TestReaderOpenCloseNoScene(t *testing.T) {
	dir := twoSceneProject(t)
	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	// F6 with no open scene should show a status message, not open reader.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF6})
	mm := m.(Model)
	if mm.reader.active() {
		t.Error("reader should not activate when no scene is open")
	}
	if mm.statusMsg == "" {
		t.Error("expected a status message when F6 pressed with no scene open")
	}
}

// TestReaderOpenCloseWithScene opens a scene, activates reading mode with F6,
// scrolls, and exits with Esc.
func TestReaderOpenCloseWithScene(t *testing.T) {
	dir := twoSceneProject(t)
	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	// Open the first scene.
	mm := m.(Model)
	path := mm.binder.SelectedFile()
	if path == "" {
		t.Skip("no file selected")
	}
	if err := mm.editor.LoadFile(path); err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	m = mm

	// F6 should activate reading mode.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF6})
	mm = m.(Model)
	if !mm.reader.active() {
		t.Fatal("F6 should activate reading mode when a scene is open")
	}
	if len(mm.reader.lines) == 0 {
		t.Error("reader should have content lines when scene is loaded")
	}

	// Arrow keys should scroll (j = down) without exiting.
	m, _ = m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	if mm2 := m.(Model); !mm2.reader.active() {
		t.Error("j should not exit reading mode")
	}

	// Esc should exit reading mode.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if mm2 := m.(Model); mm2.reader.active() {
		t.Error("Esc should close reading mode")
	}
}

// TestReaderF6ExitsReader confirms pressing F6 again while in reading mode
// closes it (same as Esc).
func TestReaderF6ExitsReader(t *testing.T) {
	dir := twoSceneProject(t)
	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	mm := m.(Model)
	path := mm.binder.SelectedFile()
	if path == "" {
		t.Skip("no file selected")
	}
	if err := mm.editor.LoadFile(path); err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	m = mm

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF6}) // open
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF6}) // close
	if mm2 := m.(Model); mm2.reader.active() {
		t.Error("second F6 should close reading mode")
	}
}

// TestReaderOpensFromStructuralView confirms F6 opens reading mode even when a
// structural view (corkboard) is active.
func TestReaderOpensFromStructuralView(t *testing.T) {
	dir := twoSceneProject(t)
	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	// Load a scene so F6 has something to show.
	mm := m.(Model)
	path := mm.binder.SelectedFile()
	if path == "" {
		t.Skip("no file selected")
	}
	if err := mm.editor.LoadFile(path); err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	m = mm

	// Switch to corkboard.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF2})
	if mm2 := m.(Model); mm2.viewMode != viewCorkboard {
		t.Fatal("expected corkboard view")
	}

	// F6 should still open reading mode.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF6})
	if mm2 := m.(Model); !mm2.reader.active() {
		t.Error("F6 should open reading mode from structural view when scene is loaded")
	}
}

// TestThemeCycling cycles through all four themes with F8 and confirms the
// model's theme field changes each press, returning to the start after four presses.
func TestThemeCycling(t *testing.T) {
	defer ApplyTheme("peach") // restore global state after test

	dir := twoSceneProject(t)
	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	want := []string{"forest", "ocean", "midnight", "peach"}
	for i, expected := range want {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF8})
		mm := m.(Model)
		if mm.theme != expected {
			t.Errorf("after %d F8: theme = %q, want %q", i+1, mm.theme, expected)
		}
	}
}

// TestSessionWordCountAccumulates opens a scene, types new words, saves, and
// confirms sessionWords is positive.
func TestSessionWordCountAccumulates(t *testing.T) {
	dir := twoSceneProject(t)
	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	// Open the first scene via Enter in the binder; openScene focuses the editor.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	mm := m.(Model)
	if mm.editor.FilePath() == "" {
		t.Skip("no file opened — skipping")
	}

	// Editor is already focused. Type extra words.
	for _, r := range " extra words here" {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}

	// Save and check that session words increased.
	m, _ = m.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	mm = m.(Model)
	if mm.sessionWords <= 0 {
		t.Errorf("sessionWords = %d after saving with new words, want > 0", mm.sessionWords)
	}
}

// TestSprintStartStop toggles the sprint timer on and off with F7.
func TestSprintStartStop(t *testing.T) {
	dir := twoSceneProject(t)
	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	// F7 should start the sprint.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF7})
	mm := m.(Model)
	if !mm.sprintActive {
		t.Fatal("F7 should activate the sprint timer")
	}
	if mm.sprintEnd.IsZero() {
		t.Error("sprintEnd should be set when sprint is active")
	}

	// F7 again should stop it.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF7})
	mm = m.(Model)
	if mm.sprintActive {
		t.Error("second F7 should stop the sprint timer")
	}
}

// TestReaderResizeUpdatesVisible confirms that a WindowSizeMsg while reading mode
// is active recomputes r.visible so lines don't overflow or under-fill.
func TestReaderResizeUpdatesVisible(t *testing.T) {
	dir := twoSceneProject(t)
	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Open a scene.
	mm := m.(Model)
	path := mm.binder.SelectedFile()
	if path == "" {
		t.Skip("no file selected")
	}
	if err := mm.editor.LoadFile(path); err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	m = mm

	// Open reader at height 40.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF6})
	mm = m.(Model)
	if !mm.reader.active() {
		t.Fatal("reader should be active after F6")
	}
	visibleAt40 := mm.reader.visible

	// Resize to height 20 — visible should shrink.
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 20})
	mm = m.(Model)
	if mm.reader.visible >= visibleAt40 {
		t.Errorf("reader.visible did not shrink after terminal height reduced: was %d at h=40, got %d at h=20",
			visibleAt40, mm.reader.visible)
	}

	// Resize back to 40 — visible should return.
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	mm = m.(Model)
	if mm.reader.visible != visibleAt40 {
		t.Errorf("reader.visible after growing back: got %d, want %d", mm.reader.visible, visibleAt40)
	}
}

// TestSprintTimerExpiry sends a sprintTickMsg that arrives after the end time
// and confirms the sprint is stopped with a status message.
func TestSprintTimerExpiry(t *testing.T) {
	dir := twoSceneProject(t)
	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	// Start sprint.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF7})
	mm := m.(Model)
	if !mm.sprintActive {
		t.Fatal("sprint should be active after F7")
	}

	// Move sprintEnd into the past so the next tick fires expiry.
	mm.sprintEnd = mm.sprintEnd.Add(-26 * 60 * 1e9) // subtract 26 minutes
	m = mm

	// Deliver a sprintTickMsg — model should stop the sprint.
	m, _ = m.Update(sprintTickMsg{})
	mm = m.(Model)
	if mm.sprintActive {
		t.Error("sprint should be inactive after expiry tick")
	}
	if !strings.Contains(mm.statusMsg, "Sprint done") {
		t.Errorf("expected 'Sprint done' status after expiry, got %q", mm.statusMsg)
	}
}

// TestSprintGainIncludesUnsavedWords confirms that stopping the sprint with F7
// counts words typed but not yet saved (bug: gain was stale before accumulateSessionWords was called).
func TestSprintGainIncludesUnsavedWords(t *testing.T) {
	dir := twoSceneProject(t)
	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	// Open the first scene.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	mm := m.(Model)
	if mm.editor.FilePath() == "" {
		t.Skip("no file opened")
	}

	// Start sprint.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF7})
	mm = m.(Model)
	if !mm.sprintActive {
		t.Fatal("sprint should be active after F7")
	}

	// Type words WITHOUT saving.
	for _, r := range " one two three four five" {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}

	// Stop sprint — gain must include the unsaved words.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF7})
	mm = m.(Model)
	if mm.sprintActive {
		t.Fatal("sprint should be inactive after second F7")
	}
	if !strings.Contains(mm.statusMsg, "+") {
		t.Errorf("sprint stop status should report word gain, got %q", mm.statusMsg)
	}
	// The status message should not say "+0 words" since we typed 5 words.
	if strings.Contains(mm.statusMsg, "+0 words") {
		t.Errorf("sprint gain was 0 — unsaved words not counted: %q", mm.statusMsg)
	}
}

// TestSprintProgressBar verifies that View() attaches ProgressBarDefault during
// an active sprint and ProgressBarNone after the sprint stops.
func TestSprintProgressBar(t *testing.T) {
	dir := twoSceneProject(t)
	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	// Before sprint: bar should be ProgressBarNone (clear state).
	v := m.(Model).View()
	if v.ProgressBar == nil {
		t.Fatal("ProgressBar should always be set (None when idle)")
	}
	if v.ProgressBar.State != tea.ProgressBarNone {
		t.Errorf("idle ProgressBar.State = %v, want ProgressBarNone", v.ProgressBar.State)
	}

	// Start sprint.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF7})
	v = m.(Model).View()
	if v.ProgressBar == nil {
		t.Fatal("ProgressBar must not be nil when sprint is active")
	}
	if v.ProgressBar.State != tea.ProgressBarDefault {
		t.Errorf("active sprint ProgressBar.State = %v, want ProgressBarDefault", v.ProgressBar.State)
	}
	if v.ProgressBar.Value <= 0 || v.ProgressBar.Value > 100 {
		t.Errorf("active sprint ProgressBar.Value = %d, want in (0, 100]", v.ProgressBar.Value)
	}

	// Stop sprint.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF7})
	v = m.(Model).View()
	if v.ProgressBar == nil {
		t.Fatal("ProgressBar should always be set (None after stop)")
	}
	if v.ProgressBar.State != tea.ProgressBarNone {
		t.Errorf("stopped sprint ProgressBar.State = %v, want ProgressBarNone", v.ProgressBar.State)
	}
}

// TestSearchOpensFromStructuralView confirms Ctrl+F works even when a structural
// view (corkboard, outliner, timeline) is the active mode.
func TestSearchOpensFromStructuralView(t *testing.T) {
	dir := twoSceneProject(t)
	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	// Open corkboard first.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF2})
	if mm := m.(Model); mm.viewMode != viewCorkboard {
		t.Fatal("expected corkboard view")
	}

	// Ctrl+F should open the search overlay from inside the corkboard.
	m, _ = m.Update(tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl})
	if mm := m.(Model); !mm.search.active() {
		t.Error("Ctrl+F should open search from a structural view")
	}
}
