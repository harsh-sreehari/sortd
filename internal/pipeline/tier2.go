package pipeline

import (
	"fmt"
	"path/filepath"

	"github.com/harsh-sreehari/sortd/internal/graph"
)

func MatchTier2(path string, folders []graph.FolderIndex) (Decision, bool) {
	filename := filepath.Base(path)
	fileTokens := graph.TokenisePath(filename)

	if len(fileTokens) == 0 {
		return Decision{}, false
	}

	bestScore := 0.0
	bestFolder := ""

	for _, folder := range folders {
		score := calculateOverlap(fileTokens, folder.Keywords)
		if score > bestScore {
			bestScore = score
			bestFolder = folder.Path
		}
	}

	threshold := 0.75 // Spec-mandated minimum; keeps false-confident matches from
	                   // bypassing Tier 3. Improve scoring algorithm if recall is low,
	                   // not the threshold.
	if bestScore >= threshold {
		return Decision{
			Path:        path,
			Destination: bestFolder,
			Confidence:  bestScore,
			Tier:        2,
			Action:      "moved",
			Reasoning:   fmt.Sprintf("Tier 2: Fuzzy match on folder '%s' (score %.2f)", filepath.Base(bestFolder), bestScore),
		}, true
	}

	return Decision{}, false
}

func MatchDescription(desc string, folders []graph.FolderIndex) (string, bool) {
	tokens := graph.TokenisePath(desc)
	if len(tokens) == 0 {
		return "", false
	}

	bestScore := 0.0
	bestFolder := ""

	for _, folder := range folders {
		score := calculateOverlap(tokens, folder.Keywords)
		if score > bestScore {
			bestScore = score
			bestFolder = folder.Path
		}
	}

	threshold := 0.50 // Stricter threshold for NL descriptions
	if bestScore >= threshold {
		return bestFolder, true
	}

	return "", false
}

func calculateOverlap(fileTokens, folderKeywords []string) float64 {
	if len(folderKeywords) == 0 {
		return 0
	}

	matchCount := 0
	fileSet := make(map[string]bool)
	for _, t := range fileTokens {
		fileSet[t] = true
	}

	for _, k := range folderKeywords {
		if fileSet[k] {
			matchCount++
		}
	}

	return float64(matchCount) / float64(len(folderKeywords))
}
