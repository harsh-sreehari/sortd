package pipeline

import (
	"log"
	"path/filepath"

	"github.com/harsh-sreehari/sortd/internal/config"
	"github.com/harsh-sreehari/sortd/internal/graph"
	"github.com/harsh-sreehari/sortd/internal/llm"
	"github.com/harsh-sreehari/sortd/internal/mover"
	"github.com/harsh-sreehari/sortd/internal/store"
)

type Pipeline struct {
	cfg    *config.Config
	Store  *store.Store
	Graph  *graph.Graph
	LLM    llm.LLMBackend
	Mover  *mover.Mover
}

func New(cfg *config.Config, s *store.Store, g *graph.Graph, l llm.LLMBackend, m *mover.Mover) *Pipeline {
	return &Pipeline{
		cfg:   cfg,
		Store: s,
		Graph: g,
		LLM:   l,
		Mover: m,
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
		indices, _ := p.Graph.GetFolderIndices()
		if decision, match = MatchTier2(path, indices); match {
			goto Execution
		}
	}

	// Tier 3: LLM
	{
		tree, _ := p.Graph.GetAllPaths()
		if decision, match = MatchTier3(path, p.LLM, tree); match {
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

	if decision.Action == "moved" || decision.Action == "Software/" {
		dest := decision.Destination
		if !filepath.IsAbs(dest) {
			parent := filepath.Dir(root)
			dest = filepath.Join(parent, dest)
		}
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
	})
}
