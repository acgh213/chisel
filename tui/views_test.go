package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// twoSceneProject writes two scenes with frontmatter and returns the dir. The
// scenes carry an explicit draft_order so their on-corkboard order is known.
func twoSceneProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	a := "---\ntitle: Alpha\nstatus: revised\nsynopsis: First scene.\nword_target: 100\ndraft_order: 1\n---\none two three\n"
	b := "---\ntitle: Beta\nstatus: draft\nsynopsis: Second scene.\ndraft_order: 2\n---\njust four words here\n"
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte(a), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.md"), []byte(b), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestBinderFocusedAtStartup guards the Phase-4 startup fix: the binder is
// focused immediately, so j/k move the tree cursor before any Tab is pressed.
func TestBinderFocusedAtStartup(t *testing.T) {
	m0, err := NewModel(twoSceneProject(t))
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	if mm := m.(Model); !mm.binder.focus {
		t.Fatal("binder should be focused right after startup")
	}

	// j must move the binder cursor without a preceding Tab.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if mm := m.(Model); mm.binder.cursor != 1 {
		t.Errorf("after j at startup, binder cursor = %d, want 1", mm.binder.cursor)
	}
}

// TestCorkboardFlow opens the corkboard (F2), checks it reflects the scene
// metadata in reading order, navigates to the second card, and opens it.
func TestCorkboardFlow(t *testing.T) {
	dir := twoSceneProject(t)
	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// F2 -> corkboard.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyF2})
	mm := m.(Model)
	if mm.viewMode != viewCorkboard {
		t.Fatalf("after F2, viewMode = %v, want corkboard", mm.viewMode)
	}
	if len(mm.corkboard.cards) != 2 {
		t.Fatalf("corkboard has %d cards, want 2", len(mm.corkboard.cards))
	}
	// draft_order 1 (Alpha) then 2 (Beta).
	if mm.corkboard.cards[0].Title != "Alpha" || mm.corkboard.cards[1].Title != "Beta" {
		t.Errorf("card order = [%q, %q], want [Alpha, Beta]",
			mm.corkboard.cards[0].Title, mm.corkboard.cards[1].Title)
	}
	if mm.corkboard.cards[0].WordCount != 3 {
		t.Errorf("Alpha word count = %d, want 3", mm.corkboard.cards[0].WordCount)
	}

	// Navigate to the second card and open it.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm = m.(Model)

	if mm.viewMode != viewMain {
		t.Errorf("after opening a card, viewMode = %v, want main", mm.viewMode)
	}
	if mm.focus != PaneEditor {
		t.Error("opening a card should focus the editor")
	}
	if got := filepath.Base(mm.editor.FilePath()); got != "b.md" {
		t.Errorf("opened %q, want b.md", got)
	}
}

// TestOutlinerFlow opens the outliner (F3), confirms it lists the scenes, and
// opens one with Enter.
func TestOutlinerFlow(t *testing.T) {
	dir := twoSceneProject(t)
	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyF3})
	mm := m.(Model)
	if mm.viewMode != viewOutliner {
		t.Fatalf("after F3, viewMode = %v, want outliner", mm.viewMode)
	}
	if len(mm.outliner.flat) != 2 {
		t.Fatalf("outliner has %d rows, want 2", len(mm.outliner.flat))
	}

	// Enter on the first row opens that scene.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm = m.(Model)
	if mm.viewMode != viewMain {
		t.Errorf("after Enter, viewMode = %v, want main", mm.viewMode)
	}
	if mm.editor.FilePath() == "" {
		t.Error("Enter in outliner should have opened a scene")
	}
}

// TestStructuralViewsFitTerminal renders the corkboard and outliner at several
// sizes and asserts the output never exceeds the terminal — same guarantee as
// the main view, extended to the Phase-4 views.
func TestStructuralViewsFitTerminal(t *testing.T) {
	dir := twoSceneProject(t)
	sizes := []struct{ w, h int }{{80, 24}, {120, 40}, {60, 20}, {40, 15}}
	keys := []tea.KeyType{tea.KeyF2, tea.KeyF3}

	for _, k := range keys {
		for _, s := range sizes {
			m0, err := NewModel(dir)
			if err != nil {
				t.Fatalf("NewModel: %v", err)
			}
			var m tea.Model = m0
			m, _ = m.Update(tea.WindowSizeMsg{Width: s.w, Height: s.h})
			m, _ = m.Update(tea.KeyMsg{Type: k})

			view := m.View()
			lines := strings.Split(view, "\n")
			if len(lines) > s.h {
				t.Errorf("key %v %dx%d: %d lines, exceeds height %d", k, s.w, s.h, len(lines), s.h)
			}
			for _, line := range lines {
				if w := lipgloss.Width(line); w > s.w {
					t.Errorf("key %v %dx%d: line width %d exceeds %d", k, s.w, s.h, w, s.w)
				}
			}
		}
	}
}
