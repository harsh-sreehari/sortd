package pipeline

import (
	"log"
	"path/filepath"

	"github.com/harsh-sreehari/sortd/internal/llm"
	"github.com/harsh-sreehari/sortd/internal/peek"
)

func MatchTier3(path string, l llm.LLMBackend, folders []string) (Decision, bool) {
	// 1. Peek content
	content := peek.PeekDispatcher(path)

	// 2. Prepare request
	req := llm.TagRequest{
		Filename:    filepath.Base(path),
		Extension:   filepath.Ext(path),
		ContentPeek: content,
		FolderTree:  folders,
	}

	// 3. Ask LLM
	resp, err := l.TagContent(req)
	if err != nil {
		log.Printf("LLM tagging failed: %v", err)
		return Decision{}, false
	}

	// 4. Determine decision
	threshold := 0.70 // Confidence threshold for LLM results
	if resp.Confidence >= threshold {
		return Decision{
			Path:        path,
			Destination: resp.Destination,
			Confidence:  resp.Confidence,
			Tier:        3,
			Action:      "moved",
			Reasoning:   "Tier 3: LLM reasoning: " + resp.Reasoning,
		}, true
	}

	return Decision{
		Path:        path,
		Confidence:  resp.Confidence,
		Tier:        3,
		Action:      "parked",
		Reasoning:   "Tier 3: Low confidence, parking in .unsorted",
	}, true
}
