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
	SplitByType         bool     `toml:"split_by_type"`
	ConfidenceThreshold float64  `toml:"confidence_threshold"`
	CreateFolders       bool     `toml:"create_folders"`
	LogPath             string   `toml:"log_path"`
	DBPath              string   `toml:"db_path"`
	DebounceSeconds     int      `toml:"debounce_seconds"`
	// AllowedRoots constrains where Tier 3 (LLM) can route files.
	// Leave empty to derive automatically from the top-level crawl roots at startup.
	AllowedRoots        []string `toml:"allowed_roots"`
	Notifications       bool     `toml:"notifications"`
	Xattr              bool     `toml:"xattr"`
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
# List of absolute paths to monitor for new file events.
# sortd will automatically ignore hidden files (starting with .) and the .unsorted/ folder.
folders = ["%[1]s/Downloads"]

# Paths to ignore within watched folders. Supports glob patterns.
ignore = []

[llm]
# Your local LLM endpoint (LM Studio or OpenAI-compatible backend).
host = "http://localhost:1234"

# The specific model ID to use for inference.
model = "qwen3-VL-4b"

# Backend type: "lmstudio" (default) or "openai".
backend = "lmstudio"

[behaviour]
# Minimum confidence (0.0-1.0) before the LLM (Tier 3) is allowed to move a file.
# Higher values (0.8+) reduce misclassifications but increase "parked" files.
confidence_threshold = 0.75

# If true, sortd will create the destination folder if it doesn't already exist.
create_folders = true

# Path to the SQLite database where sortd logs decisions and user preferences.
db_path = "%[1]s/.local/share/sortd/sortd.db"

# Path to the human-readable log file.
log_path = "%[1]s/.local/share/sortd/sortd.log"

# Seconds to wait after a file event before processing (prevents partial move issues).
debounce_seconds = 2

# List of top-level folders the LLM is restricted to sorting files into.
# If unset, sortd automatically derives these from your filesystem (Documents, Pictures, etc.).
# allowed_roots = ["Documents", "Pictures", "Videos", "Music"]

# Enable desktop notifications via notify-send for successful sorting actions.
notifications = false

# If true, write sort tags to the file's extended attributes (user.sortd.tags).
# Requires a filesystem that supports xattrs (ext4, xfs, btrfs).
xattr = false
`, home)

	return os.WriteFile(path, []byte(defaultCfg), 0644)
}
