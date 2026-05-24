package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
)

// ---------------------------------------------------------------------------
// focus target
// ---------------------------------------------------------------------------

// FocusTarget tracks which pane has keyboard focus.
type FocusTarget int

const (
	focusEditor  FocusTarget = iota
	focusBinder
)

// ---------------------------------------------------------------------------
// confirm dialog
// ---------------------------------------------------------------------------

type confirmDialog struct {
	message string
	onYes   tea.Cmd
}

// ---------------------------------------------------------------------------
// model
// ---------------------------------------------------------------------------

// Model is the root Bubble Tea model.
type Model struct {
	projectDir string
	config     Config
	manifest   []ManifestEntry

	binder    BinderModel
	editor    EditorModel
	backend   RevisionBackend
	llmClient *LLMClient

	mode  int // 1 = editor-only, 2 = binder+editor, 3 = binder+editor+llm
	focus FocusTarget

	confirm     *confirmDialog
	promptInput textinput.Model
	prompting   bool
	promptKind  string // "scene" or "ask"
	statusMsg   string
	statusTimer int // ticks remaining; 0 = clear

	// LLM panel state
	llmPanel     strings.Builder
	llmStreaming bool
	llmChan      <-chan LLMResponse
	llmOp        string // current operation for completion handling
	llmOpArg     string // e.g., research topic

	width  int
	height int

	sessionStart  time.Time
	sessionWords  int
	lastWordCount int

	// history browser state
	showHistory   bool
	revisions     []Revision
	historyCursor int
	historyDiff   string

	// corkboard / outline state
	showCorkboard bool
	showOutline   bool

	quitting bool
}

// NewModel initialises the application state for a given project directory.
func NewModel(projectDir string) Model {
	cfg, err := LoadConfig(projectDir)
	if err != nil {
		cfg = DefaultConfig()
	}

	manifest, _ := LoadManifest(projectDir)

	backend, err := NewGitBackend(projectDir)
	if err != nil {
		backend = nil
	}

	llmClient, err := NewLLMClient(projectDir)
	if err != nil {
		// LLM backend not available — TUI still works without it.
		llmClient = nil
	}

	pi := textinput.New()
	pi.Placeholder = "scene-name"
	pi.CharLimit = 64
	pi.Width = 40
	pi.Prompt = "new scene: "

	m := Model{
		projectDir:    projectDir,
		config:        cfg,
		manifest:      manifest,
		binder:        NewBinderModel(projectDir, manifest),
		editor:        NewEditorModel(),
		backend:       backend,
		llmClient:     llmClient,
		mode:          2,
		focus:         focusEditor,
		promptInput:   pi,
		sessionStart:  time.Now(),
		lastWordCount: 0,
	}

	// Select the first scene if one exists.
	if len(m.binder.linear) > 0 {
		for _, n := range m.binder.linear {
			if !n.IsDir {
				m.selectScene(n)
				break
			}
		}
	}

	return m
}

// ---------------------------------------------------------------------------
// init
// ---------------------------------------------------------------------------

func (m Model) Init() tea.Cmd {
	return tickCmd()
}

// ---------------------------------------------------------------------------
// tick
// ---------------------------------------------------------------------------

// llmResponseMsg carries a response line from the LLM backend.
type llmResponseMsg struct {
	resp LLMResponse
	err  error
}

type tickMsg struct{}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

// ---------------------------------------------------------------------------
// update
// ---------------------------------------------------------------------------

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Corkboard / outline mode — esc to exit.
	if m.showCorkboard || m.showOutline {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "esc" {
				m.showCorkboard = false
				m.showOutline = false
				return m, nil
			}
		}
		return m, nil
	}

	// History mode — overlay captures all keyboard input.
	if m.showHistory {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc", "ctrl+h":
				m.showHistory = false
				m.historyDiff = ""
				return m, nil
			case "up":
				if m.historyCursor > 0 {
					m.historyCursor--
				}
			case "down":
				if m.historyCursor < len(m.revisions)-1 {
					m.historyCursor++
				}
			case "enter":
				return m, m.historyShowDiff()
			case "r":
				return m, m.historyRestore()
			}
		}
		return m, nil
	}

	// Prompting mode — textinput captures all keyboard input.
	if m.prompting {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				value := m.promptInput.Value()
				kind := m.promptKind
				m.prompting = false
				m.promptInput.Reset()
				if value != "" {
					if kind == "ask" {
						return m, m.llmAsk(value)
					} else if kind == "research" {
						return m, m.llmResearch(value)
					} else if kind == "tag" {
						return m, m.addTag(value)
					} else if kind == "filter" {
						m.filterByTag(value)
						return m, nil
					}
					return m, m.createScene(value)
				}
				return m, nil
			case "esc":
				m.prompting = false
				m.promptInput.Reset()
				return m, nil
			}
			var cmd tea.Cmd
			m.promptInput, cmd = m.promptInput.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	// Handle confirm dialog first — it captures all input.
	if m.confirm != nil {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "y", "Y":
				cmd := m.confirm.onYes
				m.confirm = nil
				return m, cmd
			case "n", "N", "esc":
				m.confirm = nil
				return m, nil
			}
			return m, nil // swallow other keys
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.editor.textarea.SetWidth(m.editorWidth())
		m.editor.textarea.SetHeight(m.height - 3) // status bar + help line

	case tea.KeyMsg:
		return m, m.handleKey(msg)

	case tickMsg:
		if m.statusTimer > 0 {
			m.statusTimer--
			if m.statusTimer == 0 {
				m.statusMsg = ""
			}
		}
		return m, tickCmd()

	case llmResponseMsg:
		return m, m.handleLLMResponse(msg)
	}

	// Delegate to focused component.
	var cmd tea.Cmd
	if m.focus == focusEditor {
		m.editor, cmd = m.editor.Update(msg)
	} else {
		m.binder, cmd = m.binder.Update(msg)
	}

	return m, cmd
}

