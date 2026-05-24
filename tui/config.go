package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds all project-level settings loaded from config.json.
type Config struct {
	LLM     LLMConfig     `json:"llm"`
	Mirror  MirrorConfig  `json:"mirror"`
	History HistoryConfig `json:"history"`
	Editor  EditorConfig  `json:"editor"`
	Goals   GoalsConfig   `json:"goals"`
	Theme   string        `json:"theme"`
}

// LLMConfig holds the general-purpose LLM model slot.
type LLMConfig struct {
	APIBase     string  `json:"api_base"`
	Model       string  `json:"model"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
}

// MirrorConfig holds the stylistic-analysis mirror model slot.
type MirrorConfig struct {
	APIBase     string  `json:"api_base"`
	Model       string  `json:"model"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
}

// HistoryConfig holds revision-history settings.
type HistoryConfig struct {
	Backend string `json:"backend"`
}

// EditorConfig holds editor behaviour settings.
type EditorConfig struct {
	VimMode bool `json:"vim_mode"`
}

// GoalsConfig holds writing goal settings.
type GoalsConfig struct {
	DailyWordTarget int `json:"daily_word_target"`
}

// DefaultConfig returns a Config populated with safe defaults.
func DefaultConfig() Config {
	return Config{
		LLM: LLMConfig{
			APIBase:     "http://localhost:1234/v1",
			Model:       "",
			MaxTokens:   2048,
			Temperature: 0.7,
		},
		Mirror: MirrorConfig{
			APIBase:     "http://localhost:1234/v1",
			Model:       "",
			MaxTokens:   1024,
			Temperature: 0.3,
		},
		History: HistoryConfig{
			Backend: "git",
		},
		Editor: EditorConfig{
			VimMode: false,
		},
		Goals: GoalsConfig{
			DailyWordTarget: 500,
		},
		Theme: "peach",
	}
}

// LoadConfig reads and unmarshals config.json from the given project directory.
func LoadConfig(projectDir string) (Config, error) {
	cfg := DefaultConfig()

	path := filepath.Join(projectDir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("reading config: %w", err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing config: %w", err)
	}

	return cfg, nil
}

// SaveConfig writes cfg as indented JSON to config.json in the project directory.
func SaveConfig(projectDir string, cfg Config) error {
	path := filepath.Join(projectDir, "config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}
