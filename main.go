package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/acgh213/chisel/core"
	"github.com/acgh213/chisel/tui"
)

func main() {
	// Subcommand dispatch must happen before any os.Stat on args, so that
	// "init" is not mistakenly stat'd as a directory.
	if len(os.Args) >= 2 && os.Args[1] == "init" {
		runInit(os.Args[2:])
		return
	}

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  chisel <project-directory>       open a project in the editor\n")
		fmt.Fprintf(os.Stderr, "  chisel init [directory]          create a new project\n")
		fmt.Fprintf(os.Stderr, "  chisel init --template <tmpl> [directory]\n")
		fmt.Fprintf(os.Stderr, "\nTemplates: minimal, novel (default), short-stories\n")
		os.Exit(1)
	}

	root := os.Args[1]
	// Detect flags before subcommand (e.g. "chisel --template novel init")
	// and produce a friendlier error than "no such file or directory".
	if strings.HasPrefix(root, "-") {
		fmt.Fprintf(os.Stderr, "Error: %q looks like a flag. Did you mean 'chisel init %s'?\n", root, root)
		os.Exit(1)
	}
	info, err := os.Stat(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: %s is not a directory\n", root)
		os.Exit(1)
	}

	launchTUI(root)
}

// launchTUI opens root in the chisel TUI. It does not return until the user quits.
func launchTUI(root string) {
	model, err := tui.NewModel(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runInit handles the `chisel init` subcommand.
func runInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	templateFlag := fs.String("template", "", "template: minimal, novel (default), short-stories")
	noOpen := fs.Bool("no-open", false, "scaffold only; do not open the TUI after creation")
	fs.Parse(args)

	positional := fs.Args()

	var (
		dir  string
		name string
		tmpl core.Template
	)

	if *templateFlag == "" && len(positional) == 0 {
		// Interactive mode: read project name and template choice from stdin.
		reader := bufio.NewReader(os.Stdin)

		fmt.Fprint(os.Stderr, "Project name: ")
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			os.Exit(1)
		}
		name = strings.TrimSpace(line)
		if name == "" {
			name = "my-project"
		}

		fmt.Fprintln(os.Stderr, "Template:")
		fmt.Fprintln(os.Stderr, "  1) minimal       — bare directory with README")
		fmt.Fprintln(os.Stderr, "  2) novel         — scenes/, characters/, locations/ with sample chapters")
		fmt.Fprintln(os.Stderr, "  3) short-stories — single story file to start")
		fmt.Fprint(os.Stderr, "Choice [2]: ")
		choice, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			os.Exit(1)
		}
		switch strings.TrimSpace(choice) {
		case "1":
			tmpl = core.TemplateMinimal
		case "3":
			tmpl = core.TemplateShortStories
		default:
			tmpl = core.TemplateNovel
		}

		slug := core.Slugify(name)
		if slug == "" {
			fmt.Fprintf(os.Stderr, "Error: project name %q produces an empty directory name\n", name)
			os.Exit(1)
		}
		dir = slug
	} else {
		// Non-interactive mode: all options come from flags and positional args.
		// Flags must precede positional args (standard flag.FlagSet behaviour).
		rawTmpl := *templateFlag
		if rawTmpl == "" {
			rawTmpl = "novel" // default template
		}
		var ok bool
		tmpl, ok = core.ParseTemplate(rawTmpl)
		if !ok {
			fmt.Fprintf(os.Stderr, "Error: unknown template %q — choose: minimal, novel, short-stories\n", rawTmpl)
			os.Exit(1)
		}

		if len(positional) > 0 {
			dir = positional[0]
		} else {
			dir = "."
		}

		// Derive a display name from the directory.
		base := filepath.Base(dir)
		if base == "." {
			wd, err := os.Getwd()
			if err != nil {
				base = "project"
			} else {
				base = filepath.Base(wd)
			}
		}
		name = base
	}

	opts := core.ScaffoldOptions{Name: name, Template: tmpl}
	if err := core.ScaffoldProject(dir, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}
	fmt.Printf("Created '%s' from the %s template.\n", absDir, string(tmpl))

	if !*noOpen {
		launchTUI(absDir)
	}
}
