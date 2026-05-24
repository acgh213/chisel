package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textarea"
)

// ---------------------------------------------------------------------------
// editor model
// ---------------------------------------------------------------------------

// EditorModel wraps a bubbles/textarea for markdown editing.
type EditorModel struct {
	textarea     textarea.Model
	currentFile  string // relative path, e.g. "scenes/ch01-arrival.md"
	savedContent string
	modified     bool
	wordCount    int
}

// NewEditorModel returns a configured editor with no file loaded.
func NewEditorModel() EditorModel {
	ta := textarea.New()
	ta.Placeholder = "start writing..."
	ta.ShowLineNumbers = false
	ta.CharLimit = 0 // no limit
	ta.SetWidth(80)
	ta.SetHeight(24)
	ta.Focus()

	return EditorModel{
		textarea: ta,
	}
}

// Update delegates key messages to the textarea and tracks modification state.
func (e EditorModel) Update(msg tea.Msg) (EditorModel, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		var cmd tea.Cmd
		e.textarea, cmd = e.textarea.Update(msg)
		return e, cmd
	}
	return e, nil
}

// ---------------------------------------------------------------------------
// file operations
// ---------------------------------------------------------------------------

// loadSceneFile reads the content of a scene .md file.
func loadSceneFile(projectDir, relPath string) (string, error) {
	fullPath := filepath.Join(projectDir, relPath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// saveSceneFile writes content to a scene .md file, creating parent dirs as
// needed.
func saveSceneFile(projectDir, relPath, content string) error {
	fullPath := filepath.Join(projectDir, relPath)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, []byte(content), 0644)
}

// deleteSceneFile removes a scene .md file from disk.
func deleteSceneFile(projectDir, relPath string) error {
	fullPath := filepath.Join(projectDir, relPath)
	return os.Remove(fullPath)
}

// renameSceneFile renames a scene .md file on disk.
func renameSceneFile(projectDir, oldPath, newPath string) error {
	fullOld := filepath.Join(projectDir, oldPath)
	fullNew := filepath.Join(projectDir, newPath)
	dir := filepath.Dir(fullNew)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.Rename(fullOld, fullNew)
}

// ---------------------------------------------------------------------------
// word count
// ---------------------------------------------------------------------------

// countWords returns the number of words in s. A word is any contiguous
// sequence of letters, digits, or apostrophes.
func countWords(s string) int {
	inWord := false
	count := 0
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\'' {
			if !inWord {
				count++
				inWord = true
			}
		} else {
			inWord = false
		}
	}
	return count
}

// wordCountFmt formats a word count with commas.
func wordCountFmt(n int) string {
	s := fmt.Sprintf("%d", n)
	var result strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(c)
	}
	return result.String()
}
