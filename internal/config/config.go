package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Watch     WatchConfig     `toml:"watch"`
	LLM       LLMConfig       `toml:"llm"`
	Behaviour BehaviourConfig `toml:"behaviour"`
}

type WatchConfig struct {
	Folders []string `toml:"folders"`
	Ignore  []string `toml:"ignore"`
}

type LLMConfig struct {
	Backend string `toml:"backend"`
	Host    string `toml:"host"`
	Model   string `toml:"model"`
}

type BehaviourConfig struct {
	SplitByType         bool    `toml:"split_by_type"`
	ConfidenceThreshold float64 `toml:"confidence_threshold"`
	CreateFolders       bool    `toml:"create_folders"`
	LogPath             string  `toml:"log_path"`
	DBPath              string  `toml:"db_path"`
	DebounceSeconds     int     `toml:"debounce_seconds"`
}

func DefaultConfig() *Config {
	return &Config{
		Watch: WatchConfig{
			Folders: []string{"~/Downloads"},
			Ignore:  []string{},
		},
		LLM: LLMConfig{
			Backend: "lmstudio",
			Host:    "http://localhost:1234",
			Model:   "default",
		},
		Behaviour: BehaviourConfig{
			SplitByType:         false,
			ConfidenceThreshold: 0.75,
			CreateFolders:       true,
			LogPath:             "~/.local/share/sortd/sortd.log",
			DBPath:              "~/.local/share/sortd/sortd.db",
			DebounceSeconds:     2,
		},
	}
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func LoadConfig(path string) (*Config, error) {
	path = expandHome(path)
	config := DefaultConfig()

	// Parse file
	_, err := toml.DecodeFile(path, config)
	if err != nil && os.IsNotExist(err) {
		if err := writeDefaultConfig(path); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	// Expand paths
	for i, folder := range config.Watch.Folders {
		config.Watch.Folders[i] = expandHome(folder)
	}
	for i, folder := range config.Watch.Ignore {
		config.Watch.Ignore[i] = expandHome(folder)
	}

	config.Behaviour.LogPath = expandHome(config.Behaviour.LogPath)
	config.Behaviour.DBPath = expandHome(config.Behaviour.DBPath)

	return config, nil
}

func writeDefaultConfig(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	home, _ := os.UserHomeDir()
	
	defaultCfg := fmt.Sprintf(`[watch]
# Directives to monitor
folders = ["%[1]s/Downloads"]

[llm]
# Your local LLM endpoint (LM Studio/Ollama)
host = "http://localhost:1234"
model = "qwen3-VL-4b"

[behaviour]
# Minimum confidence (0.0-1.0) before a move is performed
confidence_threshold = 0.75
create_folders = true
db_path = "%[1]s/.local/share/sortd/sortd.db"
log_path = "%[1]s/.local/share/sortd/sortd.log"
`, home)

	return os.WriteFile(path, []byte(defaultCfg), 0644)
}
