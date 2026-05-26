package tui

import (
	"os"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// EditorModel wraps bubbles/textarea for markdown editing.
// Handles file loading, saving, and modified state tracking.
type EditorModel struct {
	textarea textarea.Model
	filePath string   // current file being edited, empty if none
	modified bool     // true if there are unsaved changes
	original string   // content at last save/load
	focus    bool
	width    int
	height   int
}

// NewEditor creates a new editor model.
func NewEditor() EditorModel {
	ta := textarea.New()
	ta.Placeholder = "No file open. Press Enter on a .md file in the binder to open it."
	ta.ShowLineNumbers = false // clean writing surface
	ta.CharLimit = 0           // no arbitrary limit
	ta.SetWidth(80)
	ta.SetHeight(24)

	// Style the textarea to match our theme.
	ta.FocusedStyle.Base = lipgloss.NewStyle()
	ta.BlurredStyle.Base = lipgloss.NewStyle()
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(lipgloss.Color("#2a2a3e"))
	ta.BlurredStyle.CursorLine = lipgloss.NewStyle().Background(lipgloss.Color("#2a2a3e"))
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("#5a5a6e"))
	ta.BlurredStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("#5a5a6e"))
	ta.FocusedStyle.Text = lipgloss.NewStyle().Foreground(ColorFg)
	ta.BlurredStyle.Text = lipgloss.NewStyle().Foreground(ColorFg)
	ta.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(ColorAccent)
	ta.BlurredStyle.Prompt = lipgloss.NewStyle().Foreground(ColorMuted)

	return EditorModel{
		textarea: ta,
	}
}

// Init initializes the editor model.
func (m EditorModel) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles messages for the editor.
func (m EditorModel) Update(msg tea.Msg) (EditorModel, tea.Cmd) {
	if !m.focus {
		return m, nil
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)

	// Track modified state.
	if m.textarea.Value() != m.original {
		m.modified = true
	} else {
		m.modified = false
	}

	return m, cmd
}

// View renders the editor.
func (m EditorModel) View() string {
	style := EditorStyle.Width(m.width).Height(m.height)
	if m.focus {
		style = FocusedStyle(style)
	}

	var content string
	if m.filePath != "" {
		content = m.textarea.View()
	} else {
		content = lipgloss.NewStyle().
			Foreground(ColorDim).
			Padding(1).
			Render("No file open. Navigate to a .md file in the binder and press Enter.")
	}

	return style.Render(content)
}

// LoadFile reads a markdown file into the editor.
func (m *EditorModel) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	m.filePath = path
	m.original = string(data)
	m.modified = false
	m.textarea.Reset()
	m.textarea.SetValue(m.original)
	// Move cursor to start.
	m.textarea.CursorStart()

	return nil
}

// Save writes the editor content to the current file.
func (m *EditorModel) Save() error {
	if m.filePath == "" {
		return nil // nothing to save
	}

	data := []byte(m.textarea.Value())
	if err := os.WriteFile(m.filePath, data, 0644); err != nil {
		return err
	}

	m.original = m.textarea.Value()
	m.modified = false
	return nil
}

// IsModified returns true if there are unsaved changes.
func (m EditorModel) IsModified() bool {
	return m.modified
}

// FilePath returns the current file path, or empty string.
func (m EditorModel) FilePath() string {
	return m.filePath
}

// Content returns the current editor content.
func (m EditorModel) Content() string {
	return m.textarea.Value()
}

// WordCount returns an estimate of the word count.
func (m EditorModel) WordCount() int {
	if m.textarea.Value() == "" {
		return 0
	}
	words := 0
	inWord := false
	for _, r := range m.textarea.Value() {
		if r == ' ' || r == '\n' || r == '\t' {
			inWord = false
		} else if !inWord {
			words++
			inWord = true
		}
	}
	return words
}

// Focus sets focus state.
func (m *EditorModel) Focus(v bool) {
	m.focus = v
	if v {
		m.textarea.Focus()
	} else {
		m.textarea.Blur()
	}
}

// SetSize sets the editor dimensions.
func (m *EditorModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.textarea.SetWidth(w - 4) // account for padding/borders
	m.textarea.SetHeight(h - 3)
}

// NewScene creates a new .md file and loads it.
func (m *EditorModel) NewScene(path string) error {
	// Create the file with a default template.
	content := "# Untitled\n\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}
	return m.LoadFile(path)
}
