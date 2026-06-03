package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// promptMode identifies what the inline prompt is collecting.
type promptMode int

const (
	promptNone      promptMode = iota
	promptNewFile              // typing a name for a new .md scene
	promptNewFolder            // typing a name for a new folder
	promptRename               // typing a new name for the selected node
	promptDelete               // y/n confirmation — no textinput, just keys
	promptNote                 // editing the Notes metadata field of a scene
)

// binderPrompt is the inline bottom-bar input used for binder CRUD operations.
// It occupies the same one-row slot as the status bar and disappears when done.
// Built as its own struct so the quick-note popup and right-panel can reuse the
// same infrastructure without touching the root model's key switch.
type binderPrompt struct {
	mode    promptMode
	input   textinput.Model
	context string // target path (directory for new ops; node path for rename/delete)
	label   string // full prompt string shown to the left of the input (or alone for delete)
}

func newBinderPrompt() binderPrompt {
	ti := textinput.New()
	ti.CharLimit = 200
	return binderPrompt{input: ti}
}

// open activates the prompt with the given mode and display text.
// context is the path that scopes the operation (current dir or target node).
func (p *binderPrompt) open(mode promptMode, context, label, placeholder string) {
	p.mode = mode
	p.context = context
	p.label = label
	p.input.Placeholder = placeholder
	p.input.SetValue("")
	if mode != promptDelete {
		p.input.Focus()
	}
}

// close deactivates the prompt.
func (p *binderPrompt) close() {
	p.mode = promptNone
	p.input.Blur()
	p.input.SetValue("")
}

// active reports whether a prompt is currently displayed.
func (p binderPrompt) active() bool {
	return p.mode != promptNone
}

// value returns the current textinput content.
func (p binderPrompt) value() string {
	return p.input.Value()
}

// setInitialValue pre-fills the input with existing text (used by promptNote to
// show the current note so the user can edit in place).
func (p *binderPrompt) setInitialValue(v string) {
	p.input.SetValue(v)
}

// update forwards a message to the textinput (no-op for delete prompts).
func (p binderPrompt) update(msg tea.Msg) (binderPrompt, tea.Cmd) {
	if p.mode == promptDelete {
		return p, nil
	}
	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	return p, cmd
}

// view renders the prompt bar at the given terminal width.
func (p binderPrompt) view(width int) string {
	style := PromptBarStyle.Width(width - PromptBarStyle.GetHorizontalFrameSize())
	if p.mode == promptDelete {
		return style.Render(p.label)
	}
	return style.Render(p.label + " " + p.input.View())
}
