package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExport_OrderAndBodyOnly(t *testing.T) {
	root := t.TempDir()

	writeScene(t, root, "b.md", "---\ntitle: B\ndraft_order: 2\n---\nBody of B.\n")
	writeScene(t, root, "a.md", "---\ntitle: A\ndraft_order: 1\n---\nBody of A.\n")
	writeScene(t, root, "z.md", "Body of Z.\n") // no draft_order — must come last

	result, err := NewProject(root).Export("")
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	data, err := os.ReadFile(result.MarkdownPath)
	if err != nil {
		t.Fatalf("reading result: %v", err)
	}
	content := string(data)

	posA := strings.Index(content, "Body of A.")
	posB := strings.Index(content, "Body of B.")
	posZ := strings.Index(content, "Body of Z.")
	if posA < 0 || posB < 0 || posZ < 0 {
		t.Fatalf("missing scene content in manuscript:\n%s", content)
	}
	if posA > posB {
		t.Errorf("expected A before B, got A@%d B@%d", posA, posB)
	}
	if posB > posZ {
		t.Errorf("expected B before Z, got B@%d Z@%d", posB, posZ)
	}

	// Frontmatter YAML keys must not appear in the manuscript body.
	for _, key := range []string{"title:", "draft_order:", "status:", "synopsis:"} {
		if strings.Contains(content, key) {
			t.Errorf("manuscript contains frontmatter key %q:\n%s", key, content)
		}
	}
}

func TestExport_SkipsExportsDir(t *testing.T) {
	root := t.TempDir()
	writeScene(t, root, "scene.md", "Hello world.\n")

	// Pre-create exports/manuscript.md — it must not be re-included.
	exportsDir := filepath.Join(root, "exports")
	if err := os.MkdirAll(exportsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeScene(t, exportsDir, "manuscript.md", "OLD CONTENT\n")

	result, err := NewProject(root).Export("")
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	data, _ := os.ReadFile(result.MarkdownPath)
	if strings.Contains(string(data), "OLD CONTENT") {
		t.Errorf("exports/manuscript.md was re-included in the export")
	}
}

func TestExport_EmptyProject(t *testing.T) {
	root := t.TempDir()
	result, err := NewProject(root).Export("")
	if err != nil {
		t.Fatalf("Export on empty project: %v", err)
	}
	data, _ := os.ReadFile(result.MarkdownPath)
	if len(strings.TrimSpace(string(data))) != 0 {
		t.Errorf("expected empty manuscript for empty project, got %q", string(data))
	}
}

func TestExport_WritesMarkdownPath(t *testing.T) {
	root := t.TempDir()
	writeScene(t, root, "ch1.md", "---\ndraft_order: 1\n---\nChapter one.\n")

	result, err := NewProject(root).Export("")
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	want := filepath.Join(root, "exports", "manuscript.md")
	if result.MarkdownPath != want {
		t.Errorf("MarkdownPath = %q, want %q", result.MarkdownPath, want)
	}
	if result.DocxPath != "" {
		t.Errorf("DocxPath should be empty without pandoc, got %q", result.DocxPath)
	}

	if _, err := os.Stat(want); err != nil {
		t.Errorf("manuscript.md not created: %v", err)
	}
}
