package pipeline

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

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
		score := jaccardSimilarity(fileTokens, folder.Keywords)
		
		// FEAT-07: Schema match boost
		if folder.Schema != "" && matchSchema(filename, folder.Schema) {
			score += 0.5 // High boost for matching an inferred schema pattern
		}
		
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
			Path:           path,
			OriginalSource: filepath.Dir(path),
			Destination:    bestFolder,
			Confidence:     bestScore,
			Tier:           2,
			Action:         "moved",
			Reasoning:      "Tier 2: Fuzzy match on folder '" + filepath.Base(bestFolder) + "' (score " + fmt.Sprintf("%.2f", bestScore) + ")",
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
		score := jaccardSimilarity(tokens, folder.Keywords)
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

func jaccardSimilarity(fileTokens, folderKeywords []string) float64 {
	fileSet := make(map[string]bool)
	for _, t := range fileTokens {
		fileSet[stem(t)] = true
	}
	folderSet := make(map[string]bool)
	for _, k := range folderKeywords {
		folderSet[stem(k)] = true
	}
	intersection := 0
	for k := range fileSet {
		if folderSet[k] {
			intersection++
		}
	}
	union := len(fileSet) + len(folderSet) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func matchSchema(filename, schema string) bool {
	var p strings.Builder
	for _, r := range filename {
		if unicode.IsDigit(r) {
			p.WriteString(`\d`)
		} else if unicode.IsLetter(r) {
			p.WriteString(`[a-zA-Z]`)
		} else {
			p.WriteRune(r)
		}
	}
	// Check if the filename shape matches the folder's inferred schema
	return strings.Contains(p.String(), schema)
}

func stem(word string) string {
	// simple suffix stripping — no external library needed
	suffixes := []string{"ing", "tion", "ed", "er", "s"}
	for _, s := range suffixes {
		if len(word) > len(s)+3 && strings.HasSuffix(word, s) {
			return strings.TrimSuffix(word, s)
		}
	}
	return word
}
