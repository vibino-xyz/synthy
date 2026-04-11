package calls

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/types"
	"time"

	cfile "github.com/vibino-xyz/synthy/internal/core/file"
	csymbol "github.com/vibino-xyz/synthy/internal/core/symbol"
	"golang.org/x/tools/go/packages"
)

type CallEdgeService struct {
	repo CallEdgeRepository
}

func NewCallEdgeService(repo CallEdgeRepository) *CallEdgeService {
	return &CallEdgeService{repo: repo}
}

// BuildAndSave extracts call edges from pkgs, persists them, and returns the saved edges.
func (s *CallEdgeService) BuildAndSave(ctx context.Context, repositoryID string, fileMap map[string]*cfile.File, symbolMap map[string]*csymbol.Symbol, pkgs []*packages.Package) ([]*CallEdge, error) {
	edges := Build(repositoryID, fileMap, symbolMap, pkgs)
	if len(edges) == 0 {
		return edges, nil
	}
	if err := s.repo.InsertCallEdges(ctx, edges); err != nil {
		return nil, fmt.Errorf("insert call edges: %w", err)
	}
	return edges, nil
}

// Build extracts call edges from the given packages.
// Only edges where both caller and callee exist in symbolMap are recorded.
func Build(repositoryID string, fileMap map[string]*cfile.File, symbolMap map[string]*csymbol.Symbol, pkgs []*packages.Package) []*CallEdge {
	var edges []*CallEdge

	for _, pkg := range pkgs {
		for _, f := range pkg.Syntax {
			filePath := pkg.Fset.File(f.Pos()).Name()
			fileRecord, ok := fileMap[filePath]
			if !ok {
				continue
			}

			var callerSymbol *csymbol.Symbol

			ast.Inspect(f, func(n ast.Node) bool {
				switch node := n.(type) {
				case *ast.FuncDecl:
					if obj, ok := pkg.TypesInfo.Defs[node.Name]; ok {
						if fn, ok := obj.(*types.Func); ok {
							callerSymbol = symbolMap[fn.FullName()]
						}
					}

				case *ast.CallExpr:
					if callerSymbol == nil {
						return true
					}

					var callee *types.Func

					switch fun := node.Fun.(type) {
					case *ast.Ident:
						if obj, ok := pkg.TypesInfo.Uses[fun]; ok {
							if fn, ok := obj.(*types.Func); ok {
								callee = fn
							}
						}
					case *ast.SelectorExpr:
						if sel, ok := pkg.TypesInfo.Selections[fun]; ok {
							if fn, ok := sel.Obj().(*types.Func); ok {
								callee = fn
							}
						} else if obj, ok := pkg.TypesInfo.Uses[fun.Sel]; ok {
							if fn, ok := obj.(*types.Func); ok {
								callee = fn
							}
						}
					}

					if callee == nil {
						return true
					}

					calleeSymbol, ok := symbolMap[callee.FullName()]
					if !ok {
						return true
					}

					id, err := generateID()
					if err != nil {
						return true
					}

					now := time.Now().UTC()
					edges = append(edges, &CallEdge{
						ID:             id,
						RepositoryID:   repositoryID,
						CallerSymbolID: callerSymbol.ID,
						CalleeSymbolID: calleeSymbol.ID,
						FileID:         fileRecord.ID,
						CreatedAt:      now,
						UpdatedAt:      now,
					})
				}
				return true
			})
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
