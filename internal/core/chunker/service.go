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
	"strings"
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
	var buildErr error

	add := func(n ast.Node, chunkType ChunkType) {
		if buildErr != nil {
			return
		}
		chunk, err := buildChunk(fset, n, chunkType)
		if err != nil {
			buildErr = err
			return
		}
		chunks = append(chunks, chunk)
	}

	ast.Inspect(node, func(n ast.Node) bool {
		switch decl := n.(type) {
		case *ast.FuncDecl:
			add(decl, ChunkTypeFunction)
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				if s, ok := spec.(*ast.TypeSpec); ok {
					switch s.Type.(type) {
					case *ast.StructType, *ast.InterfaceType:
						add(s, ChunkTypeType)
					}
				}
			}
		}
		return buildErr == nil
	})
	if buildErr != nil {
		return nil, fmt.Errorf("build chunks for %s: %w", filePath, buildErr)
	}

	if len(chunks) == 0 {
		chunk, err := buildChunk(fset, node, ChunkTypeOther)
		if err != nil {
			return nil, fmt.Errorf("build file chunk for %s: %w", filePath, err)
		}
		chunks = append(chunks, chunk)
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
	var unmatched, blank int
	for _, chunk := range rawChunks {
		// A blank chunk embeds to nothing and is rejected by the embedding API,
		// so never persist one — it would poison the queue for every retry.
		if strings.TrimSpace(chunk.Content) == "" {
			blank++
			continue
		}

		sym := matchSymbol(fileSymbols, chunk.StartLine, chunk.EndLine)
		if sym == nil {
			unmatched++ // symbol_id is NOT NULL; skip unmatched chunks
			continue
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

	// Dropped chunks are silent data loss otherwise: the file indexes "fine"
	// while parts of it are simply missing from retrieval.
	if unmatched > 0 || blank > 0 {
		slog.WarnContext(ctx, "skipped chunks",
			"file", filePath, "unmatched_symbol", unmatched, "blank_content", blank,
			"kept", len(chunks), "total", len(rawChunks))
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

func buildChunk(fset *token.FileSet, node ast.Node, chunkType ChunkType) (*CodeChunk, error) {
	content, err := extractSource(fset, node)
	if err != nil {
		return nil, err
	}

	return &CodeChunk{
		Content:     content,
		ContentHash: sha256hex(content),
		Type:        chunkType,
		StartLine:   fset.Position(node.Pos()).Line,
		EndLine:     fset.Position(node.End()).Line,
	}, nil
}

// extractSource returns the exact source text of node. It relies on the byte
// offsets go/token already computes: node.End() legitimately points one past
// the final byte when node is the last declaration in a file, so the bounds
// check below accepts endOffset == len(content).
func extractSource(fset *token.FileSet, node ast.Node) (string, error) {
	if node == nil {
		return "", fmt.Errorf("extract source: nil node")
	}

	start := fset.Position(node.Pos())
	end := fset.Position(node.End())

	content, err := os.ReadFile(start.Filename)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", start.Filename, err)
	}

	if start.Offset < 0 || end.Offset > len(content) || start.Offset > end.Offset {
		return "", fmt.Errorf("extract source from %s: offsets [%d,%d) out of bounds for %d bytes",
			start.Filename, start.Offset, end.Offset, len(content))
	}

	return string(content[start.Offset:end.Offset]), nil
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
