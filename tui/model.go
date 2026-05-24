package tui

import (
	"fmt"
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

	binder  BinderModel
	editor  EditorModel
	backend RevisionBackend

	mode  int // 1 = editor-only, 2 = binder+editor, 3 = binder+editor+llm
	focus FocusTarget

	confirm     *confirmDialog
	promptInput textinput.Model
	prompting   bool
	statusMsg   string
	statusTimer int // ticks remaining; 0 = clear

	width  int
	height int

	sessionStart  time.Time
	sessionWords  int
	lastWordCount int

	// history browser state
	showHistory   bool
	revisions     []Revision
	historyCursor int
	historyDiff   string // rendered diff when comparing two revs

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
		// Project might not have a git repo yet — that's ok for now.
		backend = nil
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
				name := m.promptInput.Value()
				m.prompting = false
				m.promptInput.Reset()
				if name != "" {
					return m, m.createScene(name)
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
	llmView := lipgloss.NewStyle().
		Width(llmW).
		Height(m.height-2).
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ColorBorder).
		Foreground(ColorMuted).
		Padding(0, 1).
		Render("LLM panel\n(phase 3)")

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