// ---------------------------------------------------------------------------
// key handling
// ---------------------------------------------------------------------------

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	// Global shortcuts (work regardless of focus).
	switch key {
	case "ctrl+1":
		m.mode = 1
		m.focus = focusEditor
		m.editor.textarea.SetWidth(m.editorWidth())
		return nil
	case "ctrl+2":
		m.mode = 2
		m.editor.textarea.SetWidth(m.editorWidth())
		return nil
	case "ctrl+3":
		m.mode = 3
		m.editor.textarea.SetWidth(m.editorWidth())
		return nil
	case "ctrl+s":
		return m.saveScene()
	case "ctrl+h":
		return m.openHistory()
	case "ctrl+r":
		return m.llmRewrite()
	case "ctrl+g":
		return m.llmGenerate()
	case "ctrl+shift+s":
		return m.llmSummarize()
	case "ctrl+k":
		return m.llmAskPrompt()
	case "ctrl+a":
		return m.llmAnalyze()
	case "f5":  // Ctrl+F5
		return m.llmResearchPrompt()
	case "t":
		return m.tagSceneDialog()
	case "T":
		return m.filterByTagDialog()
	case "ctrl+e":
		return m.exportManuscript()
	case "ctrl+b":
		return m.toggleCorkboard()
	case "ctrl+o":
		return m.toggleOutline()
	case "tab":
		if m.mode >= 2 {
			if m.focus == focusEditor {
				m.focus = focusBinder
			} else {
				m.focus = focusEditor
			}
		}
		return nil
	case "ctrl+c":
		if m.focus == focusBinder && m.mode >= 2 {
			m.focus = focusEditor
			return nil
		}
		// Quit — save first if needed, but don't block on it.
		if m.editor.modified {
			_ = m.saveScene()
		}
		m.quitting = true
		return tea.Quit
	case "esc":
		if m.focus == focusBinder && m.mode >= 2 {
			m.focus = focusEditor
			return nil
		}
		return nil
	}

	// Binder shortcuts.
	if m.focus == focusBinder && m.mode >= 2 {
		switch key {
		case "enter":
			return m.binderSelect()
		case "up", "down", "left", "right":
			m.binder, _ = m.binder.Update(msg)
			return nil
		case "n":
			return m.newSceneDialog()
		case "d":
			return m.deleteSceneDialog()
		case "f2":
			return m.renameSceneDialog()
		case "K":
			return m.reorderScene(-1)
		case "J":
			return m.reorderScene(1)
		}
		return nil
	}

	// Editor-local shortcuts.
	if m.focus == focusEditor {
		// Let the textarea handle its own keys.
		m.editor, _ = m.editor.Update(msg)
		if m.editor.textarea.Value() != m.editor.savedContent {
			m.editor.modified = true
		} else {
			m.editor.modified = false
		}
		return nil
	}

	return nil
}

// ---------------------------------------------------------------------------
// scene operations
// ---------------------------------------------------------------------------

func (m *Model) binderSelect() tea.Cmd {
	node := m.binder.selectedNode()
	if node == nil || node.IsDir {
		return nil
	}
	return m.selectScene(node)
}

func (m *Model) selectScene(node *SceneNode) tea.Cmd {
	// Save current scene first.
	if m.editor.modified {
		_ = m.saveScene()
	}

	content, err := loadSceneFile(m.projectDir, node.RelPath)
	if err != nil {
		m.setStatus("error loading scene: " + err.Error())
		return nil
	}

	m.editor.currentFile = node.RelPath
	m.editor.textarea.SetValue(content)
	m.editor.savedContent = content
	m.editor.modified = false
	m.editor.wordCount = countWords(content)

	m.lastWordCount = m.editor.wordCount
	m.focus = focusEditor
	m.editor.textarea.Focus()
	m.setStatus("")
	return func() tea.Msg { return textarea.Blink() }
}

