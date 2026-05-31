package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/acgh213/chisel/core"
)

// TestHistoryErrorClearsOnBack guards a regression: an error raised while in the
// diff view (e.g. a failed Diff) must not bleed onto the list view. Going back to
// the list with Esc clears it, since view() checks h.err before h.mode.
func TestHistoryErrorClearsOnBack(t *testing.T) {
	h := historyModel{
		mode: historyDiff,
		err:  "boom: diff failed",
		revs: []core.Revision{{Hash: "deadbeefdeadbeef"}},
	}

	h, action := h.update(tea.KeyMsg{Type: tea.KeyEsc})
	if action != historyNone {
		t.Errorf("Esc in diff mode = %v, want historyNone", action)
	}
	if h.mode != historyList {
		t.Errorf("after Esc, mode = %v, want historyList", h.mode)
	}
	if h.err != "" {
		t.Errorf("after Esc, err = %q, want cleared", h.err)
	}
}

// TestHistoryFlow drives the whole Phase 3 path through the root model with
// simulated keys: open a scene, edit + Ctrl+S twice (two snapshots), Ctrl+H to
// open the browser, then restore the older revision into the editor.
func TestHistoryFlow(t *testing.T) {
	dir := t.TempDir()
	scenePath := filepath.Join(dir, "scene.md")
	if err := os.WriteFile(scenePath, []byte("# Scene\n\nfirst.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m0, err := NewModel(dir)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}

	var m tea.Model = m0
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// Binder is focused with the scene at the cursor — Enter opens it.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Type, save (snapshot #1), type more, save (snapshot #2).
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Y")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})

	// Open the history browser.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlH})
	mm := m.(Model)
	if !mm.showHistory {
		t.Fatal("expected history browser to open after Ctrl+H")
	}
	if len(mm.history.revs) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(mm.history.revs))
	}

	// Compute the body we expect after restoring the older (second-in-list) rev.
	olderHash := mm.history.revs[1].Hash
	rawOlder, err := mm.revBackend.Restore(scenePath, olderHash)
	if err != nil {
		t.Fatalf("Restore (expected value): %v", err)
	}
	wantBody := core.ParseScene(scenePath, rawOlder).Body

	contentNow := mm.editor.Content()
	if contentNow == wantBody {
		t.Fatal("precondition failed: current editor body already equals the older revision")
	}

	// Navigate to the older revision and restore it.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	mm = m.(Model)

	if mm.showHistory {
		t.Error("expected history to close after restore")
	}
	if got := mm.editor.Content(); got != wantBody {
		t.Errorf("after restore, editor body = %q, want %q", got, wantBody)
	}
	if !mm.editor.IsModified() {
		t.Error("restored content should be marked modified (not yet saved)")
	}
	// Sanity: the restored (older) body must not contain the later edit.
	if strings.Contains(mm.editor.Content(), "Y") {
		t.Errorf("restored body unexpectedly contains the later edit: %q", mm.editor.Content())
	}
}
