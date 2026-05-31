package tui

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/acgh213/chisel/core"
)

// EditorModel wraps bubbles/textarea for markdown editing.
// The editor edits a scene's prose BODY only; the scene's frontmatter metadata
// is held on m.scene and preserved across edits/saves, never shown as prose.
type EditorModel struct {
	textarea textarea.Model
	scene    *core.Scene // current scene (Meta + Body), nil if none open
	modified bool        // true if the body differs from the last load/save
	original string      // body at last save/load (for modified tracking)
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

// Init returns the editor's startup command: the textarea cursor-blink ticker
// (textarea.Blink). It does not re-initialize any editor state, so callers can
// safely use it to (re)arm the blink whenever the editor gains focus.
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
	// .Width()/.Height() in lipgloss include padding but NOT the border, which
	// is drawn outside. Subtract only the border so the rendered box is exactly
	// m.width × m.height and tiles flush against the binder.
	style := EditorStyle.
		Width(m.width - EditorStyle.GetHorizontalBorderSize()).
		Height(m.height - EditorStyle.GetVerticalBorderSize())
	if m.focus {
		style = FocusedStyle(style)
	}

	var content string
	if m.scene != nil {
		content = m.textarea.View()
	} else {
		content = lipgloss.NewStyle().
			Foreground(ColorDim).
			Padding(1).
			Render("No file open. Navigate to a .md file in the binder and press Enter.")
	}

	return style.Render(content)
}

// LoadFile reads a markdown file into the editor. The textarea shows only the
// prose body; the scene's frontmatter is kept on m.scene for save-time.
func (m *EditorModel) LoadFile(path string) error {
	sc, err := core.LoadScene(path)
	if err != nil {
		return err
	}

	m.scene = sc
	m.original = sc.Body
	m.modified = false
	m.textarea.Reset()
	m.textarea.SetValue(sc.Body)
	// Move cursor to start.
	m.textarea.CursorStart()

	return nil
}

// Save writes the editor's body back to the current scene, preserving (and, for
// scenes with metadata, refreshing) the frontmatter.
func (m *EditorModel) Save() error {
	if m.scene == nil {
		return nil // nothing to save
	}

	m.scene.Body = m.textarea.Value()
	if err := m.scene.Save(); err != nil {
		return err
	}

	m.original = m.scene.Body
	m.modified = false
	return nil
}

// IsModified returns true if there are unsaved changes.
func (m EditorModel) IsModified() bool {
	return m.modified
}

// FilePath returns the current scene's file path, or empty string if none open.
func (m EditorModel) FilePath() string {
	if m.scene == nil {
		return ""
	}
	return m.scene.Path
}

// Content returns the current editor content.
func (m EditorModel) Content() string {
	return m.textarea.Value()
}

// WordCount returns an estimate of the word count.
func (m EditorModel) WordCount() int {
	return core.WordCount(m.textarea.Value())
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

// SetSize sets the editor dimensions. w and h are the OUTER box size; the
// textarea is sized to the inner content area (box minus border + padding).
func (m *EditorModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	taW := w - EditorStyle.GetHorizontalFrameSize()
	taH := h - EditorStyle.GetVerticalFrameSize()
	if taW < 1 {
		taW = 1
	}
	if taH < 1 {
		taH = 1
	}
	m.textarea.SetWidth(taW)
	m.textarea.SetHeight(taH)
}

// NewScene creates a new .md file and loads it.
func (m *EditorModel) NewScene(path string) error {
	if _, err := core.CreateScene(path); err != nil {
		return err
	}
	return m.LoadFile(path)
}
