package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Pane indicates which pane has focus.
type Pane int

const (
	PaneBinder Pane = iota
	PaneEditor
)

// minBinderWidth is the narrowest the binder pane may shrink to before the
// editor starts giving up columns. Keeps the tree readable on small terminals.
const minBinderWidth = 20

// layoutSizes holds the OUTER box dimensions (including each pane's border)
// for the binder and editor, plus the shared pane height.
type layoutSizes struct {
	binderW int
	editorW int
	paneH   int
}

// computeLayout splits a terminal of the given size into binder/editor pane
// dimensions. It is pure (no model state) so it can be unit-tested without a
// running terminal. Widths are OUTER box sizes and sum to width when the
// terminal is wide enough; one row is reserved for the status bar. Every value
// is floored at 1 so a tiny terminal can never produce a zero/negative box
// (which is what makes lipgloss/textarea misbehave).
func computeLayout(width, height int) layoutSizes {
	paneH := height - 1
	if paneH < 1 {
		paneH = 1
	}

	binderW := width / 3
	if binderW < minBinderWidth {
		binderW = minBinderWidth
	}
	editorW := width - binderW

	// When the terminal is too narrow for the preferred split, the editor
	// takes priority and the binder shrinks to whatever is left.
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
}

// NewModel creates a new chisel root model for the given project directory.
func NewModel(root string) (Model, error) {
	binder := NewBinder(root)
	if err := binder.Refresh(); err != nil {
		return Model{}, fmt.Errorf("reading project directory: %w", err)
	}

	return Model{
		binder: binder,
		editor: NewEditor(),
		focus:  PaneBinder,
		root:   root,
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
		// Capture the pending-quit state before clearing it: the
		// "press Ctrl+Q again" guard reads the value from the *previous*
		// key press, while any key cancels a pending quit for the next one.
		wasPending := m.pendingQuit
		m.pendingQuit = false

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
			}

		case "enter":
			if m.focus == PaneBinder {
				path := m.binder.SelectedFile()
				if path != "" {
					if m.editor.IsModified() {
						if err := m.editor.Save(); err != nil {
							m.statusMsg = fmt.Sprintf("Error saving: %v", err)
							m.statusTimer = 3
							cmds = append(cmds, statusTick())
							return m, tea.Batch(cmds...)
						}
					}
					if err := m.editor.LoadFile(path); err != nil {
						m.statusMsg = fmt.Sprintf("Error opening: %v", err)
						m.statusTimer = 3
						cmds = append(cmds, statusTick())
						return m, tea.Batch(cmds...)
					}
					m.focus = PaneEditor
					m.binder.Focus(false)
					m.editor.Focus(true)
					m.statusMsg = fmt.Sprintf("Opened %s", filepath.Base(path))
					m.statusTimer = 2
					// Init() arms the cursor blink on focus gain.
					cmds = append(cmds, statusTick(), m.editor.Init())
				} else {
					// A folder (or nothing) is selected — let the binder
					// toggle it open/closed.
					var cmd tea.Cmd
					m.binder, cmd = m.binder.Update(msg)
					cmds = append(cmds, cmd)
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
					m.statusMsg = fmt.Sprintf("Saved %s (%d words)",
						filepath.Base(m.editor.FilePath()), m.editor.WordCount())
				}
				m.statusTimer = 3
				cmds = append(cmds, statusTick())
			} else {
				m.statusMsg = "No file open to save."
				m.statusTimer = 2
				cmds = append(cmds, statusTick())
			}

		case "ctrl+n":
			return m, m.newScenePrompt()

		default:
			// Safety net: any key without an explicit case above is forwarded
			// to the focused pane. Keys that DO have their own case (e.g.
			// "enter") forward to the editor themselves when focused; this
			// catches everything else so a future key case can't silently
			// swallow editor input.
			if m.focus == PaneBinder {
				var cmd tea.Cmd
				m.binder, cmd = m.binder.Update(msg)
				cmds = append(cmds, cmd)
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

	case newSceneMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error creating scene: %v", msg.err)
			m.statusTimer = 3
			cmds = append(cmds, statusTick())
		} else {
			if m.editor.IsModified() {
				m.editor.Save()
			}
			if err := m.editor.NewScene(msg.path); err != nil {
				m.statusMsg = fmt.Sprintf("Error: %v", err)
				m.statusTimer = 3
				cmds = append(cmds, statusTick())
			} else {
				m.binder.Refresh()
				m.focus = PaneEditor
				m.binder.Focus(false)
				m.editor.Focus(true)
				m.statusMsg = "New scene created."
				m.statusTimer = 2
				// Init() arms the cursor blink on focus gain.
				cmds = append(cmds, statusTick(), m.editor.Init())
			}
		}
		return m, tea.Batch(cmds...)

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
	panes := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.binder.View(),
		m.editor.View(),
	)

	// Status bar.
	var statusParts []string
	if m.statusMsg != "" {
		statusParts = append(statusParts, m.statusMsg)
	} else if m.editor.FilePath() != "" {
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
		statusParts = append(statusParts, "[Binder]  Tab=Switch")
	} else {
		statusParts = append(statusParts, "[Editor]  Tab=Switch  ^S=Save  ^N=New")
	}

	status := StatusBarStyle.Width(m.width).Render(strings.Join(statusParts, "  │  "))

	return lipgloss.JoinVertical(lipgloss.Left, panes, status)
}

// layout recalculates pane sizes after a window resize. It is the single place
// that pushes sizes into the panes.
func (m *Model) layout() {
	l := computeLayout(m.width, m.height)
	m.binder.SetSize(l.binderW, l.paneH)
	m.editor.SetSize(l.editorW, l.paneH)
}

// newScenePrompt handles the Ctrl+N new-scene flow.
func (m *Model) newScenePrompt() tea.Cmd {
	scenesDir := filepath.Join(m.root, "scenes")
	if info, err := os.Stat(scenesDir); err != nil || !info.IsDir() {
		scenesDir = m.root
	}

	filename := "new-scene.md"
	path := filepath.Join(scenesDir, filename)

	for i := 2; fileExists(path); i++ {
		filename = fmt.Sprintf("new-scene-%d.md", i)
		path = filepath.Join(scenesDir, filename)
	}

	return func() tea.Msg {
		return newSceneMsg{path: path}
	}
}

// Custom message types.
type newSceneMsg struct {
	path string
	err  error
}

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
