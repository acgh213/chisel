package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_MissingFile(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig on missing file: %v", err)
	}
	if cfg.Theme != "" || cfg.DailyGoal != 0 {
		t.Errorf("expected empty config on missing file, got %+v", cfg)
	}
}

func TestLoadConfig_SaveConfig_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	want := ChiselConfig{Theme: "ocean", DailyGoal: 500}
	if err := SaveConfig(dir, want); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	got, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if got.Theme != want.Theme || got.DailyGoal != want.DailyGoal {
		t.Errorf("round-trip failed: got %+v, want %+v", got, want)
	}
}

func TestLoadConfig_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, configFileName), []byte("theme: [unclosed"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadConfig(dir)
	if err == nil {
		t.Error("expected error on malformed YAML, got nil")
	}
}