func (m *Model) saveScene() tea.Cmd {
	if m.editor.currentFile == "" {
		return nil
	}

	content := m.editor.textarea.Value()
	wc := countWords(content)

	if err := saveSceneFile(m.projectDir, m.editor.currentFile, content); err != nil {
		m.setStatus("save error: " + err.Error())
		return nil
	}

	// Auto-commit via revision backend.
	if m.backend != nil {
		msg := fmt.Sprintf("scene: %s — %s words",
			strings.TrimSuffix(filepath.Base(m.editor.currentFile), ".md"),
			wordCountFmt(wc))
		fullPath := filepath.Join(m.projectDir, m.editor.currentFile)
		if err := m.backend.Save(fullPath, msg); err != nil {
			m.setStatus("commit error: " + err.Error())
			// Don't return — the file is still saved.
		}
	}

	// Track session words.
	diff := wc - m.lastWordCount
	if diff > 0 {
		m.sessionWords += diff
	}
	m.lastWordCount = wc

	m.updateManifestEntry(m.editor.currentFile, wc)

	m.editor.savedContent = content
	m.editor.modified = false
	m.editor.wordCount = wc
	m.setStatus(fmt.Sprintf("saved — %s words", wordCountFmt(wc)))
	return nil
}

func (m *Model) newSceneDialog() tea.Cmd {
	m.prompting = true
	m.promptKind = "scene"
	m.promptInput.Placeholder = "scene-name"
	m.promptInput.Prompt = "new scene: "
	m.promptInput.Focus()
	return textinput.Blink
}

func (m *Model) createScene(name string) tea.Cmd {
	name = sanitiseName(name)
	if !strings.HasSuffix(name, ".md") {
		name += ".md"
	}

	relPath := "scenes/" + name
	if err := saveSceneFile(m.projectDir, relPath, ""); err != nil {
		m.setStatus("error creating scene: " + err.Error())
		return nil
	}

	now := time.Now().Format(time.RFC3339)
	entry := ManifestEntry{
		ID:         strings.TrimSuffix(name, ".md"),
		File:       relPath,
		Title:      strings.TrimSuffix(name, ".md"),
		Status:     "draft",
		WordCount:  0,
		DraftOrder: len(m.manifest) + 1,
		Created:    now,
		Modified:   now,
	}
	m.manifest = append(m.manifest, entry)
	_ = SaveManifest(m.projectDir, m.manifest)

	m.binder = NewBinderModel(m.projectDir, m.manifest)
	m.setStatus(fmt.Sprintf("created %s", name))
	return nil
}

func (m *Model) deleteSceneDialog() tea.Cmd {
	node := m.binder.selectedNode()
	if node == nil || node.IsDir {
		return nil
	}

	name := node.Name
	m.confirm = &confirmDialog{
		message: fmt.Sprintf("delete %s? (y/n)", name),
		onYes: func() tea.Msg {
			cmd := m.deleteScene(node)
			return cmd()
		},
	}
	return nil
}

func (m *Model) deleteScene(node *SceneNode) tea.Cmd {
	if err := deleteSceneFile(m.projectDir, node.RelPath); err != nil {
		m.setStatus("error deleting: " + err.Error())
		return nil
	}

	id := strings.TrimSuffix(node.Name, ".md")
	filtered := make([]ManifestEntry, 0, len(m.manifest))
	for _, e := range m.manifest {
		if e.ID != id {
			filtered = append(filtered, e)
		}
	}
	m.manifest = filtered
	_ = SaveManifest(m.projectDir, m.manifest)

	if m.editor.currentFile == node.RelPath {
		m.editor.currentFile = ""
		m.editor.textarea.SetValue("")
		m.editor.savedContent = ""
		m.editor.modified = false
		m.editor.textarea.Blur()
	}

	m.binder = NewBinderModel(m.projectDir, m.manifest)
	m.setStatus(fmt.Sprintf("deleted %s", node.Name))
	return nil
}

func (m *Model) renameSceneDialog() tea.Cmd {
	node := m.binder.selectedNode()
	if node == nil || node.IsDir {
		return nil
	}

	oldName := node.Name
	newName := strings.TrimSuffix(oldName, ".md") + "-renamed.md"
	oldPath := node.RelPath
	newPath := "scenes/" + newName
	oldID := strings.TrimSuffix(oldName, ".md")
	newID := strings.TrimSuffix(newName, ".md")

	if err := renameSceneFile(m.projectDir, oldPath, newPath); err != nil {
		m.setStatus("error renaming: " + err.Error())
		return nil
	}

	for i, e := range m.manifest {
		if e.ID == oldID {
			m.manifest[i].ID = newID
			m.manifest[i].File = newPath
			m.manifest[i].Title = newID
			m.manifest[i].Modified = time.Now().Format(time.RFC3339)
		}
	}
	_ = SaveManifest(m.projectDir, m.manifest)

	if m.editor.currentFile == oldPath {
		m.editor.currentFile = newPath
	}

	m.binder = NewBinderModel(m.projectDir, m.manifest)
	m.setStatus(fmt.Sprintf("renamed %s → %s", oldName, newName))
	return nil
}

