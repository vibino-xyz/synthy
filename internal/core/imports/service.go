package imports

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	cfile "github.com/vibino-xyz/synthy/internal/core/file"
	"golang.org/x/tools/go/packages"
)

type ImportEdgeService struct {
	repo ImportEdgeRepository
}

func NewImportEdgeService(repo ImportEdgeRepository) *ImportEdgeService {
	return &ImportEdgeService{repo: repo}
}

// BuildAndSave extracts import edges from pkgs, persists them, and returns the saved edges.
func (s *ImportEdgeService) BuildAndSave(ctx context.Context, repositoryID string, fileMap map[string]*cfile.File, pkgs []*packages.Package) ([]*ImportEdge, error) {
	edges := Build(repositoryID, fileMap, pkgs)
	if len(edges) == 0 {
		return edges, nil
	}
	if err := s.repo.InsertImportEdges(ctx, edges); err != nil {
		return nil, fmt.Errorf("insert import edges: %w", err)
	}
	return edges, nil
}

// Build extracts import edges from per-file import specs.
// Only edges between files already in fileMap (internal imports) are recorded,
// since to_file_id is NOT NULL in the schema.
func Build(repositoryID string, fileMap map[string]*cfile.File, pkgs []*packages.Package) []*ImportEdge {
	var edges []*ImportEdge

	for _, pkg := range pkgs {
		for _, f := range pkg.Syntax {
			fromFilePath := pkg.Fset.File(f.Pos()).Name()
			fromFile, ok := fileMap[fromFilePath]
			if !ok {
				continue
			}

			for _, imp := range f.Imports {
				importPath := strings.Trim(imp.Path.Value, `"`)

				importedPkg, found := pkg.Imports[importPath]
				if !found || importedPkg == nil {
					continue
				}

				// Find a file from the imported package that exists in our fileMap.
				var toFileID string
				for _, goFile := range importedPkg.GoFiles {
					if tf, ok := fileMap[goFile]; ok {
						toFileID = tf.ID
						break
					}
				}
				if toFileID == "" {
					continue // external package or not indexed
				}

				var alias *string
				if imp.Name != nil {
					a := imp.Name.Name
					alias = &a
				}

				id, err := generateID()
				if err != nil {
					continue
				}

				now := time.Now().UTC()
				edges = append(edges, &ImportEdge{
					ID:           id,
					RepositoryID: repositoryID,
					FromFileID:   fromFile.ID,
					ToFileID:     toFileID,
					ImportPath:   importPath,
					IsExternal:   false,
					Alias:        alias,
					CreatedAt:    now,
					UpdatedAt:    now,
				})
			}
		}
	}

	return edges
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
