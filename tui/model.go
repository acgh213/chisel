package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/acgh213/chisel/core"
)

// Pane indicates which pane has focus.
type Pane int

const (
	PaneBinder Pane = iota
	PaneEditor
)

// viewMode selects the top-level layout. viewMain is the binder+editor split;
// the structural views (corkboard, outliner) replace it full-width until closed.
type viewMode int

const (
	viewMain viewMode = iota
	viewCorkboard
	viewOutliner
	viewTimeline
)

// minBinderWidth is the narrowest the binder pane may shrink to before the
// editor starts giving up columns. Keeps the tree readable on small terminals.
const minBinderWidth = 20

// minRightPanelWidth is the narrowest the right panel may be rendered.
const minRightPanelWidth = 28

// layoutSizes holds the OUTER box dimensions (including each pane's border)
// for the binder, editor, and optional right panel, plus the shared pane height.
type layoutSizes struct {
	binderW      int
	editorW      int
	rightPanelW  int // 0 when the right panel is not shown
	paneH        int
}

// computeLayout splits a terminal of the given size into pane dimensions. It
// is pure (no model state) so it can be unit-tested without a running terminal.
// When showRight is false, the two-pane split (binder+editor) is used and
// rightPanelW is 0. When showRight is true, a three-pane split is computed.
// All widths are OUTER box sizes; they sum to width for terminals wide enough
// to satisfy the minimums. Every dimension is floored at 1 so a tiny terminal
// can never produce a zero/negative box.
func computeLayout(width, height int, showRight bool) layoutSizes {
	paneH := height - 1
	if paneH < 1 {
		paneH = 1
	}

	if !showRight {
		// Two-pane: binder | editor.
		binderW := width / 3
		if binderW < minBinderWidth {
			binderW = minBinderWidth
		}
		editorW := width - binderW
		if editorW < 1 {
			binderW = width - 1
			editorW = 1
		}
		if binderW < 1 {
			binderW = 1
		}
		if editorW < 1 {
			editorW = 1
		}
		return layoutSizes{binderW: binderW, editorW: editorW, paneH: paneH}
	}

	// Three-pane: binder | editor | right panel.
	binderW := width / 5
	if binderW < minBinderWidth {
		binderW = minBinderWidth
	}
	rightW := width / 4
	if rightW < minRightPanelWidth {
		rightW = minRightPanelWidth
	}
	editorW := width - binderW - rightW

	if editorW < 1 {
		// Narrow terminal: shrink right panel first, then binder.
		editorW = 1
		rightW = width - binderW - editorW
		if rightW < 1 {
			rightW = 1
			binderW = width - editorW - rightW
			if binderW < 1 {
				binderW = 1
			}
		}
	}
	if binderW < 1 {
		binderW = 1
	}
	if editorW < 1 {
		editorW = 1
	}
	if rightW < 1 {
		rightW = 1
	}
	// Note: at width < 3, three panes each floored to 1 sum to 3 > width.
	// This is mathematically unavoidable — three positive integers cannot sum
	// to less than 3 — and only affects unusably narrow terminals (< 3 cols).
	return layoutSizes{binderW: binderW, editorW: editorW, rightPanelW: rightW, paneH: paneH}
}

// Model is the root bubbletea model for chisel.
type Model struct {
	binder      BinderModel
	editor      EditorModel
	focus       Pane
	root        string
	width       int
	height      int
	quitting    bool
	pendingQuit bool   // true after first quit attempt with unsaved changes
	statusMsg   string // temporary status message (e.g., "Saved.")
	statusTimer int    // ticks remaining for status message

	// Revision history (Phase 3). The backend is opened lazily on first save or
	// history view; history overlays the panes when showHistory is true.
	revBackend  core.RevisionBackend
	history     historyModel
	showHistory bool

	// Structural views (Phase 4+). viewMode picks which is shown; each view is
	// loaded on entry and owns all keys while active.
	viewMode  viewMode
	corkboard corkboardModel
	outliner  outlinerModel
	timeline  timelineModel

	// pandocPath is the resolved path to the pandoc binary, or "" if not
	// found. Detected once in NewModel; gates the .docx export offer.
	pandocPath string

	// prompt is the inline bottom-bar input for binder CRUD (new, rename, delete).
	prompt binderPrompt

	// Right panel (Phase 8). showRightPanel toggles the panel; the panel itself
	// is binder-driven (it reflects the current selection, no independent focus).
	showRightPanel bool
	rightPanel     rightPanelModel

	// Quick-note popup (Phase 11). Backtick opens from any state; the popup
	// owns all keys while active and is checked before all other dispatch.
	quickNote quickNoteModel
}