func (m *Model) updateManifestEntry(relPath string, wordCount int) {
	now := time.Now().Format(time.RFC3339)
	id := strings.TrimSuffix(filepath.Base(relPath), ".md")
	for i, e := range m.manifest {
		if e.ID == id {
			m.manifest[i].WordCount = wordCount
			m.manifest[i].Modified = now
			_ = SaveManifest(m.projectDir, m.manifest)
			return
		}
	}
	// Not in manifest — append.
	entry := ManifestEntry{
		ID:        id,
		File:      relPath,
		Title:     id,
		Status:    "draft",
		WordCount: wordCount,
		Created:   now,
		Modified:  now,
	}
	m.manifest = append(m.manifest, entry)
	_ = SaveManifest(m.projectDir, m.manifest)
}

// ---------------------------------------------------------------------------
// reorder
// ---------------------------------------------------------------------------

func (m *Model) reorderScene(delta int) tea.Cmd {
	node := m.binder.selectedNode()
	if node == nil || node.IsDir {
		return nil
	}

	id := strings.TrimSuffix(node.Name, ".md")

	// Sort manifest by draft_order and find the index.
	// Assign simple sequential draft orders based on current manifest order.
	// (The manifest order is the draft order.)
	idx := -1
	for i, e := range m.manifest {
		if e.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil
	}

	swapIdx := idx + delta
	if swapIdx < 0 || swapIdx >= len(m.manifest) {
		return nil
	}

	// Swap manifest entries.
	m.manifest[idx], m.manifest[swapIdx] = m.manifest[swapIdx], m.manifest[idx]

	// Update draft_order fields to reflect new positions.
	for i := range m.manifest {
		m.manifest[i].DraftOrder = i + 1
	}

	_ = SaveManifest(m.projectDir, m.manifest)
	m.binder = NewBinderModel(m.projectDir, m.manifest)
	m.setStatus(fmt.Sprintf("reordered %s", id))
	return nil
}

// ---------------------------------------------------------------------------
// history browser
// ---------------------------------------------------------------------------

func (m *Model) openHistory() tea.Cmd {
	if m.editor.currentFile == "" || m.backend == nil {
		m.setStatus("no file or no revision backend")
		return nil
	}

	fullPath := filepath.Join(m.projectDir, m.editor.currentFile)
	revs, err := m.backend.Log(fullPath)
	if err != nil {
		m.setStatus("history error: " + err.Error())
		return nil
	}

	if len(revs) == 0 {
		m.setStatus("no revisions yet")
		return nil
	}

	m.revisions = revs
	m.historyCursor = 0
	m.historyDiff = ""
	m.showHistory = true
	return nil
}

func (m *Model) historyShowDiff() tea.Cmd {
	if m.historyCursor < 0 || m.historyCursor >= len(m.revisions) {
		return nil
	}

	fullPath := filepath.Join(m.projectDir, m.editor.currentFile)
	cur := m.revisions[m.historyCursor]

	// Compare current with previous (or empty parent if first).
	var prevHash string
	if m.historyCursor+1 < len(m.revisions) {
		prevHash = m.revisions[m.historyCursor+1].Hash
	} else {
		// First commit — diff against empty.
		m.historyDiff = fmt.Sprintf("initial commit\n%s\n%s",
			cur.Hash[:8], cur.Message)
		return nil
	}

	diff, err := m.backend.Diff(fullPath, prevHash, cur.Hash)
	if err != nil {
		m.historyDiff = fmt.Sprintf("diff error: %v", err)
		return nil
	}

	m.historyDiff = fmt.Sprintf("%s ← %s\n\n%s", cur.Hash[:8], prevHash[:8], diff)
	return nil
}

func (m *Model) historyRestore() tea.Cmd {
	if m.historyCursor < 0 || m.historyCursor >= len(m.revisions) {
		return nil
	}

	rev := m.revisions[m.historyCursor]
	fullPath := filepath.Join(m.projectDir, m.editor.currentFile)

	content, err := m.backend.Restore(fullPath, rev.Hash)
	if err != nil {
		m.setStatus("restore error: " + err.Error())
		return nil
	}

	m.editor.textarea.SetValue(content)
	m.editor.savedContent = content
	m.editor.modified = false
	m.editor.wordCount = countWords(content)
	m.showHistory = false
	m.historyDiff = ""
	m.focus = focusEditor
	m.setStatus(fmt.Sprintf("restored %s", rev.Hash[:8]))
	return nil
}

// ---------------------------------------------------------------------------
// llm operations
// ---------------------------------------------------------------------------

