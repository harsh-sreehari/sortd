package pipeline

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/harsh-sreehari/sortd/internal/config"
	"github.com/harsh-sreehari/sortd/internal/graph"
	"github.com/harsh-sreehari/sortd/internal/llm"
	"github.com/harsh-sreehari/sortd/internal/mover"
	"github.com/harsh-sreehari/sortd/internal/store"
	"os"
	"regexp"
	"strings"
)

type Pipeline struct {
	cfg    *config.Config
	Store  *store.Store
	Graph  *graph.Graph
	LLM    llm.LLMBackend
	Mover  *mover.Mover
	AllowedRoots []string
	NotifyFunc   func(title, message string)
}

func New(cfg *config.Config, s *store.Store, g *graph.Graph, l llm.LLMBackend, m *mover.Mover, notify func(string, string)) *Pipeline {
	return &Pipeline{
		cfg:          cfg,
		Store:        s,
		Graph:        g,
		LLM:          l,
		Mover:        m,
		AllowedRoots: []string{}, // populated by caller via SetAllowedRoots()
		NotifyFunc:   notify,
	}
}

// SetAllowedRoots replaces the pipeline's allowed root list.
// Called by initPipeline after deriving roots from config or index crawl.
func (p *Pipeline) SetAllowedRoots(roots []string) {
	p.AllowedRoots = roots
}

func (p *Pipeline) Match(path string) Decision {
	var decision Decision
	var match bool

	matchPath := path
	if p.cfg.Behaviour.AutoRename {
		matchPath = p.cleanPath(path)
	}

	// Tier 1: Rules
	if decision, match = MatchTier1(matchPath); match {
		if !filepath.IsAbs(decision.Destination) {
			home, _ := os.UserHomeDir()
			decision.Destination = filepath.Join(home, decision.Destination)
		}
		return decision
	}

	// Tier 2: Fuzzy (needs folder keywords)
	{
		indices, _ := p.Graph.ListFolders()
		if decision, match = MatchTier2(matchPath, indices); match {
			return decision
		}
	}

	// Tier 3: LLM
	{
		tree, _ := p.Graph.GetAllPaths()
		affinities, _ := p.Store.GetAffinities(nil)
		if decision, match = MatchTier3(matchPath, p.LLM, tree, p.AllowedRoots, p.cfg.Behaviour.ConfidenceThreshold, affinities); match {
			return decision
		}
	}

	// Default: Park
	return Decision{
		Path:           path,
		OriginalSource: filepath.Dir(path),
		Action:         "parked",
		Tier:           0,
	}
}

func (p *Pipeline) Process(path string) Decision {
	decision := p.Match(path)

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

	if decision.Action == "moved" && p.cfg.Behaviour.Notifications && p.NotifyFunc != nil {
		p.NotifyFunc("File Organized", fmt.Sprintf("Sorted %s to %s", filepath.Base(path), filepath.Base(decision.Destination)))
	}

	if decision.Action == "moved" && p.cfg.Behaviour.Xattr {
		p.Mover.WriteXattr(decision.Destination, decision.Tags)
	}

	return decision
}

func (p *Pipeline) cleanPath(originalPath string) string {
	dir := filepath.Dir(originalPath)
	base := filepath.Base(originalPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// Strip common suffixes: (1), [1], _1, -1, " copy"
	re := regexp.MustCompile(`(?i)( \(\d+\)|_\d+|\[\d+\]| copy)$`)
	name = re.ReplaceAllString(name, "")

	// Normalize separators for better keyword matching
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.TrimSpace(name)

	return filepath.Join(dir, name+ext)
}

func (p *Pipeline) logDecision(d Decision) {
	log.Printf("PIPELINE [%d] -> %s -> %s (%0.2f)", d.Tier, d.Action, d.Destination, d.Confidence)
	p.Store.LogDecision(store.Decision{
		File:             d.Path,
		OriginalFilename: filepath.Base(d.Path),
		OriginalSource:   d.OriginalSource,
		Destination:      d.Destination,
		Tier:             d.Tier,
		Confidence:       d.Confidence,
		Action:           d.Action,
		Tags:             d.Tags,
		Reasoning:        d.Reasoning,
	})
}
