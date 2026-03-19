package pipeline

import (
	"strings"

	"github.com/harsh-sreehari/sortd/internal/graph"
)

func MatchTier2(path string, folders []graph.FolderIndex) (Decision, bool) {
	filename := strings.ToLower(strings.TrimSuffix(path, ".pdf"))
	fileTokens := graph.TokenisePath(filename)

	if len(fileTokens) == 0 {
		return Decision{}, false
	}

	bestScore := 0.0
	bestFolder := ""

	for _, folder := range folders {
		score := calculateJaccard(fileTokens, folder.Keywords)
		if score > bestScore {
			bestScore = score
			bestFolder = folder.Path
		}
	}

	threshold := 0.75 // Default confidence threshold
	if bestScore >= threshold {
		return Decision{
			Path:        path,
			Destination: bestFolder,
			Confidence:  bestScore,
			Tier:        2,
			Action:      "moved",
			Reasoning:   "Tier 2: Fuzzy similarity match",
		}, true
	}

	return Decision{}, false
}

func calculateJaccard(a, b []string) float64 {
	intersection := 0
	setA := make(map[string]bool)
	for _, s := range a {
		setA[s] = true
	}

	for _, s := range b {
		if setA[s] {
			intersection++
		}
	}

	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}
