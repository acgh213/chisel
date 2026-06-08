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

// OutlineExportResult holds the result of an outline-only export.
type OutlineExportResult struct {
	Path       string
	SceneCount int
	WordCount  int
}

// OutlineExport writes an outline-only export to exports/outline.md. The
// outline includes each scene's title, status, word count, word target, tags,
// POV, timeline date, synopsis, and notes — but no prose body.
func (p Project) OutlineExport() (OutlineExportResult, error) {
	var scenes []*Scene
	if err := walkMarkdown(p.Root, func(path string) error {
		sc, err := LoadScene(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}
		scenes = append(scenes, sc)
		return nil
	}); err != nil {
		return OutlineExportResult{}, err
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
	sb.WriteString("# Outline\n\n")
	totalWords := 0

	for _, sc := range scenes {
		title := sc.Meta.Title
		if title == "" {
			title = strings.TrimSuffix(filepath.Base(sc.Path), ".md")
		}
		sb.WriteString("### " + title + "\n\n")

		// Metadata line.
		var meta []string
		if sc.Meta.Status != "" {
			meta = append(meta, string(sc.Meta.Status))
		}
		wc := WordCount(sc.Body)
		totalWords += wc
		mt := fmt.Sprintf("%d words", wc)
		if sc.Meta.WordTarget > 0 {
			mt += fmt.Sprintf(" / %d target", sc.Meta.WordTarget)
		}
		meta = append(meta, mt)
		if len(sc.Meta.Tags) > 0 {
			meta = append(meta, "tags: "+strings.Join(sc.Meta.Tags, ", "))
		}
		if sc.Meta.POV != "" {
			meta = append(meta, "POV: "+sc.Meta.POV)
		}
		if sc.Meta.TimelineDate != nil {
			meta = append(meta, "date: "+sc.Meta.TimelineDate.Format("2006-01-02"))
		}
		sb.WriteString(strings.Join(meta, " · ") + "\n\n")

		if sc.Meta.Synopsis != "" {
			sb.WriteString("> " + sc.Meta.Synopsis + "\n\n")
		}
		if sc.Meta.Notes != "" {
			sb.WriteString(sc.Meta.Notes + "\n\n")
		}
	}

	outDir := filepath.Join(p.Root, "exports")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return OutlineExportResult{}, fmt.Errorf("creating exports directory: %w", err)
	}

	mdPath := filepath.Join(outDir, "outline.md")
	if err := os.WriteFile(mdPath, []byte(sb.String()), 0o644); err != nil {
		return OutlineExportResult{}, fmt.Errorf("writing outline.md: %w", err)
	}

	return OutlineExportResult{
		Path:       mdPath,
		SceneCount: len(scenes),
		WordCount:  totalWords,
	}, nil
}