// NewModel creates a new chisel root model for the given project directory.
func NewModel(root string) (Model, error) {
	binder := NewBinder(root)
	if err := binder.Refresh(); err != nil {
		return Model{}, fmt.Errorf("reading project directory: %w", err)
	}
	// The binder starts focused (focus defaults to PaneBinder), so its view
	// state must agree — otherwise the tree ignores j/k until the first Tab.
	binder.Focus(true)

	pandocPath, _ := exec.LookPath("pandoc")

	return Model{
		binder:     binder,
		editor:     NewEditor(),
		focus:      PaneBinder,
		root:       root,
		pandocPath: pandocPath,
		prompt:     newBinderPrompt(),
		rightPanel: newRightPanel(root),
		quickNote:  newQuickNote(),
	}, nil
}

// Init initializes the root model. Alt-screen is enabled via tea.WithAltScreen
// in main, so it is not requested again here.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.binder.Init(),
		m.editor.Init(),
	)
}

// Update handles all messages for the root model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Backtick opens the quick-note popup from any state.
		if msg.String() == "`" && !m.quickNote.active() {
			cmd := m.quickNote.open()
			return m, cmd
		}
		// When the quick-note popup is open it owns all keys.
		if m.quickNote.active() {
			return m.updateQuickNote(msg)
		}
		// When the history browser is open it owns all keys.
		if m.showHistory {
			return m.updateHistory(msg)
		}
		// A structural view (corkboard/outliner/timeline) likewise owns all keys.
		if m.viewMode != viewMain {
			return m.updateView(msg)
		}
		// An active binder prompt owns all keys — must precede esc/quit handling
		// so Esc cancels the prompt instead of quitting the app.
		if m.prompt.active() {
			return m.updatePrompt(msg)
		}

		// Capture the pending-quit state before the switch so the
		// "press Ctrl+Q again" guard reads the *previous* key press.
		// pendingQuit is cleared in the default case and on status expiry.
		wasPending := m.pendingQuit

		switch msg.String() {
		case "ctrl+q", "esc":
			if m.editor.IsModified() && !wasPending {
				m.pendingQuit = true
				m.statusMsg = "Unsaved changes! Press Ctrl+Q again to quit without saving."
				m.statusTimer = 3
				cmds = append(cmds, statusTick())
				return m, tea.Batch(cmds...)
			}
			m.quitting = true
			return m, tea.Quit

		case "tab":
			if m.focus == PaneBinder {
				m.focus = PaneEditor
				m.binder.Focus(false)
				m.editor.Focus(true)
				// Init() == textarea.Blink: arm the cursor blink on focus gain.
				cmds = append(cmds, m.editor.Init())
			} else {
				m.focus = PaneBinder
				m.editor.Focus(false)
				m.binder.Focus(true)
				// Sync the panel when focus returns to the binder — the editor
				// may have saved a character file while focused, changing what
				// the panel should display.
				m.syncRightPanel()
			}

		case "enter":
			if m.focus == PaneBinder {
				path := m.binder.SelectedFile()
				if path != "" {
					cmds = append(cmds, m.openScene(path))
					m.syncRightPanel()
				} else {
					// A folder (or nothing) is selected — let the binder
					// toggle it open/closed.
					var cmd tea.Cmd
					m.binder, cmd = m.binder.Update(msg)
					cmds = append(cmds, cmd)
					m.syncRightPanel()
				}
			} else {
				// Editor focused — Enter must insert a newline, not be
				// swallowed here.
				var cmd tea.Cmd
				m.editor, cmd = m.editor.Update(msg)
				cmds = append(cmds, cmd)
			}

		case "ctrl+s":
			if m.editor.FilePath() != "" {
				if err := m.editor.Save(); err != nil {
					m.statusMsg = fmt.Sprintf("Error saving: %v", err)
				} else {
					path := m.editor.FilePath()
					words := m.editor.WordCount()
					m.statusMsg = fmt.Sprintf("Saved %s (%d words)", filepath.Base(path), words)
					// Snapshot the save. A snapshot failure is non-fatal — the
					// file is already saved; we just note it in the status bar.
					commitMsg := fmt.Sprintf("scene: %s — %d words", filepath.Base(path), words)
					if serr := m.snapshot(path, commitMsg); serr != nil {
						m.statusMsg = fmt.Sprintf("Saved %s (snapshot failed: %v)", filepath.Base(path), serr)
					}
					// Refresh the panel — the saved file may be a character
					// whose display details just changed.
					m.syncRightPanel()
				}
				m.statusTimer = 3
				cmds = append(cmds, statusTick())
			} else {
				m.statusMsg = "No file open to save."
				m.statusTimer = 2
				cmds = append(cmds, statusTick())
			}

		case "ctrl+h":
			if m.editor.FilePath() != "" {
				if err := m.openHistory(); err != nil {
					m.statusMsg = fmt.Sprintf("Error opening history: %v", err)
					m.statusTimer = 3
					cmds = append(cmds, statusTick())
				}
			} else {
				m.statusMsg = "Open a scene first to view its history."
				m.statusTimer = 2
				cmds = append(cmds, statusTick())
			}

		case "ctrl+n", "n":
			if msg.String() == "n" && m.focus != PaneBinder {
				// 'n' in the editor inserts a literal 'n'.
				var cmd tea.Cmd
				m.editor, cmd = m.editor.Update(msg)
				cmds = append(cmds, cmd)
				break
			}
			dir := m.binder.CurrentDir()
			m.prompt.open(promptNewFile, dir, "New scene:", "my-scene")

		case "N":
			if m.focus != PaneBinder {
				var cmd tea.Cmd
				m.editor, cmd = m.editor.Update(msg)
				cmds = append(cmds, cmd)
				break
			}
			dir := m.binder.CurrentDir()
			m.prompt.open(promptNewFolder, dir, "New folder:", "chapter-1")

		case "r":
			if m.focus != PaneBinder {
				var cmd tea.Cmd
				m.editor, cmd = m.editor.Update(msg)
				cmds = append(cmds, cmd)
				break
			}
			if node := m.binder.SelectedNode(); node != nil {
				m.prompt.open(promptRename, node.Path,
					fmt.Sprintf("Rename '%s' to:", node.Name), node.Name)
			}

		case "d":
			if m.focus != PaneBinder {
				var cmd tea.Cmd
				m.editor, cmd = m.editor.Update(msg)
				cmds = append(cmds, cmd)
				break
			}
			if node := m.binder.SelectedNode(); node != nil {
				m.prompt.open(promptDelete, node.Path,
					fmt.Sprintf("Delete '%s'? (y=confirm  Esc=cancel)", node.Name), "")
			}

		case "W":
			if m.focus != PaneBinder {
				var cmd tea.Cmd
				m.editor, cmd = m.editor.Update(msg)
				cmds = append(cmds, cmd)
				break
			}
			if m.showRightPanel {
				m.rightPanel.ToggleNoteMode()
				m.syncRightPanel()
			}

		case "e":
			if m.focus != PaneBinder {
				var cmd tea.Cmd
				m.editor, cmd = m.editor.Update(msg)
				cmds = append(cmds, cmd)
				break
			}
			if m.showRightPanel && m.rightPanel.noteMode {
				path := m.binder.SelectedFile()
				if path != "" {
					m.prompt.open(promptNote, path, "Note:", "")
					m.prompt.setInitialValue(m.rightPanel.currentNotes)
				}
			} else {
				var cmd tea.Cmd
				m.binder, cmd = m.binder.Update(msg)
				cmds = append(cmds, cmd)
			}

		case "f2":
			if err := m.enterCorkboard(); err != nil {
				m.statusMsg = fmt.Sprintf("Error opening corkboard: %v", err)
				m.statusTimer = 3
				cmds = append(cmds, statusTick())
			}

		case "f3":
			if err := m.enterOutliner(); err != nil {
				m.statusMsg = fmt.Sprintf("Error opening outliner: %v", err)
				m.statusTimer = 3
				cmds = append(cmds, statusTick())
			}

		case "f4":
			if err := m.enterTimeline(); err != nil {
				m.statusMsg = fmt.Sprintf("Error opening timeline: %v", err)
				m.statusTimer = 3
				cmds = append(cmds, statusTick())
			}

		case "f5":
			m.showRightPanel = !m.showRightPanel
			m.layout()
			if m.showRightPanel {
				m.syncRightPanel()
			}

		case "ctrl+e":
			p := core.NewProject(m.root)
			result, err := p.Export(m.pandocPath)
			if err != nil {
				m.statusMsg = fmt.Sprintf("Export failed: %v", err)
			} else if result.DocxPath != "" {
				m.statusMsg = fmt.Sprintf("Exported: %s + %s",
					filepath.Base(result.MarkdownPath),
					filepath.Base(result.DocxPath))
			} else {
				m.statusMsg = fmt.Sprintf("Exported: %s", filepath.Base(result.MarkdownPath))
			}
			m.statusTimer = 3
			cmds = append(cmds, statusTick())

		default:
			// Safety net: any key without an explicit case above is forwarded
			// to the focused pane. Keys that DO have their own case (e.g.
			// "enter") forward to the editor themselves when focused; this
			// catches everything else so a future key case can't silently
			// swallow editor input.
			m.pendingQuit = false
			if m.focus == PaneBinder {
				var cmd tea.Cmd
				m.binder, cmd = m.binder.Update(msg)
				cmds = append(cmds, cmd)
				m.syncRightPanel()
			} else {
				var cmd tea.Cmd
				m.editor, cmd = m.editor.Update(msg)
				cmds = append(cmds, cmd)
			}
		}

		return m, tea.Batch(cmds...)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout()
		return m, nil

	case statusTickMsg:
		if m.statusTimer > 0 {
			m.statusTimer--
			if m.statusTimer > 0 {
				cmds = append(cmds, statusTick())
			} else {
				m.statusMsg = ""
				m.pendingQuit = false // clear pending quit when message expires
			}
		}
		return m, tea.Batch(cmds...)
	}

	// Forward any other message (notably the textarea cursor-blink tick) to
	// the editor so the blink keeps animating. Blink is armed once on focus
	// gain and self-perpetuates through this path — no per-update re-arming.
	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)
	return m, cmd
}

