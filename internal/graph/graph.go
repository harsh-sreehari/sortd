package graph

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"database/sql"
	"sort"
	"github.com/harsh-sreehari/sortd/internal/store"
)

type Graph struct {
	Store *store.Store
}

type FolderIndex struct {
	Path     string
	Keywords []string
	Depth    int
	Schema   string
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
	// 0. Wipe existing index to prevent "ghost" folders
	_, err := g.Store.DB().Exec("DELETE FROM folder_index")
	if err != nil {
		log.Printf("Failed to clear index: %v", err)
	}

	folderCount := 0

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

			// FEAT-12: Index sibling filenames as additional keywords
			dirEntries, _ := os.ReadDir(path)
			var fileTokens []string
			for _, e := range dirEntries {
				if !e.IsDir() {
					fTokens := TokenisePath(strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())))
					fileTokens = append(fileTokens, fTokens...)
				}
				if len(fileTokens) > 100 {
					break
				}
			}
			tokens = deduplicate(append(tokens, fileTokens...))

			// Store in index
			keywords := strings.Join(tokens, ",")
			
			// Get parent folder if any
			parent := filepath.Dir(path)
			depth := len(strings.Split(path, string(filepath.Separator)))
			schema := inferSchema(path)

			_, err = g.Store.DB().Exec("INSERT OR REPLACE INTO folder_index (path, keywords, depth, parent, schema) VALUES (?, ?, ?, ?, ?)", path, keywords, depth, parent, schema)
			if err != nil {
				log.Printf("Failed to index folder %s: %v", path, err)
			} else {
				folderCount++
				fmt.Printf("\rIndexing... found %d folders", folderCount)
			}
			return nil
		})

		if err != nil {
			return err
		}
	}
	fmt.Println() // Newline after progress
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
	rows, err := g.Store.DB().Query("SELECT path, keywords, depth, schema FROM folder_index")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indices []FolderIndex
	for rows.Next() {
		var f FolderIndex
		var keywordsStr string
		var schema sql.NullString
		if err := rows.Scan(&f.Path, &keywordsStr, &f.Depth, &schema); err != nil {
			return nil, err
		}
		f.Keywords = strings.Split(keywordsStr, ",")
		f.Schema = schema.String
		indices = append(indices, f)
	}
	return indices, nil
}

func deduplicate(tokens []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, t := range tokens {
		if !seen[t] {
			result = append(result, t)
			seen[t] = true
		}
	}
	return result
}

func inferSchema(path string) string {
	parent := filepath.Dir(path)
	entries, _ := os.ReadDir(parent)
	
	patterns := make(map[string]int)
	for _, e := range entries {
		if e.IsDir() {
			name := e.Name()
			var p strings.Builder
			for _, r := range name {
				if unicode.IsDigit(r) {
					p.WriteString(`\d`)
				} else if unicode.IsLetter(r) {
					p.WriteString(`[a-zA-Z]`)
				} else {
					p.WriteRune(r)
				}
			}
			patterns[p.String()]++
		}
	}
	
	for pat, count := range patterns {
		if count >= 3 {
			return pat
		}
	}
	return ""
}

func (g *Graph) PrintTree() {
	folders, err := g.ListFolders()
	if err != nil {
		fmt.Printf("Failed to list folders: %v\n", err)
		return
	}

	tree := make(map[string][]string)
	paths := make(map[string]bool)
	for _, f := range folders {
		paths[f.Path] = true
	}

	var roots []string
	for _, f := range folders {
		parent := filepath.Dir(f.Path)
		if paths[parent] {
			tree[parent] = append(tree[parent], f.Path)
		} else {
			roots = append(roots, f.Path)
		}
	}

	sort.Strings(roots)
	for i, r := range roots {
		isLast := i == len(roots)-1
		g.printBranch(r, "", isLast, tree)
	}
}

func (g *Graph) printBranch(path string, indent string, isLast bool, tree map[string][]string) {
	marker := "├── "
	if isLast {
		marker = "└── "
	}
	fmt.Printf("%s%s%s\n", indent, marker, filepath.Base(path))

	newIndent := indent
	if isLast {
		newIndent += "    "
	} else {
		newIndent += "│   "
	}

	children := tree[path]
	sort.Strings(children)
	for i, child := range children {
		isChildLast := i == len(children)-1
		g.printBranch(child, newIndent, isChildLast, tree)
	}
}
