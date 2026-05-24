package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Run is the CLI entry point. args[0] is the subcommand, args[1:] are
// arguments to that subcommand. If no subcommand is given, launches the TUI
// when inside a chisel project directory, or prints help otherwise.
func Run(args []string) error {
	if len(args) < 1 {
		// No subcommand — try to launch TUI.
		projectDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
		if isChiselProject(projectDir) {
			return StartTUI(projectDir)
		}
		printHelp()
		return nil
	}

	switch args[0] {
	case "new":
		if len(args) < 2 {
			return fmt.Errorf("usage: chisel new <project-name>")
		}
		return newProject(args[1])
	case "help", "-h", "--help":
		printHelp()
		return nil
	default:
		// Could be a path to a chisel project.
		projectDir := args[0]
		if isChiselProject(projectDir) {
			return StartTUI(projectDir)
		}
		return fmt.Errorf("unknown command: %s\nrun 'chisel help' for usage", args[0])
	}
}

// StartTUI launches the Bubble Tea TUI for the given project directory.
func StartTUI(projectDir string) error {
	m := NewModel(projectDir)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// isChiselProject returns true if the directory contains both manifest.jsonl
// and config.json.
func isChiselProject(dir string) bool {
	manifest := filepath.Join(dir, "manifest.jsonl")
	config := filepath.Join(dir, "config.json")
	_, errM := os.Stat(manifest)
	_, errC := os.Stat(config)
	return errM == nil && errC == nil
}

// newProject scaffolds a chisel project directory.
func newProject(name string) error {
	name = sanitiseName(name)

	projectDir, err := filepath.Abs(name)
	if err != nil {
		return fmt.Errorf("resolving project path: %w", err)
	}

	if _, err := os.Stat(projectDir); err == nil {
		return fmt.Errorf("directory already exists: %s", projectDir)
	}

	dirs := []string{
		projectDir,
		filepath.Join(projectDir, "scenes"),
		filepath.Join(projectDir, "research"),
		filepath.Join(projectDir, "exports"),
		filepath.Join(projectDir, "characters"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}

	// Write .gitignore.
	gitignore := "exports/\nconfig.json\n"
	if err := os.WriteFile(
		filepath.Join(projectDir, ".gitignore"),
		[]byte(gitignore),
		0644,
	); err != nil {
		return fmt.Errorf("writing .gitignore: %w", err)
	}

	// Write default config.json.
	cfg := DefaultConfig()
	if err := SaveConfig(projectDir, cfg); err != nil {
		return fmt.Errorf("writing config.json: %w", err)
	}

	// Write empty manifest.jsonl.
	manifestPath := filepath.Join(projectDir, "manifest.jsonl")
	if err := os.WriteFile(manifestPath, nil, 0644); err != nil {
		return fmt.Errorf("writing manifest.jsonl: %w", err)
	}

	// Initialise git repo and create initial commit via go-git.
	repo, err := git.PlainInit(projectDir, false)
	if err != nil {
		return fmt.Errorf("initialising git repo: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("getting worktree: %w", err)
	}

	if _, err := wt.Add("."); err != nil {
		return fmt.Errorf("staging files: %w", err)
	}

	_, err = wt.Commit("initial commit — chisel project scaffold", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "chisel",
			Email: "chisel@local",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("creating initial commit: %w", err)
	}

	fmt.Printf("created chisel project: %s\n", projectDir)
	return nil
}

// sanitiseName strips characters that are reserved on Windows file systems.
func sanitiseName(name string) string {
	reserved := map[rune]bool{
		'\\': true, '/': true, ':': true, '*': true,
		'?': true, '"': true, '<': true, '>': true, '|': true,
	}
	runes := make([]rune, 0, len(name))
	for _, r := range name {
		if !reserved[r] {
			runes = append(runes, r)
		}
	}
	return string(runes)
}

func printHelp() {
	fmt.Println(`chisel — a local-first, markdown-native writing tool with LLM augmentation.

usage:
  chisel                 launch TUI in current directory (if it's a project)
  chisel <project-dir>   launch TUI for an existing project
  chisel new <name>      scaffold a new writing project
  chisel help            show this help

shortcuts (when running):
  Ctrl+1/2/3   pane mode
  Ctrl+S       save (auto-commits)
  Ctrl+Z       undo
  Ctrl+F       find
  Ctrl+H       revision history
  Ctrl+R       rewrite (LLM)
  Ctrl+G       generate (LLM)
  Ctrl+Shift+S summarize (LLM)
  Ctrl+K       ask (LLM)
  Ctrl+A       analyse style (mirror)
  Ctrl+F5      research topic
  Ctrl+E       export manuscript
  Ctrl+Shift+E export docx (pandoc)
  Ctrl+B       corkboard view
  Ctrl+O       outline view
  Ctrl+L       timeline view
  Ctrl+T       cycle theme
  Ctrl+Shift+V toggle vim mode
  Ctrl+Shift+P writing sprint
  Ctrl+Shift+T typewriter mode
  Ctrl+Shift+C character sheets
  Ctrl+Shift+N scene notes
  Tab          switch focus
  Esc          return to editor`)
}