// View renders the entire application.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if m.width == 0 {
		return "Starting..."
	}

	// Pane sizes are set in layout() on WindowSizeMsg; View only reads state.
	var statusParts []string
	if m.statusMsg != "" {
		statusParts = append(statusParts, m.statusMsg)
	}

	var body string
	switch {
	case m.showHistory:
		body = m.history.view()
		if m.history.mode == historyDiff {
			statusParts = append(statusParts, "[History]  ↑/↓ Scroll  Esc=Back  r=Restore")
		} else {
			statusParts = append(statusParts, "[History]  ↑/↓ Navigate  Enter=Diff  r=Restore  Esc=Close")
		}

	case m.viewMode == viewCorkboard:
		body = m.corkboard.view()
		statusParts = append(statusParts, "[Corkboard]  ←→↑↓ Navigate  Enter=Open  F3=Outliner  F4=Timeline  Esc=Back")

	case m.viewMode == viewOutliner:
		body = m.outliner.view()
		statusParts = append(statusParts, "[Outliner]  ↑/↓ Navigate  ←/→ Collapse/Expand  Enter=Open  F2=Corkboard  F4=Timeline  Esc=Back")

	case m.viewMode == viewTimeline:
		body = m.timeline.view()
		statusParts = append(statusParts, "[Timeline]  ↑/↓ Navigate  Enter=Open  F2=Corkboard  F3=Outliner  Esc=Back")

	default:
		if m.showRightPanel {
			body = lipgloss.JoinHorizontal(
				lipgloss.Top,
				m.binder.View(),
				m.editor.View(),
				m.rightPanel.view(),
			)
		} else {
			body = lipgloss.JoinHorizontal(
				lipgloss.Top,
				m.binder.View(),
				m.editor.View(),
			)
		}

		if m.statusMsg == "" && m.editor.FilePath() != "" {
			mod := ""
			if m.editor.IsModified() {
				mod = " ●"
			}
			statusParts = append(statusParts, fmt.Sprintf("%s — %d words%s",
				filepath.Base(m.editor.FilePath()),
				m.editor.WordCount(),
				mod,
			))
		}

		if m.focus == PaneBinder {
			statusParts = append(statusParts, "[Binder]  Tab=Switch  n=New  N=Folder  r=Rename  d=Delete  F2=Corkboard  F3=Outliner  F4=Timeline  F5=Panel")
		} else {
			statusParts = append(statusParts, "[Editor]  Tab=Switch  ^S=Save  ^N=New  F2=Corkboard  F4=Timeline  F5=Panel  ^E=Export")
		}
	}

	// The bottom row is either the prompt bar (during CRUD operations) or the
	// regular status bar. Both are exactly one row.
	var bottomBar string
	if m.prompt.active() {
		bottomBar = m.prompt.view(m.width)
	} else {
		statusText := truncate(strings.Join(statusParts, "  │  "),
			m.width-StatusBarStyle.GetHorizontalFrameSize())
		bottomBar = StatusBarStyle.Width(m.width).MaxHeight(1).Render(statusText)
	}

	full := lipgloss.JoinVertical(lipgloss.Left, body, bottomBar)

	// Quick-note popup overlays the existing view; background content stays visible.
	if m.quickNote.active() {
		return m.quickNote.view(m.width, m.height, full)
	}

	return full
}