func (m *Model) llmRewrite() tea.Cmd {
	if m.llmClient == nil {
		m.setStatus("LLM backend not available")
		return nil
	}
	text := m.editor.textarea.Value()
	if text == "" {
		return nil
	}
	// Use selected text if available, otherwise use full scene.
	// textarea doesn't expose selection — use the full text for now.
	return m.sendLLM(LLMRequest{
		Op:   "rewrite",
		Text: text,
	})
}

func (m *Model) llmGenerate() tea.Cmd {
	if m.llmClient == nil {
		m.setStatus("LLM backend not available")
		return nil
	}
	text := m.editor.textarea.Value()
	return m.sendLLM(LLMRequest{
		Op:      "generate",
		Text:    text,
		Guidance: "",
	})
}

func (m *Model) llmSummarize() tea.Cmd {
	if m.llmClient == nil {
		m.setStatus("LLM backend not available")
		return nil
	}
	text := m.editor.textarea.Value()
	if text == "" {
		return nil
	}
	return m.sendLLM(LLMRequest{
		Op:   "summarize",
		Text: text,
	})
}

func (m *Model) llmAskPrompt() tea.Cmd {
	if m.llmClient == nil {
		m.setStatus("LLM backend not available")
		return nil
	}
	m.prompting = true
	m.promptKind = "ask"
	m.promptInput.Placeholder = "ask about your text..."
	m.promptInput.Prompt = "ask: "
	m.promptInput.Focus()
	return textinput.Blink
}

func (m *Model) llmAnalyze() tea.Cmd {
	if m.llmClient == nil {
		m.setStatus("LLM backend not available")
		return nil
	}
	text := m.editor.textarea.Value()
	if text == "" {
		return nil
	}
	return m.sendLLM(LLMRequest{
		Op:   "analyze",
		Text: text,
	})
}

func (m *Model) llmResearchPrompt() tea.Cmd {
	if m.llmClient == nil {
		m.setStatus("LLM backend not available")
		return nil
	}
	m.prompting = true
	m.promptKind = "research"
	m.promptInput.Placeholder = "topic to research..."
	m.promptInput.Prompt = "research: "
	m.promptInput.Focus()
	return textinput.Blink
}

func (m *Model) llmResearch(topic string) tea.Cmd {
	// Save the research result to research/ directory.
	return m.sendLLM(LLMRequest{
		Op:       "ask",
		Question: fmt.Sprintf(
			"Research the following topic thoroughly. Provide key facts, context, and relevant details. Be concise but comprehensive.\n\nTopic: %s",
			topic,
		),
		Text: m.editor.textarea.Value(),
	})
}

func (m *Model) llmAsk(question string) tea.Cmd {
	return m.sendLLM(LLMRequest{
		Op:       "ask",
		Question: question,
		Text:     m.editor.textarea.Value(),
	})
}

func (m *Model) sendLLM(req LLMRequest) tea.Cmd {
	// Switch to mode 3 so the user sees the LLM panel.
	if m.mode < 3 {
		m.mode = 3
		m.editor.textarea.SetWidth(m.editorWidth())
	}

	// Clear previous panel content and show a loading indicator.
	m.llmPanel.Reset()
	fmt.Fprintf(&m.llmPanel, "▸ %s ...\n", req.Op)
	m.llmStreaming = true
	m.llmOp = req.Op
	m.llmOpArg = ""
	if req.Op == "ask" && req.Question != "" {
		m.llmOp = "research"
		m.llmOpArg = req.Question
	}

	ch, err := m.llmClient.Send(req)
	if err != nil {
		m.llmPanel.WriteString(fmt.Sprintf("error: %v\n", err))
		m.llmStreaming = false
		return nil
	}

	m.llmChan = ch
	return m.awaitLLMResponse()
}

func (m *Model) awaitLLMResponse() tea.Cmd {
	return func() tea.Msg {
		resp, ok := <-m.llmChan
		if !ok {
			return llmResponseMsg{err: fmt.Errorf("channel closed")}
		}
		return llmResponseMsg{resp: resp}
	}
}

func (m *Model) handleLLMResponse(msg llmResponseMsg) tea.Cmd {
	if msg.err != nil {
		m.llmPanel.WriteString(fmt.Sprintf("error: %v\n", msg.err))
		m.llmStreaming = false
		return nil
	}

	resp := msg.resp

	switch resp.Status {
	case "streaming":
		m.llmPanel.WriteString(resp.Result)
	case "ok":
		m.llmPanel.WriteString("\n")
		m.llmStreaming = false
		// Save research results to file.
		if m.llmOp == "research" {
			m.saveResearchNote()
		}
	case "error":
		m.llmPanel.WriteString(fmt.Sprintf("error: %s\n", resp.Result))
		m.llmStreaming = false
	}

	// If still streaming, request the next response.
	if m.llmStreaming {
		return m.awaitLLMResponse()
	}
	return nil
}

// ---------------------------------------------------------------------------
// tags
// ---------------------------------------------------------------------------

