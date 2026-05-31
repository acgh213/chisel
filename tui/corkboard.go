package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/acgh213/chisel/core"
)

// viewAction is what a structural view (corkboard/outliner) asks the root model
// to do after a key it can't handle on its own.
type viewAction int

const (
	viewActionNone  viewAction = iota
	viewActionClose            // return to the main binder+editor view
	viewActionOpen             // open the selected scene in the editor
)

// Card text dimensions (the wrap width and line budget inside a card's border +
// padding). The outer card size is derived from these plus the style frame.
const (
	cardTextW         = 26
	cardSynopsisLines = 3
	cardGap           = 1 // blank columns between cards in a row
)

// corkboardModel is Scrivener's corkboard: a scrollable grid of index cards for
// the scenes in one folder. It owns view state only; scene data comes from core.
type corkboardModel struct {
	dir   string // folder being shown
	name  string // display name for the header
	cards []core.SceneInfo

	cursor int // index into cards
	offset int // first visible row (in card-rows), for scrolling

	width  int
	height int
}

// open loads the scenes in dir and resets the grid. A read error is returned so
// the root model can surface it without switching views.
func (c *corkboardModel) open(dir, name string) error {
	cards, err := core.FolderScenes(dir)
	if err != nil {
		return err
	}
	c.dir = dir
	c.name = name
	c.cards = cards
	c.cursor = 0
	c.offset = 0
	return nil
}

// SetSize sets the corkboard's outer dimensions.
func (c *corkboardModel) SetSize(w, h int) {
	c.width = w
	c.height = h
}

// selected returns the path of the card under the cursor, or "".
func (c corkboardModel) selected() string {
	if c.cursor < 0 || c.cursor >= len(c.cards) {
		return ""
	}
	return c.cards[c.cursor].Path
}

// cols is how many cards fit across the current width (at least one).
func (c corkboardModel) cols() int {
	outer := cardTextW + CardStyle.GetHorizontalFrameSize()
	cols := (c.width + cardGap) / (outer + cardGap)
	if cols < 1 {
		cols = 1
	}
	return cols
}

// cardOuterH is one card's total height: title + meta + synopsis lines, plus
// the border (CardStyle has no vertical padding).
func cardOuterH() int {
	return 2 + cardSynopsisLines + CardStyle.GetVerticalFrameSize()
}

// visibleRows is how many card-rows fit under the header (at least one).
func (c corkboardModel) visibleRows() int {
	rows := (c.height - 1) / cardOuterH() // minus the header row
	if rows < 1 {
		rows = 1
	}
	return rows
}

// update handles a key press and reports any action the root model must take.
func (c corkboardModel) update(msg tea.KeyMsg) (corkboardModel, viewAction) {
	cols := c.cols()
	switch msg.String() {
	case "left", "h":
		if c.cursor > 0 {
			c.cursor--
		}
	case "right", "l":
		if c.cursor < len(c.cards)-1 {
			c.cursor++
		}
	case "up", "k":
		if c.cursor-cols >= 0 {
			c.cursor -= cols
		}
	case "down", "j":
		if c.cursor+cols < len(c.cards) {
			c.cursor += cols
		}
	case "enter":
		if c.selected() != "" {
			return c, viewActionOpen
		}
	}
	c.scrollToCursor()
	return c, viewActionNone
}

// scrollToCursor keeps the selected card's row within the visible window.
func (c *corkboardModel) scrollToCursor() {
	cols := c.cols()
	rows := c.visibleRows()
	cursorRow := c.cursor / cols
	if cursorRow < c.offset {
		c.offset = cursorRow
	}
	if cursorRow >= c.offset+rows {
		c.offset = cursorRow - rows + 1
	}
	if c.offset < 0 {
		c.offset = 0
	}
}

// view renders the corkboard: a header line plus the visible card grid.
func (c corkboardModel) view() string {
	header := ViewHeaderStyle.Render(truncate(
		fmt.Sprintf("Corkboard — %s   (%d scenes)", c.name, len(c.cards)), c.width))

	if len(c.cards) == 0 {
		empty := lipgloss.NewStyle().Foreground(ColorDim).
			Render("(no scenes in this folder — Ctrl+N to add one, Esc to go back)")
		return lipgloss.JoinVertical(lipgloss.Left, header, "", empty)
	}

	cols := c.cols()
	rows := c.visibleRows()
	startRow := c.offset
	startIdx := startRow * cols
	endIdx := startIdx + rows*cols
	if endIdx > len(c.cards) {
		endIdx = len(c.cards)
	}

	var rowBlocks []string
	for i := startIdx; i < endIdx; i += cols {
		var cells []string
		for j := i; j < i+cols && j < endIdx; j++ {
			if len(cells) > 0 {
				cells = append(cells, strings.Repeat(" ", cardGap))
			}
			cells = append(cells, c.renderCard(c.cards[j], j == c.cursor))
		}
		rowBlocks = append(rowBlocks, lipgloss.JoinHorizontal(lipgloss.Top, cells...))
	}

	grid := lipgloss.JoinVertical(lipgloss.Left, rowBlocks...)
	return lipgloss.JoinVertical(lipgloss.Left, header, grid)
}

// renderCard draws one index card. A forced-width synopsis block fixes every
// card to cardTextW so the grid columns align regardless of content.
func (c corkboardModel) renderCard(card core.SceneInfo, selected bool) string {
	title := CardTitleStyle.Render(truncate(card.Title, cardTextW))
	meta := CardMetaStyle.Render(truncate(cardMetaLine(card), cardTextW))

	syn := card.Synopsis
	if syn == "" {
		syn = "(no synopsis)"
	}
	synBlock := CardSynopsisStyle.
		Width(cardTextW).
		Height(cardSynopsisLines).
		MaxHeight(cardSynopsisLines).
		Render(syn)

	content := lipgloss.JoinVertical(lipgloss.Left, title, meta, synBlock)

	style := CardStyle
	if selected {
		style = CardSelectedStyle
	}
	return style.Render(content)
}

// cardMetaLine is the status · word-count line under a card's title.
func cardMetaLine(card core.SceneInfo) string {
	var parts []string
	if card.Status != "" {
		parts = append(parts, string(card.Status))
	}
	parts = append(parts, wordsLabel(card))
	return strings.Join(parts, " · ")
}

// wordsLabel renders a scene's progress as "123/2000" when it has a target, or
// "123w" when it doesn't.
func wordsLabel(card core.SceneInfo) string {
	if card.WordTarget > 0 {
		return fmt.Sprintf("%d/%d", card.WordCount, card.WordTarget)
	}
	return fmt.Sprintf("%dw", card.WordCount)
}

// folderDisplayName turns a directory path into a short header label, using the
// project root's base name (or "Project") for the root itself.
func folderDisplayName(dir, root string) string {
	if dir == root {
		if b := filepath.Base(root); b != "." && b != string(filepath.Separator) {
			return b
		}
		return "Project"
	}
	return filepath.Base(dir)
}
