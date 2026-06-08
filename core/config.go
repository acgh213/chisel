package core

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ChiselConfig holds project-level app preferences. It is persisted to
// .chisel.yaml at the project root. The "filesystem is the project" rule
// applies to scene/content data only; app preferences like theme live here.
type ChiselConfig struct {
	Theme         string   `yaml:"theme,omitempty"`
	DailyGoal     int      `yaml:"daily_goal,omitempty"`
	ProjectTarget int      `yaml:"project_target,omitempty"`
	Bookmarks     []string `yaml:"bookmarks,omitempty"`
}

const configFileName = ".chisel.yaml"

// LoadConfig reads .chisel.yaml from the project root. If the file does not
// exist it returns an empty ChiselConfig (use defaults) with no error.
func LoadConfig(root string) (ChiselConfig, error) {
	data, err := os.ReadFile(filepath.Join(root, configFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return ChiselConfig{}, nil
		}
		return ChiselConfig{}, err
	}
	var cfg ChiselConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return ChiselConfig{}, err
	}
	return cfg, nil
}

// SaveConfig writes cfg to .chisel.yaml in the project root.
func SaveConfig(root string, cfg ChiselConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, configFileName), data, 0o644)
}
