package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Template identifies which project scaffold to use.
type Template string

const (
	TemplateMinimal      Template = "minimal"
	TemplateNovel        Template = "novel"
	TemplateShortStories Template = "short-stories"
)

// ParseTemplate converts a string to a Template, reporting whether it was valid.
func ParseTemplate(s string) (Template, bool) {
	switch Template(s) {
	case TemplateMinimal, TemplateNovel, TemplateShortStories:
		return Template(s), true
	}
	return "", false
}

// Slugify converts a human name into a safe directory name: lowercase,
// non-alphanumeric characters replaced with hyphens, multiple hyphens
// collapsed, leading/trailing hyphens trimmed. Returns "" if the result
// would be empty (caller should error rather than using an empty path).
func Slugify(name string) string {
	var sb strings.Builder
	prevHyphen := false
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
			prevHyphen = false
		} else if !prevHyphen && sb.Len() > 0 {
			sb.WriteRune('-')
			prevHyphen = true
		}
	}
	return strings.TrimRight(sb.String(), "-")
}

// ScaffoldOptions controls what ScaffoldProject creates.
type ScaffoldOptions struct {
	Name     string   // project name shown in README (not the directory name)
	Template Template // which template to apply
}

// ScaffoldProject writes a new chisel project scaffold into dir. dir is
// created if it does not exist; if it already exists and is non-empty the
// call fails so existing work is never overwritten. ScaffoldProject is pure
// filesystem work — it produces no output; all messages belong in the caller.
func ScaffoldProject(dir string, opts ScaffoldOptions) error {
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		return fmt.Errorf("%s already exists and is not empty", dir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	readme := fmt.Sprintf("# %s\n\nA chisel writing project.\n", opts.Name)
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readme), 0o644); err != nil {
		return err
	}

	switch opts.Template {
	case TemplateMinimal:
		return nil // README is all minimal needs

	case TemplateNovel:
		return scaffoldNovel(dir)

	case TemplateShortStories:
		return scaffoldShortStories(dir)

	default:
		return fmt.Errorf("unknown template: %s", opts.Template)
	}
}

func scaffoldNovel(dir string) error {
	for _, sub := range []string{"scenes", "characters", "locations"} {
		if err := os.Mkdir(filepath.Join(dir, sub), 0o755); err != nil {
			return err
		}
	}

	scenes := []struct {
		file       string
		title      string
		draftOrder int
		body       string
	}{
		{"ch01-opening.md", "Opening", 1, "# Opening\n\nYour story begins here.\n"},
		{"ch02-rising-action.md", "Rising Action", 2, "# Rising Action\n\n"},
	}
	for _, s := range scenes {
		sc := &Scene{
			Path: filepath.Join(dir, "scenes", s.file),
			Meta: Metadata{
				Title:      s.title,
				Status:     StatusDraft,
				DraftOrder: s.draftOrder,
			},
			Body: s.body,
		}
		if err := sc.Save(); err != nil {
			return err
		}
	}
	return nil
}

func scaffoldShortStories(dir string) error {
	sc := &Scene{
		Path: filepath.Join(dir, "story-01.md"),
		Meta: Metadata{
			Title:      "Story One",
			Status:     StatusDraft,
			DraftOrder: 1,
		},
		Body: "# Story One\n\nYour story begins here.\n",
	}
	return sc.Save()
}
