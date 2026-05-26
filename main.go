package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/acgh213/chisel/tui"
)

func main() {
	// Parse arguments: chisel <directory>
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: chisel <project-directory>\n")
		fmt.Fprintf(os.Stderr, "\nOpens a directory as a writing project.\n")
		fmt.Fprintf(os.Stderr, "Folders and .md files become the binder tree.\n")
		os.Exit(1)
	}

	root := os.Args[1]

	// Verify the directory exists.
	info, err := os.Stat(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: %s is not a directory\n", root)
		os.Exit(1)
	}

	// Create the root model.
	model, err := tui.NewModel(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Run the TUI.
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
