package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseFrontmatter(t *testing.T) {
	raw := "---\n" +
		"title: Chapter One\n" +
		"status: revised\n" +
		"tags:\n  - opening\n  - establishing\n" +
		"draft_order: 1\n" +
		"---\n" +
		"# Chapter One\n\nThe train pulled in at dusk.\n"

	meta, body := parseFrontmatter(raw)

	if meta.Title != "Chapter One" {
		t.Errorf("title = %q, want %q", meta.Title, "Chapter One")
	}
	if meta.Status != StatusRevised {
		t.Errorf("status = %q, want %q", meta.Status, StatusRevised)
	}
	if len(meta.Tags) != 2 || meta.Tags[0] != "opening" || meta.Tags[1] != "establishing" {
		t.Errorf("tags = %v, want [opening establishing]", meta.Tags)
	}
	if meta.DraftOrder != 1 {
		t.Errorf("draft_order = %d, want 1", meta.DraftOrder)
	}
	if want := "# Chapter One\n\nThe train pulled in at dusk.\n"; body != want {
		t.Errorf("body = %q, want %q", body, want)
	}
}

func TestParseNoFrontmatter(t *testing.T) {
	raw := "# Just Prose\n\nNo frontmatter here.\n"
	meta, body := parseFrontmatter(raw)
	if !meta.IsEmpty() {
		t.Errorf("expected empty metadata, got %+v", meta)
	}
	if body != raw {
		t.Errorf("body = %q, want whole content %q", body, raw)
	}
}

func TestParseMalformedFrontmatter(t *testing.T) {
	// Opening delimiter present, but the YAML is invalid. Must degrade to
	// body-only (whole content) rather than erroring.
	raw := "---\nstatus: [unterminated\n  : : :\n---\nbody text\n"
	meta, body := parseFrontmatter(raw)
	if !meta.IsEmpty() {
		t.Errorf("malformed frontmatter should yield empty metadata, got %+v", meta)
	}
	if body != raw {
		t.Errorf("malformed frontmatter should keep whole content as body; got %q", body)
	}
}

func TestParseUnterminatedFrontmatter(t *testing.T) {
	// Opening delimiter but no closing one — treat the whole file as body.
	raw := "---\ntitle: Oops\nno closing delimiter\n"
	meta, body := parseFrontmatter(raw)
	if !meta.IsEmpty() {
		t.Errorf("unterminated frontmatter should yield empty metadata, got %+v", meta)
	}
	if body != raw {
		t.Errorf("unterminated frontmatter should keep whole content as body; got %q", body)
	}
}

func TestSerializeEmptyMetaIsBodyOnly(t *testing.T) {
	body := "# Plain\n\nProse.\n"
	out, err := serializeScene(Metadata{}, body)
	if err != nil {
		t.Fatalf("serializeScene: %v", err)
	}
	if out != body {
		t.Errorf("empty metadata should serialize to body only; got %q", out)
	}
}

// TestRoundTrip proves serialize → parse preserves metadata and body exactly.
func TestRoundTrip(t *testing.T) {
	created := time.Date(2026, 5, 24, 14, 0, 0, 0, time.UTC)
	modified := time.Date(2026, 5, 31, 22, 30, 0, 0, time.UTC)
	meta := Metadata{
		Title:      "Arrival",
		Status:     StatusDone,
		Synopsis:   "She steps onto the platform alone.",
		Tags:       []string{"opening", "rain"},
		DraftOrder: 1,
		WordTarget: 1500,
		POV:        "first",
		WordCount:  1247,
		Created:    &created,
		Modified:   &modified,
	}
	body := "# Arrival\n\nThe train pulled in at dusk.\n\nShe stepped down.\n"

	out, err := serializeScene(meta, body)
	if err != nil {
		t.Fatalf("serializeScene: %v", err)
	}

	gotMeta, gotBody := parseFrontmatter(out)

	if gotBody != body {
		t.Errorf("body not preserved:\n got %q\nwant %q", gotBody, body)
	}
	if gotMeta.Title != meta.Title || gotMeta.Status != meta.Status ||
		gotMeta.Synopsis != meta.Synopsis || gotMeta.DraftOrder != meta.DraftOrder ||
		gotMeta.WordTarget != meta.WordTarget || gotMeta.POV != meta.POV ||
		gotMeta.WordCount != meta.WordCount {
		t.Errorf("scalar metadata not preserved:\n got %+v\nwant %+v", gotMeta, meta)
	}
	if len(gotMeta.Tags) != len(meta.Tags) || gotMeta.Tags[0] != "opening" || gotMeta.Tags[1] != "rain" {
		t.Errorf("tags not preserved: got %v want %v", gotMeta.Tags, meta.Tags)
	}
	if gotMeta.Created == nil || !gotMeta.Created.Equal(created) {
		t.Errorf("created not preserved: got %v want %v", gotMeta.Created, created)
	}
	if gotMeta.Modified == nil || !gotMeta.Modified.Equal(modified) {
		t.Errorf("modified not preserved: got %v want %v", gotMeta.Modified, modified)
	}

	// Serializing the parsed result again must be byte-identical (stable order).
	out2, err := serializeScene(gotMeta, gotBody)
	if err != nil {
		t.Fatalf("second serializeScene: %v", err)
	}
	if out2 != out {
		t.Errorf("serialization not stable across round-trip:\n first:\n%s\n second:\n%s", out, out2)
	}
}

// TestReadMetadataMatchesParse confirms the lazy frontmatter-only reader used
// by BuildTree returns the same metadata that the full parser would, across the
// frontmatter / no-frontmatter / malformed / unterminated cases. (readMetadata
// stops at the closing delimiter instead of reading whole files.)
func TestReadMetadataMatchesParse(t *testing.T) {
	cases := map[string]string{
		"with-frontmatter.md": "---\ntitle: T\nstatus: done\ntags:\n  - a\n  - b\n---\n# Body\n\nlots of prose\n",
		"no-frontmatter.md":   "# Just Prose\n\nno header\n",
		"malformed.md":        "---\nstatus: [bad\n : :\n---\nbody\n",
		"unterminated.md":     "---\ntitle: Oops\nno close\n",
		"crlf.md":             "---\r\nstatus: revised\r\n---\r\n# Body\r\n",
	}
	tmp := t.TempDir()
	for name, content := range cases {
		path := filepath.Join(tmp, name)
		if err := os.WriteFile(path, []byte(content), scenePerm); err != nil {
			t.Fatal(err)
		}
		want, _ := parseFrontmatter(content)
		got := readMetadata(path)
		if got.Status != want.Status || got.Title != want.Title ||
			len(got.Tags) != len(want.Tags) {
			t.Errorf("%s: readMetadata = %+v, want (from parseFrontmatter) %+v", name, got, want)
		}
	}
}

// TestBodyPreservedWithBlankLines ensures multi-paragraph bodies (blank lines,
// trailing newline) survive the frontmatter split byte-for-byte.
func TestBodyPreservedWithBlankLines(t *testing.T) {
	body := "# Title\n\nFirst paragraph.\n\n\nThird after a double blank.\n\n"
	raw := "---\nstatus: draft\n---\n" + body
	_, gotBody := parseFrontmatter(raw)
	if gotBody != body {
		t.Errorf("body with blank lines not preserved:\n got %q\nwant %q", gotBody, body)
	}
	if strings.Count(gotBody, "\n") != strings.Count(body, "\n") {
		t.Errorf("newline count changed: got %d want %d",
			strings.Count(gotBody, "\n"), strings.Count(body, "\n"))
	}
}
