package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TestComputeLayoutSumsToWidth is the "panes tile the terminal exactly"
// guarantee, asserted rather than eyeballed. For widths at or above
// minBinderWidth+1 (21), the binder keeps its minimum and the editor takes the
// rest, so the outer widths sum to the terminal width, the panes leave one row
// for the status bar, and the binder never drops below minBinderWidth. (Below
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
		if l.paneH != c.h-1 {
			t.Errorf("computeLayout(%d,%d): paneH=%d, want %d", c.w, c.h, l.paneH, c.h-1)
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

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
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
		view := updated.View()
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
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("`")})
	if mm := m.(Model); !mm.quickNote.active() {
		t.Fatal("backtick should activate the quick-note popup")
	}

	// Type some text.
	for _, r := range "hello world" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Enter should save and close.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
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

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("`")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if mm := m.(Model); mm.quickNote.active() {
		t.Error("quick-note popup should be closed after Esc")
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
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyF2})
	if mm := m.(Model); mm.viewMode != viewCorkboard {
		t.Fatal("expected corkboard view")
	}

	// Backtick should open the quick-note popup even from corkboard.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("`")})
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
