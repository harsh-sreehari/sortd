package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Watch.Folders[0] != "~/Downloads" {
		t.Errorf("expected default watch folder ~/Downloads, got %s", cfg.Watch.Folders[0])
	}
	if cfg.Behaviour.DebounceSeconds != 2 {
		t.Errorf("expected default debounce 2s, got %d", cfg.Behaviour.DebounceSeconds)
	}
}

func TestLoadConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sortd-config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.toml")
	tomlData := `
[watch]
folders = ["/tmp/test"]

[behaviour]
debounce_seconds = 5
`
	if err := os.WriteFile(configPath, []byte(tomlData), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Watch.Folders[0] != "/tmp/test" {
		t.Errorf("expected folder /tmp/test, got %s", cfg.Watch.Folders[0])
	}
	if cfg.Behaviour.DebounceSeconds != 5 {
		t.Errorf("expected debounce 5s, got %d", cfg.Behaviour.DebounceSeconds)
	}
}