// layout recalculates pane sizes after a window resize or panel toggle. It is
// the single place that pushes sizes into the panes.
func (m *Model) layout() {
	l := computeLayout(m.width, m.height, m.showRightPanel)
	m.binder.SetSize(l.binderW, l.paneH)
	m.editor.SetSize(l.editorW, l.paneH)
	m.rightPanel.SetSize(l.rightPanelW, l.paneH)
	// The history browser and the structural views all take the full width
	// above the status bar.
	fullH := m.height - 1
	if fullH < 1 {
		fullH = 1
	}
	m.history.SetSize(m.width, fullH)
	m.corkboard.SetSize(m.width, fullH)
	m.outliner.SetSize(m.width, fullH)
	m.timeline.SetSize(m.width, fullH)
}

// syncRightPanel updates the right panel's content to match the current binder
// selection. No-op when the panel is hidden. Called after any binder key event
// and after CRUD operations that refresh the tree.
func (m *Model) syncRightPanel() {
	if !m.showRightPanel {
		return
	}
	selectedFile := m.binder.SelectedFile()
	m.rightPanel.SyncToSelection(selectedFile)
	// Keep scene notes in sync. Use the editor's in-memory notes when the
	// selected file is also open in the editor; otherwise read from disk.
	if selectedFile != "" {
		var notes string
		if m.editor.FilePath() == selectedFile {
			notes = m.editor.Notes()
		} else {
			notes = core.ReadSceneNotes(selectedFile)
		}
		m.rightPanel.SyncNotes(selectedFile, notes)
	}
}

