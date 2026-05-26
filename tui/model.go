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

// Init initializes the root model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.binder.Init(),
		m.editor.Init(),
		tea.EnterAltScreen,
	)
}

// Update handles all messages for the root model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Clear pending quit if user presses anything else.
		m.pendingQuit = false

		switch msg.String() {
		case "ctrl+q", "esc":
			if m.editor.IsModified() && !m.pendingQuit {
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
					cmds = append(cmds, statusTick())
				}
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
			// Pass through to focused pane.
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

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout()

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
				cmds = append(cmds, statusTick())
			}
		}

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
	}

	// Always blink the textarea cursor.
	cmds = append(cmds, m.editor.Init())

	return m, tea.Batch(cmds...)
}

// View renders the entire application.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if m.width == 0 {
		return "Starting..."
	}

	binderWidth := m.width / 3
	editorWidth := m.width - binderWidth
	paneHeight := m.height - 1

	m.binder.SetSize(binderWidth, paneHeight)
	m.editor.SetSize(editorWidth, paneHeight)

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

// layout recalculates pane sizes after a window resize.
func (m *Model) layout() {
	binderWidth := m.width / 3
	editorWidth := m.width - binderWidth
	paneHeight := m.height - 1

	m.binder.SetSize(binderWidth, paneHeight)
	m.editor.SetSize(editorWidth, paneHeight)
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
