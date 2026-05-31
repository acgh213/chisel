package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
func (p Project) Export(pandocPath string) (ExportResult, error) {
	scenes, err := collectProjectScenes(p.Root)
	if err != nil {
		return ExportResult{}, err
	}
	SortScenesForReading(scenes)

	var sb strings.Builder
	for i, info := range scenes {
		sc, err := LoadScene(info.Path)
		if err != nil {
			return ExportResult{}, fmt.Errorf("reading %s: %w", info.Path, err)
		}
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

// collectProjectScenes recursively collects SceneInfo for all .md files under
// root. The exports/ subdirectory is skipped so manuscript.md is never
// re-included on subsequent exports. Hidden entries (leading dot) are also
// skipped.
func collectProjectScenes(root string) ([]SceneInfo, error) {
	exportsDir := filepath.Join(root, "exports")
	var scenes []SceneInfo

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		name := d.Name()

		if len(name) > 0 && name[0] == '.' {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() && filepath.Clean(path) == filepath.Clean(exportsDir) {
			return filepath.SkipDir
		}

		if !d.IsDir() && filepath.Ext(name) == ".md" {
			scenes = append(scenes, ReadSceneInfo(path))
		}
		return nil
	})
	return scenes, err
}
