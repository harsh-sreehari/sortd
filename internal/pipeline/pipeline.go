package pipeline

import (
	"log"

	"github.com/harsh-sreehari/sortd/internal/config"
	"github.com/harsh-sreehari/sortd/internal/graph"
	"github.com/harsh-sreehari/sortd/internal/llm"
	"github.com/harsh-sreehari/sortd/internal/store"
)

type Pipeline struct {
	cfg    *config.Config
	Store  *store.Store
	Graph  *graph.Graph
	LLM    llm.LLMBackend
}

func New(cfg *config.Config, s *store.Store, g *graph.Graph, l llm.LLMBackend) *Pipeline {
	return &Pipeline{
		cfg:   cfg,
		Store: s,
		Graph: g,
		LLM:   l,
	}
}

func (p *Pipeline) Process(path string) Decision {
	// Tier 1: Rules
	if decision, ok := MatchTier1(path); ok {
		p.logDecision(decision)
		return decision
	}

	// Tier 2: Fuzzy (needs folder keywords)
	// folders := p.Store.GetFolders() 
	folders := []graph.FolderIndex{} // Placeholder
	if decision, ok := MatchTier2(path, folders); ok {
		p.logDecision(decision)
		return decision
	}

	// Tier 3: LLM
	// tree := p.Store.GetFolderPaths()
	tree := p.cfg.Watch.Folders // Placeholder
	if decision, ok := MatchTier3(path, p.LLM, tree); ok {
		p.logDecision(decision)
		return decision
	}

	// Default: Park
	decision := Decision{
		Path:   path,
		Action: "parked",
		Tier:   0,
	}
	p.logDecision(decision)
	return decision
}

func (p *Pipeline) logDecision(d Decision) {
	log.Printf("PIPELINE [%d] -> %s -> %s (%0.2f)", d.Tier, d.Action, d.Destination, d.Confidence)
	// p.Store.LogDecision(d)
}
