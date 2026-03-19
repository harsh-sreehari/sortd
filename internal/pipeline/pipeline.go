package pipeline

import (
	"log"
	"path/filepath"

	"github.com/harsh-sreehari/sortd/internal/config"
	"github.com/harsh-sreehari/sortd/internal/graph"
	"github.com/harsh-sreehari/sortd/internal/llm"
	"github.com/harsh-sreehari/sortd/internal/mover"
	"github.com/harsh-sreehari/sortd/internal/store"
	"os"
	"strings"
)

type Pipeline struct {
	cfg    *config.Config
	Store  *store.Store
	Graph  *graph.Graph
	LLM    llm.LLMBackend
	Mover  *mover.Mover
	AllowedRoots []string
}

func New(cfg *config.Config, s *store.Store, g *graph.Graph, l llm.LLMBackend, m *mover.Mover) *Pipeline {
	return &Pipeline{
		cfg:   cfg,
		Store: s,
		Graph: g,
		LLM:   l,
		Mover: m,
		AllowedRoots: []string{"Downloads/", "Documents/", "Videos/", "Pictures/", "Music/"},
	}
}

func (p *Pipeline) Process(path string) Decision {
	var decision Decision
	var match bool

	// Tier 1: Rules
	if decision, match = MatchTier1(path); match {
		goto Execution
	}

	// Tier 2: Fuzzy (needs folder keywords)
	{
		indices, _ := p.Graph.ListFolders()
		if decision, match = MatchTier2(path, indices); match {
			goto Execution
		}
	}

	// Tier 3: LLM
	{
		tree, _ := p.Graph.GetAllPaths()
		if decision, match = MatchTier3(path, p.LLM, tree, p.AllowedRoots, p.cfg.Behaviour.ConfidenceThreshold); match {
			goto Execution
		}
	}

	// Default: Park
	decision = Decision{
		Path:   path,
		Action: "parked",
		Tier:   0,
	}

Execution:
	// Execute the action with the mover
	if decision.Action == "skipped" {
		p.logDecision(decision)
		return decision
	}

	var root string
	if len(p.cfg.Watch.Folders) > 0 {
		root = p.cfg.Watch.Folders[0]
	}

	var finalPath string
	var err error

	if decision.Action == "moved" {
		dest := decision.Destination
		if !filepath.IsAbs(dest) {
			// Resolve relative paths to Home
			home, _ := os.UserHomeDir()
			
			// Validate restriction: Must start with one of the allowed categories
			isAllowed := false
			for _, a := range p.AllowedRoots {
				if strings.HasPrefix(strings.ToLower(dest), strings.ToLower(a)) {
					isAllowed = true
					break
				}
			}

			if !isAllowed && decision.Tier == 3 {
				// LLM tried to go outside allowed roots (e.g. "Research/")
				// Force it into Documents/
				dest = filepath.Join("Documents/", dest)
			}

			dest = filepath.Join(home, dest)
		}
		
		// ALWAYS join filename to the destination to ensure it stays a directory
		dest = filepath.Join(dest, filepath.Base(path))
		finalPath, err = p.Mover.Move(path, dest)
	} else {
		finalPath, err = p.Mover.Park(path, root)
		decision.Destination = finalPath
	}

	if err != nil {
		log.Printf("Failed to move/park file %s: %v", path, err)
		return decision
	}

	decision.Destination = finalPath
	if decision.Action == "moved" && finalPath == path {
		decision.Action = "skipped"
		decision.Reasoning = "Already at destination"
	}

	p.logDecision(decision)
	return decision
}

func (p *Pipeline) logDecision(d Decision) {
	log.Printf("PIPELINE [%d] -> %s -> %s (%0.2f)", d.Tier, d.Action, d.Destination, d.Confidence)
	p.Store.LogDecision(store.Decision{
		File:        d.Path,
		Destination: d.Destination,
		Tier:        d.Tier,
		Confidence:  d.Confidence,
		Action:      d.Action,
		Tags:        d.Tags,
		Reasoning:   d.Reasoning,
	})
}