func (m *Model) tagSceneDialog() tea.Cmd {
	node := m.binder.selectedNode()
	if node == nil || node.IsDir {
		return nil
	}
	m.prompting = true
	m.promptKind = "tag"
	m.promptInput.Placeholder = "tag name"
	m.promptInput.Prompt = "add tag: "
	m.promptInput.Focus()
	return textinput.Blink
}

func (m *Model) addTag(tag string) tea.Cmd {
	node := m.binder.selectedNode()
	if node == nil || node.IsDir {
		return nil
	}
	id := strings.TrimSuffix(node.Name, ".md")
	for i, e := range m.manifest {
		if e.ID == id {
			for _, t := range e.Tags {
				if t == tag {
					m.setStatus("tag already exists")
					return nil
				}
			}
			m.manifest[i].Tags = append(m.manifest[i].Tags, tag)
			_ = SaveManifest(m.projectDir, m.manifest)
			m.setStatus(fmt.Sprintf("tagged %s → %s", id, tag))
			return nil
		}
	}
	return nil
}

func (m *Model) filterByTagDialog() tea.Cmd {
	m.prompting = true
	m.promptKind = "filter"
	m.promptInput.Placeholder = "tag to filter by (empty to clear)"
	m.promptInput.Prompt = "filter: "
	m.promptInput.Focus()
	return textinput.Blink
}

func (m *Model) filterByTag(tag string) {
	if tag == "" {
		// Clear filter.
		m.binder = NewBinderModel(m.projectDir, m.manifest)
		m.setStatus("filter cleared")
		return
	}

	// Filter manifest entries by tag.
	var filtered []ManifestEntry
	for _, e := range m.manifest {
		for _, t := range e.Tags {
			if t == tag {
				filtered = append(filtered, e)
				break
			}
		}
	}

	// Rebuild binder with only tagged scenes.
	m.binder = NewBinderModelFiltered(m.projectDir, filtered)
	m.setStatus(fmt.Sprintf("filter: %s (%d scenes)", tag, len(filtered)))
}

// ---------------------------------------------------------------------------
// export
// ---------------------------------------------------------------------------

func (m *Model) exportManuscript() tea.Cmd {
	// Sort manifest by draft_order.
	ordered := make([]ManifestEntry, len(m.manifest))
	copy(ordered, m.manifest)
	// Simple: manifest order = draft order.
	var sb strings.Builder
	for _, e := range ordered {
		content, err := loadSceneFile(m.projectDir, e.File)
		if err != nil {
			continue
		}
		sb.WriteString(content)
		sb.WriteString("\n\n")
	}
	exportPath := filepath.Join(m.projectDir, "exports", "manuscript.md")
	if err := os.MkdirAll(filepath.Dir(exportPath), 0755); err != nil {
		m.setStatus("export error: " + err.Error())
		return nil
	}
	if err := os.WriteFile(exportPath, []byte(sb.String()), 0644); err != nil {
		m.setStatus("export error: " + err.Error())
		return nil
	}
	m.setStatus(fmt.Sprintf("exported manuscript (%d scenes)", len(ordered)))
	return nil
}

// ---------------------------------------------------------------------------
// corkboard
// ---------------------------------------------------------------------------

func (m *Model) toggleCorkboard() tea.Cmd {
	m.showCorkboard = !m.showCorkboard
	m.showOutline = false
	return nil
}

func (m *Model) toggleOutline() tea.Cmd {
	m.showOutline = !m.showOutline
	m.showCorkboard = false
	return nil
}

// ---------------------------------------------------------------------------
// research
// ---------------------------------------------------------------------------

func (m *Model) saveResearchNote() {
	if m.llmOpArg == "" || m.editor.currentFile == "" {
		return
	}

	// Create slug from topic.
	slug := sanitiseName(m.llmOpArg)
	slug = strings.ToLower(slug)
	slug = strings.ReplaceAll(slug, " ", "-")
	if len(slug) > 40 {
		slug = slug[:40]
	}

	relPath := "research/" + slug + ".md"

	// Save the research note.
	content := fmt.Sprintf("# %s\n\n%s", m.llmOpArg, m.llmPanel.String())
	if err := saveSceneFile(m.projectDir, relPath, content); err != nil {
		m.setStatus("research save error: " + err.Error())
		return
	}

	// Auto-tag current scene.
	if m.editor.currentFile != "" {
		id := strings.TrimSuffix(filepath.Base(m.editor.currentFile), ".md")
		for i, e := range m.manifest {
			if e.ID == id {
				// Add research ref if not already present.
				found := false
				for _, ref := range e.ResearchRefs {
					if ref == slug {
						found = true
						break
					}
				}
				if !found {
					m.manifest[i].ResearchRefs = append(m.manifest[i].ResearchRefs, slug)
					_ = SaveManifest(m.projectDir, m.manifest)
				}
				break
			}
		}
	}

	m.setStatus(fmt.Sprintf("research saved: %s", slug))
}

// ---------------------------------------------------------------------------
// status
// ---------------------------------------------------------------------------