// ensureBackend lazily opens (initializing on first use) the revision backend
// for the project. Git init happens here, not at startup.
func (m *Model) ensureBackend() (core.RevisionBackend, error) {
	if m.revBackend == nil {
		b, err := core.OpenGitBackend(m.root)
		if err != nil {
			return nil, err
		}
		m.revBackend = b
	}
	return m.revBackend, nil
}

// snapshot records a revision of path. Trigger lives here (the caller decides
// when); the backend just snapshots.
func (m *Model) snapshot(path, message string) error {
	backend, err := m.ensureBackend()
	if err != nil {
		return err
	}
	return backend.Snapshot(path, message)
}

// openHistory loads the current scene's revision history into the browser.
func (m *Model) openHistory() error {
	backend, err := m.ensureBackend()
	if err != nil {
		return err
	}
	path := m.editor.FilePath()
	name := strings.TrimSuffix(filepath.Base(path), ".md")
	if err := m.history.open(backend, path, name); err != nil {
		return err
	}
	histH := m.height - 1
	if histH < 1 {
		histH = 1
	}
	m.history.SetSize(m.width, histH)
	m.showHistory = true
	return nil
}

// updateQuickNote routes a key press to the quick-note popup.
func (m Model) updateQuickNote(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	qn, action, cmd := m.quickNote.update(msg)
	m.quickNote = qn

	switch action {
	case quickNoteConfirmed:
		text := m.quickNote.value()
		m.quickNote.close()
		if err := core.AppendScratch(m.root, text); err != nil {
			m.statusMsg = fmt.Sprintf("Note error: %v", err)
		} else {
			m.statusMsg = "Note saved → notes/scratch.md"
			m.binder.RefreshPreservingExpanded()
		}
		m.statusTimer = 3
		return m, tea.Batch(statusTick(), cmd)
	case quickNoteCancelled:
		return m, cmd
	}
	return m, cmd
}

