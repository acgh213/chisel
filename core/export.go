package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// ExportResult holds the paths produced by Export.
type ExportResult struct {
	MarkdownPath string
	DocxPath     string // empty when pandoc was not requested
}

// Export compiles every .md scene in the project (whole-project, recursive)
// into <root>/exports/manuscript.md. Scenes are ordered by draft_order then
// filename; scenes without a draft_order follow those that have one. Only the
// prose body is included — frontmatter is stripped. If pandocPath is
// non-empty, the markdown is also converted to manuscript.docx via pandoc.
//
// World-building directories (characters/, locations/, notes/) and hidden
// entries are excluded — only prose scenes are compiled.
func (p Project) Export(pandocPath string) (ExportResult, error) {
	var scenes []*Scene
	if err := walkMarkdown(p.Root, func(path string) error {
		sc, err := LoadScene(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}
		scenes = append(scenes, sc)
		return nil
	}); err != nil {
		return ExportResult{}, err
	}

	sort.SliceStable(scenes, func(i, j int) bool {
		ao, bo := scenes[i].Meta.DraftOrder, scenes[j].Meta.DraftOrder
		if (ao == 0) != (bo == 0) {
			return ao != 0
		}
		if ao != bo {
			return ao < bo
		}
		ni := strings.ToLower(strings.TrimSuffix(filepath.Base(scenes[i].Path), ".md"))
		nj := strings.ToLower(strings.TrimSuffix(filepath.Base(scenes[j].Path), ".md"))
		return ni < nj
	})

	var sb strings.Builder
	for i, sc := range scenes {
		if i > 0 {
			sb.WriteString("\n\n---\n\n")
		}
		sb.WriteString(strings.TrimSpace(sc.Body))
		sb.WriteString("\n")
	}

	outDir := filepath.Join(p.Root, "exports")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return ExportResult{}, fmt.Errorf("creating exports directory: %w", err)
	}

	mdPath := filepath.Join(outDir, "manuscript.md")
	if err := os.WriteFile(mdPath, []byte(sb.String()), 0o644); err != nil {
		return ExportResult{}, fmt.Errorf("writing manuscript.md: %w", err)
	}

	result := ExportResult{MarkdownPath: mdPath}

	if pandocPath != "" {
		docxPath := filepath.Join(outDir, "manuscript.docx")
		cmd := exec.Command(pandocPath, mdPath, "-o", docxPath)
		if out, err := cmd.CombinedOutput(); err != nil {
			return result, fmt.Errorf("pandoc: %w — %s", err, strings.TrimSpace(string(out)))
		}
		result.DocxPath = docxPath
	}

	return result, nil
}
