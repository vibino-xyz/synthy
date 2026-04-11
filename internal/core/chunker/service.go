package chunker

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	csymbol "github.com/vibino-xyz/synthy/internal/core/symbol"
)

// Build parses filePath and returns code chunks for every function, struct, and
// interface declaration it contains. The returned chunks have Content, ContentHash,
// Type, StartLine, and EndLine populated; the remaining DB fields are filled in by
// ChunkService.ProcessFile.
func Build(filePath string) ([]*CodeChunk, error) {
	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var chunks []*CodeChunk

	ast.Inspect(node, func(n ast.Node) bool {
		switch decl := n.(type) {
		case *ast.FuncDecl:
			chunks = append(chunks, buildFunctionChunk(fset, decl))
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					switch t := s.Type.(type) {
					case *ast.StructType:
						chunks = append(chunks, buildStructChunk(fset, s, t))
					case *ast.InterfaceType:
						chunks = append(chunks, buildInterfaceChunk(fset, s, t))
					}
				}
			}
		}
		return true
	})

	if len(chunks) == 0 {
		chunks = append(chunks, buildFileChunk(fset, node))
	}

	return chunks, nil
}

type ChunkService struct {
	repo ChunkRepository
}

func NewChunkService(repo ChunkRepository) *ChunkService {
	return &ChunkService{repo: repo}
}

// ProcessFile builds chunks for filePath, matches each chunk to a symbol by line
// overlap, fills in the DB fields, persists them, and returns the saved chunks.
func (s *ChunkService) ProcessFile(ctx context.Context, repositoryID, fileID, filePath string, fileSymbols []*csymbol.Symbol) ([]*CodeChunk, error) {
	rawChunks, err := Build(filePath)
	if err != nil {
		return nil, fmt.Errorf("build chunks for %s: %w", filePath, err)
	}

	lang := LanguageOther
	if filepath.Ext(filePath) == ".go" {
		lang = LanguageGo
	}

	var chunks []*CodeChunk
	for _, chunk := range rawChunks {
		sym := matchSymbol(fileSymbols, chunk.StartLine, chunk.EndLine)
		if sym == nil {
			continue // symbol_id is NOT NULL; skip unmatched chunks
		}

		id, err := generateID()
		if err != nil {
			return nil, err
		}

		now := time.Now().UTC()
		chunk.ID = id
		chunk.RepositoryID = repositoryID
		chunk.FileID = fileID
		chunk.SymbolID = sym.ID
		chunk.Language = lang
		chunk.CreatedAt = now
		chunk.UpdatedAt = now

		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		return chunks, nil
	}

	if err := s.repo.InsertChunks(ctx, chunks); err != nil {
		return nil, fmt.Errorf("insert chunks: %w", err)
	}

	return chunks, nil
}

// matchSymbol returns the symbol whose line range fully contains [startLine, endLine].
func matchSymbol(symbols []*csymbol.Symbol, startLine, endLine int) *csymbol.Symbol {
	for _, sym := range symbols {
		if sym.StartLine <= startLine && sym.EndLine >= endLine {
			return sym
		}
	}
	return nil
}

func buildFunctionChunk(fset *token.FileSet, fn *ast.FuncDecl) *CodeChunk {
	startLine := fset.Position(fn.Pos()).Line
	endLine := fset.Position(fn.End()).Line
	content := extractSource(fset, fn)

	return &CodeChunk{
		Content:     content,
		ContentHash: sha256hex(content),
		Type:        ChunkTypeFunction,
		StartLine:   startLine,
		EndLine:     endLine,
	}
}

func buildStructChunk(fset *token.FileSet, ts *ast.TypeSpec, _ *ast.StructType) *CodeChunk {
	startLine := fset.Position(ts.Pos()).Line
	endLine := fset.Position(ts.End()).Line
	content := extractSource(fset, ts)

	return &CodeChunk{
		Content:     content,
		ContentHash: sha256hex(content),
		Type:        ChunkTypeType,
		StartLine:   startLine,
		EndLine:     endLine,
	}
}

func buildInterfaceChunk(fset *token.FileSet, ts *ast.TypeSpec, _ *ast.InterfaceType) *CodeChunk {
	startLine := fset.Position(ts.Pos()).Line
	endLine := fset.Position(ts.End()).Line
	content := extractSource(fset, ts)

	return &CodeChunk{
		Content:     content,
		ContentHash: sha256hex(content),
		Type:        ChunkTypeType,
		StartLine:   startLine,
		EndLine:     endLine,
	}
}

func buildFileChunk(fset *token.FileSet, node ast.Node) *CodeChunk {
	startLine := fset.Position(node.Pos()).Line
	endLine := fset.Position(node.End()).Line
	content := extractSource(fset, node)

	return &CodeChunk{
		Content:     content,
		ContentHash: sha256hex(content),
		Type:        ChunkTypeOther,
		StartLine:   startLine,
		EndLine:     endLine,
	}
}

func extractSource(fset *token.FileSet, node ast.Node) string {
	if node == nil {
		return ""
	}

	start := fset.Position(node.Pos())
	end := fset.Position(node.End())

	content, err := os.ReadFile(start.Filename)
	if err != nil {
		slog.Error("failed to read file content", "error", err)
		return ""
	}

	startOffset := offsetFromPosition(content, start)
	endOffset := offsetFromPosition(content, end)

	if startOffset == -1 || endOffset == -1 || startOffset > endOffset {
		slog.Error("invalid offsets for source extraction", "startOffset", startOffset, "endOffset", endOffset)
		return ""
	}

	return string(content[startOffset:endOffset])
}

func offsetFromPosition(content []byte, pos token.Position) int {
	line := 1
	col := 1

	for i, b := range content {
		if line == pos.Line && col == pos.Column {
			return i
		}
		if b == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return -1
}

func sha256hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
