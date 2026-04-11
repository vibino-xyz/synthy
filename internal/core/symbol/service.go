package symbol

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"time"

	cfile "github.com/vibino-xyz/synthy/internal/core/file"
	"golang.org/x/tools/go/packages"
)

type SymbolService struct {
	repo SymbolRepository
}

func NewSymbolService(repo SymbolRepository) *SymbolService {
	return &SymbolService{repo: repo}
}

// ExtractAndSave extracts all symbols from pkgs, persists them, and returns the saved symbols.
func (s *SymbolService) ExtractAndSave(ctx context.Context, repositoryID string, fileMap map[string]*cfile.File, pkgs []*packages.Package) ([]*Symbol, error) {
	symbols := Extract(repositoryID, fileMap, pkgs)
	if len(symbols) == 0 {
		return symbols, nil
	}
	if err := s.repo.InsertSymbols(ctx, symbols); err != nil {
		return nil, fmt.Errorf("insert symbols: %w", err)
	}
	return symbols, nil
}

// ToFqNameMap builds a lookup map from fully-qualified symbol name to Symbol.
func ToFqNameMap(symbols []*Symbol) map[string]*Symbol {
	m := make(map[string]*Symbol, len(symbols))
	for _, s := range symbols {
		m[s.FqName] = s
	}
	return m
}

// Extract derives Symbol records from pkgs, resolving file IDs via fileMap.
func Extract(repositoryID string, fileMap map[string]*cfile.File, pkgs []*packages.Package) []*Symbol {
	var result []*Symbol
	seen := make(map[types.Object]bool)

	for _, pkg := range pkgs {
		for _, f := range pkg.Syntax {
			filePath := pkg.Fset.File(f.Pos()).Name()
			fileRecord, ok := fileMap[filePath]
			if !ok {
				continue
			}

			var currentDeclTok token.Token

			ast.Inspect(f, func(n ast.Node) bool {
				switch decl := n.(type) {
				case *ast.GenDecl:
					currentDeclTok = decl.Tok
					return true

				case *ast.FuncDecl:
					obj, ok := pkg.TypesInfo.Defs[decl.Name]
					if !ok || seen[obj] {
						return true
					}
					seen[obj] = true

					sym := buildFunctionSymbol(pkg.Fset, decl, obj, repositoryID, fileRecord.ID)
					if sym != nil {
						result = append(result, sym)
					}

				case *ast.TypeSpec:
					obj, ok := pkg.TypesInfo.Defs[decl.Name]
					if !ok || seen[obj] {
						return true
					}
					seen[obj] = true

					sym := buildTypeSymbol(pkg.Fset, decl, obj, repositoryID, fileRecord.ID)
					if sym != nil {
						result = append(result, sym)
					}

				case *ast.ValueSpec:
					for _, name := range decl.Names {
						if name.Name == "_" {
							continue
						}
						obj, ok := pkg.TypesInfo.Defs[name]
						if !ok || seen[obj] {
							continue
						}
						seen[obj] = true

						sym := buildValueSymbol(pkg.Fset, decl, name, obj, currentDeclTok, repositoryID, fileRecord.ID)
						if sym != nil {
							result = append(result, sym)
						}
					}
				}
				return true
			})
		}
	}

	return result
}

func buildFunctionSymbol(fset *token.FileSet, decl *ast.FuncDecl, obj types.Object, repositoryID, fileID string) *Symbol {
	fn, ok := obj.(*types.Func)
	if !ok {
		return nil
	}

	startPos := fset.Position(decl.Pos())
	endPos := fset.Position(decl.End())

	var receiverType *ReceiverType
	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		rt := ReceiverTypeValue
		if _, ok := decl.Recv.List[0].Type.(*ast.StarExpr); ok {
			rt = ReceiverTypePointer
		}
		receiverType = &rt
	}

	visibility := SymbolVisibilityPrivate
	if ast.IsExported(decl.Name.Name) {
		visibility = SymbolVisibilityPublic
	}

	sig := ""
	if s, ok := fn.Type().(*types.Signature); ok {
		sig = types.TypeString(s, nil)
	}

	id, err := generateID()
	if err != nil {
		return nil
	}

	now := time.Now().UTC()
	return &Symbol{
		ID:           id,
		FileID:       fileID,
		RepositoryID: repositoryID,
		FqName:       fn.FullName(),
		Type:         SymbolTypeFunction,
		Language:     "GO",
		Signature:    sig,
		ReceiverType: receiverType,
		Visibility:   visibility,
		StartLine:    startPos.Line,
		EndLine:      endPos.Line,
		StartColumn:  startPos.Column,
		EndColumn:    endPos.Column,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func buildTypeSymbol(fset *token.FileSet, spec *ast.TypeSpec, obj types.Object, repositoryID, fileID string) *Symbol {
	if obj.Pkg() == nil {
		return nil
	}

	startPos := fset.Position(spec.Pos())
	endPos := fset.Position(spec.End())

	symbolType := SymbolTypeType
	if _, ok := spec.Type.(*ast.InterfaceType); ok {
		symbolType = SymbolTypeInterface
	}

	visibility := SymbolVisibilityPrivate
	if ast.IsExported(spec.Name.Name) {
		visibility = SymbolVisibilityPublic
	}

	id, err := generateID()
	if err != nil {
		return nil
	}

	now := time.Now().UTC()
	return &Symbol{
		ID:           id,
		FileID:       fileID,
		RepositoryID: repositoryID,
		FqName:       fmt.Sprintf("%s.%s", obj.Pkg().Path(), obj.Name()),
		Type:         symbolType,
		Language:     "GO",
		Signature:    types.TypeString(obj.Type(), nil),
		Visibility:   visibility,
		StartLine:    startPos.Line,
		EndLine:      endPos.Line,
		StartColumn:  startPos.Column,
		EndColumn:    endPos.Column,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func buildValueSymbol(fset *token.FileSet, spec *ast.ValueSpec, name *ast.Ident, obj types.Object, tok token.Token, repositoryID, fileID string) *Symbol {
	if obj.Pkg() == nil {
		return nil
	}

	startPos := fset.Position(spec.Pos())
	endPos := fset.Position(spec.End())

	symbolType := SymbolTypeVariable
	if tok == token.CONST {
		symbolType = SymbolTypeConstant
	}

	visibility := SymbolVisibilityPrivate
	if ast.IsExported(name.Name) {
		visibility = SymbolVisibilityPublic
	}

	id, err := generateID()
	if err != nil {
		return nil
	}

	now := time.Now().UTC()
	return &Symbol{
		ID:           id,
		FileID:       fileID,
		RepositoryID: repositoryID,
		FqName:       fmt.Sprintf("%s.%s", obj.Pkg().Path(), obj.Name()),
		Type:         symbolType,
		Language:     "GO",
		Signature:    types.TypeString(obj.Type(), nil),
		Visibility:   visibility,
		StartLine:    startPos.Line,
		EndLine:      endPos.Line,
		StartColumn:  startPos.Column,
		EndColumn:    endPos.Column,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