func (m *Model) setStatus(msg string) {
	m.statusMsg = msg
	m.statusTimer = 4 // ticks (~4 seconds)
}

// ---------------------------------------------------------------------------
// view
// ---------------------------------------------------------------------------

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if m.width == 0 || m.height == 0 {
		return "starting..."
	}

	// History overlay takes over the whole screen.
	if m.showHistory {
		return m.renderHistory()
	}

	// Corkboard view.
	if m.showCorkboard {
		return m.renderCorkboard()
	}

	// Outline view.
	if m.showOutline {
		return m.renderOutline()
	}

	// Build the main content area.
	var content string
	switch m.mode {
	case 1:
		content = m.renderMode1()
	case 2:
		content = m.renderMode2()
	case 3:
		content = m.renderMode3()
	}

	// Status bar (or prompt input).
	var status string
	if m.prompting {
		status = m.renderPrompt()
	} else {
		status = m.renderStatusBar()
	}

	// Help line.
	help := m.renderHelp()

	return lipgloss.JoinVertical(
		lipgloss.Top,
		content,
		status,
		help,
	)
}

func (m Model) renderMode1() string {
	editorView := m.renderEditor(m.width)
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height - 2).
		Render(editorView)
}

func (m Model) renderMode2() string {
	binderW := m.binderWidth()
	editorW := m.width - binderW

	binderView := m.binder.View(binderW)
	editorView := m.renderEditor(editorW)

	binderStyled := lipgloss.NewStyle().
		Width(binderW).
		Height(m.height - 2).
		BorderRight(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ColorBorder).
		Render(binderView)

	editorStyled := lipgloss.NewStyle().
		Width(editorW).
		Height(m.height - 2).
		Render(editorView)

	return lipgloss.JoinHorizontal(lipgloss.Top, binderStyled, editorStyled)
}

func (m Model) renderMode3() string {
	// Same as mode 2 but with an LLM panel placeholder on the right.
	binderW := m.binderWidth()
	llmW := m.width / 4
	if llmW < 20 {
		llmW = 20
	}
	editorW := m.width - binderW - llmW

	binderView := m.binder.View(binderW)
	editorView := m.renderEditor(editorW)
	llmContent := m.llmPanel.String()
	if llmContent == "" {
		llmContent = "LLM panel\n(no output yet)"
	}
	llmView := lipgloss.NewStyle().
		Width(llmW).
		Height(m.height-2).
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ColorBorder).
		Foreground(ColorFg).
		Padding(0, 1).
		Render(llmContent)

	binderStyled := lipgloss.NewStyle().
		Width(binderW).
		Height(m.height - 2).
		BorderRight(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ColorBorder).
		Render(binderView)

	editorStyled := lipgloss.NewStyle().
		Width(editorW).
		Height(m.height - 2).
		Render(editorView)

	return lipgloss.JoinHorizontal(lipgloss.Top, binderStyled, editorStyled, llmView)
}

func (m Model) renderEditor(width int) string {
	// Apply width to textarea.
	m.editor.textarea.SetWidth(width - 2)

	// Show focus indicator.
	style := lipgloss.NewStyle().Width(width).Padding(0, 1)
	if m.focus == focusEditor {
		style = style.BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(ColorAccent)
	} else {
		style = style.BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorBorder)
	}

	return style.Render(m.editor.textarea.View())
}

func (m Model) renderStatusBar() string {
	// Build status segments.
	var parts []string

	// Current scene.
	if m.editor.currentFile != "" {
		name := filepath.Base(m.editor.currentFile)
		parts = append(parts, strings.TrimSuffix(name, ".md"))
	}

	// Word count.
	parts = append(parts, wordCountFmt(m.editor.wordCount)+" words")

	// Modified indicator.
	if m.editor.modified {
		parts = append(parts, "[modified]")
	}

	// Session word count.
	if m.sessionWords > 0 {
		parts = append(parts, fmt.Sprintf("+%s this session", wordCountFmt(m.sessionWords)))
	}

	// Session timer.
	elapsed := time.Since(m.sessionStart)
	parts = append(parts, elapsed.Truncate(time.Second).String())

	// Focus indicator.
	if m.focus == focusBinder {
		parts = append(parts, "[binder]")
	}

	// Mode indicator.
	switch m.mode {
	case 1:
		parts = append(parts, "mode 1")
	case 2:
		// default — don't show
	case 3:
		parts = append(parts, "mode 3")
	}

	left := strings.Join(parts, " · ")

	// Status message (right-aligned transient message).
	right := ""
	if m.statusMsg != "" {
		right = m.statusMsg
	}

	// Pending confirm.
	if m.confirm != nil {
		right = m.confirm.message
	}

	barStyle := lipgloss.NewStyle().
		Background(ColorHighlight).
		Foreground(ColorMuted).
		Width(m.width).
		Padding(0, 1)

	if right != "" {
		// Simple approach: pad left and right.
		leftStyled := lipgloss.NewStyle().
			Background(ColorHighlight).
			Foreground(ColorMuted).
			Render(left)
		rightStyled := lipgloss.NewStyle().
			Background(ColorHighlight).
			Foreground(ColorAccent).
			Render(right)
		spacer := m.width - lipgloss.Width(leftStyled) - lipgloss.Width(rightStyled) - 2
		if spacer < 1 {
			spacer = 1
		}
		return barStyle.Render(leftStyled + strings.Repeat(" ", spacer) + rightStyled)
	}

	return barStyle.Render(left)
}