// updateHistory routes a key press to the history browser and applies whatever
// action it reports (close, or restore the selected revision into the editor).
func (m Model) updateHistory(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var action historyAction
	m.history, action = m.history.update(msg)

	switch action {
	case historyClose:
		m.showHistory = false
		m.focus = PaneEditor
		m.binder.Focus(false)
		m.editor.Focus(true)
		return m, m.editor.Init()

	case historyRestore:
		hash := m.history.selectedHash()
		m.showHistory = false
		m.focus = PaneEditor
		m.binder.Focus(false)
		m.editor.Focus(true)
		if hash == "" {
			return m, m.editor.Init()
		}
		content, err := m.revBackend.Restore(m.editor.FilePath(), hash)
		if err != nil {
			m.statusMsg = fmt.Sprintf("Restore failed: %v", err)
		} else {
			m.editor.LoadRevision(m.editor.FilePath(), content)
			m.statusMsg = fmt.Sprintf("Restored %s — review and Ctrl+S to keep", core.ShortHash(hash))
		}
		m.statusTimer = 4
		return m, tea.Batch(statusTick(), m.editor.Init())
	}

	return m, nil
}

// openScene loads path into the editor and switches to the main view with the
// editor focused, saving the current scene first if it has unsaved edits. It
// returns the command to run (status ticks + cursor blink). Shared by the binder
// and the structural views so "open a scene" behaves identically everywhere.
func (m *Model) openScene(path string) tea.Cmd {
	if m.editor.IsModified() {
		if err := m.editor.Save(); err != nil {
			m.statusMsg = fmt.Sprintf("Error saving: %v", err)
			m.statusTimer = 3
			return statusTick()
		}
	}
	if err := m.editor.LoadFile(path); err != nil {
		m.statusMsg = fmt.Sprintf("Error opening: %v", err)
		m.statusTimer = 3
		return statusTick()
	}
	m.viewMode = viewMain
	m.focus = PaneEditor
	m.binder.Focus(false)
	m.editor.Focus(true)
	m.statusMsg = fmt.Sprintf("Opened %s", filepath.Base(path))
	m.statusTimer = 2
	// Init() arms the cursor blink on focus gain.
	return tea.Batch(statusTick(), m.editor.Init())
}

// enterCorkboard loads the corkboard for the binder's current folder and shows
// it. Sizes are already current (layout runs on resize); SetSize here guards the
// case where no resize has happened yet.
func (m *Model) enterCorkboard() error {
	dir := m.binder.CurrentDir()
	name := folderDisplayName(dir, m.root)
	if err := m.corkboard.open(dir, name); err != nil {
		return err
	}
	fullH := m.height - 1
	if fullH < 1 {
		fullH = 1
	}
	m.corkboard.SetSize(m.width, fullH)
	m.viewMode = viewCorkboard
	return nil
}

