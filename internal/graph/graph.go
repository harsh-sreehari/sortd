package graph

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/harsh-sreehari/sortd/internal/store"
)

type Graph struct {
	Store *store.Store
}

type FolderIndex struct {
	Path     string
	Keywords []string
	Depth    int
}

func TokenisePath(folderName string) []string {
	// Split on separators: -, _, space
	f := func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	}
	parts := strings.FieldsFunc(folderName, f)

	// CamelCase split logic
	var tokens []string
	for _, part := range parts {
		if part == "" {
			continue
		}
		
		var currentToken strings.Builder
		for i, r := range part {
			if i > 0 && unicode.IsUpper(r) && !unicode.IsUpper(rune(part[i-1])) {
				tokens = append(tokens, strings.ToLower(currentToken.String()))
				currentToken.Reset()
			}
			currentToken.WriteRune(r)
		}
		tokens = append(tokens, strings.ToLower(currentToken.String()))
	}

	// Deduplicate and filter small tokens
	seen := make(map[string]bool)
	var final []string
	for _, t := range tokens {
		t = strings.TrimSpace(t)
		if len(t) > 2 && !seen[t] {
			final = append(final, t)
			seen[t] = true
		}
	}

	return final
}

func (g *Graph) Crawl(roots []string, ignore []string) error {
	for _, root := range roots {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip folders we can't read
			}

			if !info.IsDir() {
				return nil
			}

			// Depth check (TODO)
			
			// Ignore check
			for _, ig := range ignore {
				if strings.Contains(path, ig) {
					return filepath.SkipDir
				}
			}

			// Hidden folder skip
			if strings.HasPrefix(info.Name(), ".") && path != root {
				return filepath.SkipDir
			}

			// Tokenize folder name
			tokens := TokenisePath(info.Name())
			if len(tokens) == 0 {
				return nil
			}

			// Store in index
			keywords := strings.Join(tokens, ",")
			
			// Get parent folder if any
			parent := filepath.Dir(path)
			depth := len(strings.Split(path, string(filepath.Separator)))

			_, err = g.Store.DB().Exec("INSERT OR REPLACE INTO folder_index (path, keywords, depth, parent) VALUES (?, ?, ?, ?)", path, keywords, depth, parent)
			if err != nil {
				log.Printf("Failed to index folder %s: %v", path, err)
			}
			return nil
		})

		if err != nil {
			return err
		}
	}
	return nil
}

func (g *Graph) GetAllPaths() ([]string, error) {
	rows, err := g.Store.DB().Query("SELECT path FROM folder_index")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	return paths, nil
}

func (g *Graph) ListFolders() ([]FolderIndex, error) {
	rows, err := g.Store.DB().Query("SELECT path, keywords, depth FROM folder_index")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indices []FolderIndex
	for rows.Next() {
		var f FolderIndex
		var keywordsStr string
		if err := rows.Scan(&f.Path, &keywordsStr, &f.Depth); err != nil {
			return nil, err
		}
		f.Keywords = strings.Split(keywordsStr, ",")
		indices = append(indices, f)
	}
	return indices, nil
}