func (m Model) renderHistory() string {
	// Header.
	header := lipgloss.NewStyle().
		Background(ColorAccent).
		Foreground(ColorBg).
		Width(m.width).
		Padding(0, 1).
		Render("revision history — " + filepath.Base(m.editor.currentFile))

	// Build the list of revisions.
	var list strings.Builder
	for i, rev := range m.revisions {
		cursor := "  "
		if i == m.historyCursor {
			cursor = "> "
		}
		ts := rev.Timestamp.Format("2006-01-02 15:04")
		shortHash := rev.Hash[:8]
		list.WriteString(fmt.Sprintf("%s%s  %s  %s\n", cursor, shortHash, ts, rev.Message))
	}

	revisionList := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height/2 - 3).
		Padding(0, 1).
		Render(list.String())

	// Diff panel (if showing a diff).
	diffPanel := ""
	if m.historyDiff != "" {
		diffPanel = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height/2 - 1).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1).
			Render(m.historyDiff)
	}

	// Help bar.
	help := StyleHelp.Width(m.width).Render(
		"↑↓ select │ enter diff │ r restore │ esc back",
	)

	return lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		revisionList,
		diffPanel,
		help,
	)
}

func (m Model) renderCorkboard() string {
	header := lipgloss.NewStyle().
		Background(ColorAccent).
		Foreground(ColorBg).
		Width(m.width).
		Padding(0, 1).
		Render("corkboard — esc back")

	var cards strings.Builder
	cols := 3
	if m.width < 80 {
		cols = 2
	}
	cardWidth := m.width/cols - 2

	for _, e := range m.manifest {
		content, _ := loadSceneFile(m.projectDir, e.File)
		firstLine := strings.SplitN(content, "\n", 2)[0]
		if len(firstLine) > cardWidth-4 {
			firstLine = firstLine[:cardWidth-7] + "..."
		}

		card := lipgloss.NewStyle().
			Width(cardWidth).
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1).
			Render(fmt.Sprintf("%s\n%d words · %s\n%s",
				e.Title, e.WordCount, e.Status, firstLine))

		cards.WriteString(card)
	}

	return lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		lipgloss.NewStyle().Width(m.width).Render(cards.String()),
		StyleHelp.Width(m.width).Render("esc back"),
	)
}

func (m Model) renderOutline() string {
	header := lipgloss.NewStyle().
		Background(ColorAccent).
		Foreground(ColorBg).
		Width(m.width).
		Padding(0, 1).
		Render("outline — esc back")

	var lines strings.Builder
	for _, e := range m.manifest {
		statusIcon := "○"
		switch e.Status {
		case "revised":
			statusIcon = "◑"
		case "done":
			statusIcon = "●"
		}
		lines.WriteString(fmt.Sprintf("%s  %s  %s words\n",
			statusIcon, e.Title, wordCountFmt(e.WordCount)))
	}

	return lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		lipgloss.NewStyle().
			Width(m.width).
			Height(m.height-2).
			Padding(0, 1).
			Render(lines.String()),
		StyleHelp.Width(m.width).Render("esc back"),
	)
}

func (m Model) renderPrompt() string {
	return lipgloss.NewStyle().
		Background(ColorHighlight).
		Foreground(ColorAccent).
		Width(m.width).
		Padding(0, 1).
		Render(m.promptInput.View())
}

func (m Model) renderHelp() string {
	var parts []string

	switch m.focus {
	case focusBinder:
		parts = append(parts,
			"↑↓ navigate",
			"enter open",
			"←→ fold",
			"n new",
			"d delete",
			"F2 rename",
			"K/J reorder",
		)
	default:
		parts = append(parts,
			"^S save",
			"^Z undo",
			"^F find",
		)
	}

	parts = append(parts,
		"^1/^2/^3 mode",
		"tab focus",
		"^C quit",
	)

	return StyleHelp.Width(m.width).Render(strings.Join(parts, " │ "))
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func (m Model) editorWidth() int {
	switch m.mode {
	case 1:
		return m.width
	case 2:
		return m.width - m.binderWidth()
	case 3:
		llmW := m.width / 4
		if llmW < 20 {
			llmW = 20
		}
		return m.width - m.binderWidth() - llmW
	}
	return m.width
}

func (m Model) binderWidth() int {
	w := m.width / 4
	if w < 15 {
		w = 15
	}
	return w
}