// enterOutliner loads the project outline and shows it.
func (m *Model) enterOutliner() error {
	if err := m.outliner.open(m.root); err != nil {
		return err
	}
	fullH := m.height - 1
	if fullH < 1 {
		fullH = 1
	}
	m.outliner.SetSize(m.width, fullH)
	m.viewMode = viewOutliner
	return nil
}

// enterTimeline loads the project timeline and shows it.
func (m *Model) enterTimeline() error {
	if err := m.timeline.open(m.root); err != nil {
		return err
	}
	fullH := m.height - 1
	if fullH < 1 {
		fullH = 1
	}
	m.timeline.SetSize(m.width, fullH)
	m.viewMode = viewTimeline
	return nil
}

// updateView routes a key press to the active structural view. F1/Esc returns to
// the main view; F2/F3 hop directly between the structural views; everything else
// is forwarded to the active view, whose reported action (open/close) is applied.
func (m Model) updateView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "f1", "esc":
		m.viewMode = viewMain
		return m, nil
	case "f2":
		if err := m.enterCorkboard(); err != nil {
			m.statusMsg = fmt.Sprintf("Error opening corkboard: %v", err)
			m.statusTimer = 3
			return m, statusTick()
		}
		return m, nil
	case "f3":
		if err := m.enterOutliner(); err != nil {
			m.statusMsg = fmt.Sprintf("Error opening outliner: %v", err)
			m.statusTimer = 3
			return m, statusTick()
		}
		return m, nil
	case "f4":
		if err := m.enterTimeline(); err != nil {
			m.statusMsg = fmt.Sprintf("Error opening timeline: %v", err)
			m.statusTimer = 3
			return m, statusTick()
		}
		return m, nil
	}

	var action viewAction
	var path string
	switch m.viewMode {
	case viewCorkboard:
		m.corkboard, action = m.corkboard.update(msg)
		path = m.corkboard.selected()
	case viewOutliner:
		m.outliner, action = m.outliner.update(msg)
		path = m.outliner.selected()
	case viewTimeline:
		m.timeline, action = m.timeline.update(msg)
		path = m.timeline.selected()
	}

	switch action {
	case viewActionClose:
		m.viewMode = viewMain
	case viewActionOpen:
		if path != "" {
			m.syncRightPanel()
			return m, m.openScene(path)
		}
	}
	return m, nil
}

// updatePrompt routes a key press to the active binder prompt.
func (m Model) updatePrompt(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.prompt.mode {
	case promptDelete:
		switch msg.String() {
		case "y", "Y":
			return m.executePrompt()
		default:
			// Any other key (including Esc, n, N) cancels.
			m.prompt.close()
		}
		return m, nil
	default:
		switch msg.String() {
		case "esc":
			m.prompt.close()
			return m, nil
		case "enter":
			return m.executePrompt()
		default:
			var cmd tea.Cmd
			m.prompt, cmd = m.prompt.update(msg)
			return m, cmd
		}
	}
}

