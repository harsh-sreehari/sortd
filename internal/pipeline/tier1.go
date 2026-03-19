package pipeline

import (
	"path/filepath"
	"strings"
)

type Decision struct {
	Path        string
	Destination string
	Confidence  float64
	Tier        int
	Action      string
	Reasoning   string
}

type Rule struct {
	Extensions  []string
	Pattern     string
	Destination string
	Action      string
}

var Tier1Rules = []Rule{
	// Software
	{Extensions: []string{".appimage", ".deb", ".rpm", ".flatpak", ".iso", ".exe", ".msi"}, Destination: "Software/"},
	// Skip rules
	{Extensions: []string{".crdownload", ".part", ".tmp", ".download", ".torrent"}, Action: "skipped"},
}

func MatchTier1(path string) (Decision, bool) {
	filename := filepath.Base(path)
	ext := strings.ToLower(filepath.Ext(filename))

	for _, rule := range Tier1Rules {
		// Extension match
		for _, rExt := range rule.Extensions {
			if ext == rExt {
				return Decision{
					Path:        path,
					Destination: rule.Destination,
					Confidence:  1.0,
					Tier:        1,
					Action:      rule.Action,
					Reasoning:   "Tier 1: Extension rule match",
				}, true
			}
		}

		// Pattern match (TODO: if pattern is set)
	}

	return Decision{}, false
}