// executePrompt commits the active prompt action (create, rename, or delete).
func (m Model) executePrompt() (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	mode := m.prompt.mode
	ctx := m.prompt.context
	name := strings.TrimSpace(m.prompt.value())
	m.prompt.close()

	switch mode {
	case promptNewFile:
		if name == "" {
			m.statusMsg = "Name cannot be empty."
			m.statusTimer = 2
			return m, tea.Batch(append(cmds, statusTick())...)
		}
		path := filepath.Join(ctx, name+".md")
		if fileExists(path) {
			m.statusMsg = fmt.Sprintf("'%s.md' already exists.", name)
			m.statusTimer = 3
			return m, tea.Batch(append(cmds, statusTick())...)
		}
		if m.editor.IsModified() {
			if err := m.editor.Save(); err != nil {
				m.statusMsg = fmt.Sprintf("Error saving: %v", err)
				m.statusTimer = 3
				return m, tea.Batch(statusTick())
			}
		}
		if err := m.editor.NewScene(path); err != nil {
			m.statusMsg = fmt.Sprintf("Error creating scene: %v", err)
			m.statusTimer = 3
			cmds = append(cmds, statusTick())
			break
		}
		m.binder.RefreshPreservingExpanded()
		m.binder.SelectPath(path)
		m.rightPanel.markWorldDirty()
		m.syncRightPanel()
		m.focus = PaneEditor
		m.binder.Focus(false)
		m.editor.Focus(true)
		m.statusMsg = fmt.Sprintf("Created '%s.md'", name)
		m.statusTimer = 2
		cmds = append(cmds, statusTick(), m.editor.Init())

	case promptNewFolder:
		if name == "" {
			m.statusMsg = "Name cannot be empty."
			m.statusTimer = 2
			return m, tea.Batch(append(cmds, statusTick())...)
		}
		newPath, err := core.CreateFolder(ctx, name)
		if err != nil {
			m.statusMsg = fmt.Sprintf("Error creating folder: %v", err)
			m.statusTimer = 3
			cmds = append(cmds, statusTick())
			break
		}
		m.binder.RefreshPreservingExpanded()
		m.binder.SelectPath(newPath)
		m.rightPanel.markWorldDirty()
		m.syncRightPanel()
		m.statusMsg = fmt.Sprintf("Created folder '%s'", name)
		m.statusTimer = 2
		cmds = append(cmds, statusTick())

	case promptRename:
		if name == "" {
			m.statusMsg = "Name cannot be empty."
			m.statusTimer = 2
			return m, tea.Batch(append(cmds, statusTick())...)
		}
		newPath, err := core.RenameNode(ctx, name)
		if err != nil {
			m.statusMsg = fmt.Sprintf("Error renaming: %v", err)
			m.statusTimer = 3
			cmds = append(cmds, statusTick())
			break
		}
		if m.editor.FilePath() == ctx {
			m.editor.SetPath(newPath)
		}
		m.binder.RefreshPreservingExpanded()
		m.binder.SelectPath(newPath)
		m.rightPanel.markWorldDirty()
		m.syncRightPanel()
		m.statusMsg = fmt.Sprintf("Renamed to '%s'", filepath.Base(newPath))
		m.statusTimer = 2
		cmds = append(cmds, statusTick())

	case promptDelete:
		baseName := filepath.Base(ctx)
		if err := core.DeleteNode(ctx); err != nil {
			m.statusMsg = fmt.Sprintf("Error deleting: %v", err)
			m.statusTimer = 3
			cmds = append(cmds, statusTick())
			break
		}
		// Clear the editor if the open file (or a file inside a deleted folder) is gone.
		if isUnderDir(m.editor.FilePath(), ctx) {
			m.editor.Clear()
		}
		m.binder.RefreshPreservingExpanded()
		m.rightPanel.markWorldDirty()
		m.syncRightPanel()
		m.statusMsg = fmt.Sprintf("Deleted '%s'", baseName)
		m.statusTimer = 2
		cmds = append(cmds, statusTick())

	case promptNote:
		// ctx is the scene path. Route through the editor when the same file is
		// open in memory — that way the note is persisted on the next Ctrl+S
		// together with any unsaved body edits, preventing clobber.
		if m.editor.FilePath() == ctx {
			m.editor.SetNotes(name)
			m.statusMsg = "Note updated — Ctrl+S to save"
		} else {
			sc, err := core.LoadScene(ctx)
			if err != nil {
				m.statusMsg = fmt.Sprintf("Error loading scene: %v", err)
				m.statusTimer = 3
				cmds = append(cmds, statusTick())
				break
			}
			sc.Meta.Notes = name
			if err := sc.Save(); err != nil {
				m.statusMsg = fmt.Sprintf("Error saving note: %v", err)
				m.statusTimer = 3
				cmds = append(cmds, statusTick())
				break
			}
			m.statusMsg = "Note saved"
		}
		m.statusTimer = 2
		cmds = append(cmds, statusTick())
		m.syncRightPanel()
	}

	return m, tea.Batch(cmds...)
}

// Custom message types.
type statusTickMsg struct{}

func statusTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return statusTickMsg{}
	})
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
