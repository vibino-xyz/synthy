package retrieval

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/vibino-xyz/synthy/internal/core/calls"
	"github.com/vibino-xyz/synthy/internal/core/chunker"
	"github.com/vibino-xyz/synthy/internal/core/embedding"
	"github.com/vibino-xyz/synthy/internal/core/imports"
	"github.com/vibino-xyz/synthy/internal/core/llm"
)

// defaultTopK is the number of nearest chunks retrieved when the caller does
// not specify a limit.
const defaultTopK = 7

// retrievalNamespace is the pinecone namespace queried for similar vectors. It
// MUST match the namespace the ingestion pipeline upserts into
// (see interface/mq/embeddingEventController.go). Per-tenant namespacing is a
// future change that has to be applied on both the ingest and retrieval sides
// together.
const retrievalNamespace = "__default__"

type RetrievalPipeline struct {
	EmbeddingClient      llm.EmbeddingClient
	ChunkRepository      chunker.ChunkRepository
	CallEdgeRepository   calls.CallEdgeRepository
	ImportEdgeRepository imports.ImportEdgeRepository
	idxConnection        embedding.PineconeRepository
}

func NewRetrievalPipeline(
	embeddingClient llm.EmbeddingClient,
	chunkRepository chunker.ChunkRepository,
	callEdgeRepository calls.CallEdgeRepository,
	importEdgeRepository imports.ImportEdgeRepository,
	idxConnection embedding.PineconeRepository,
) *RetrievalPipeline {
	return &RetrievalPipeline{
		EmbeddingClient:      embeddingClient,
		ChunkRepository:      chunkRepository,
		CallEdgeRepository:   callEdgeRepository,
		ImportEdgeRepository: importEdgeRepository,
		idxConnection:        idxConnection,
	}
}

// RetrievalContext holds the assembled context ready to be passed to an LLM.
type RetrievalContext struct {
	// Formatted is the full context string built from retrieved chunks, call
	// edges, and import edges.
	Formatted string
	// Chunks are the raw chunks that back Formatted, exposed for citations and
	// cross-source ranking by the caller.
	Chunks []*chunker.CodeChunk
}

// Retrieve embeds the query, finds the nearest code chunks in the vector index
// and assembles them (with their call and import edges) into an LLM-ready
// context. It performs no generation — the caller owns the LLM step. Passing
// topK <= 0 uses defaultTopK.
func (p *RetrievalPipeline) Retrieve(ctx context.Context, query string, topK int) (*RetrievalContext, error) {
	if topK <= 0 {
		topK = defaultTopK
	}

	slog.InfoContext(ctx, "Retrieval query received", "q", query, "topK", topK)
	embeddings, err := p.EmbeddingClient.GenerateEmbeddings(ctx, query, llm.InputTypeQuery)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to generate embeddings", "error", err)
		return nil, err
	}

	similarEmbeddingIDs, err := p.idxConnection.QuerySimilarVectors(ctx, embeddings, topK, retrievalNamespace)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to query similar vectors", "error", err)
		return nil, err
	}

	chunks, err := p.ChunkRepository.GetChunksByEmbeddingIDs(ctx, similarEmbeddingIDs)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get chunks by embedding IDs", "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "Retrieved chunks", "similarEmbeddingIDs", similarEmbeddingIDs, "chunkCount", len(chunks))
	return p.buildContext(ctx, chunks)
}

// buildContext assembles a structured context string from code chunks together
// with their outgoing call edges and the import edges of their source files.
func (p *RetrievalPipeline) buildContext(ctx context.Context, chunks []*chunker.CodeChunk) (*RetrievalContext, error) {
	// Deduplicate symbol IDs and file IDs to avoid redundant lookups.
	seenSymbols := make(map[string]bool)
	seenFiles := make(map[string]bool)

	callEdgesBySymbol := make(map[string][]*calls.CallEdge)
	importEdgesByFile := make(map[string][]*imports.ImportEdge)

	for _, c := range chunks {
		if c.SymbolID != "" && !seenSymbols[c.SymbolID] {
			seenSymbols[c.SymbolID] = true
			edges, err := p.CallEdgeRepository.GetCallEdgesByCallerSymbolID(ctx, c.SymbolID)
			if err != nil {
				return nil, fmt.Errorf("fetch call edges for symbol %s: %w", c.SymbolID, err)
			}
			callEdgesBySymbol[c.SymbolID] = edges
		}

		if c.FileID != "" && !seenFiles[c.FileID] {
			seenFiles[c.FileID] = true
			edges, err := p.ImportEdgeRepository.GetImportEdgesByFromFileID(ctx, c.FileID)
			if err != nil {
				return nil, fmt.Errorf("fetch import edges for file %s: %w", c.FileID, err)
			}
			importEdgesByFile[c.FileID] = edges
		}
	}

	var sb strings.Builder
	sb.WriteString("=== Retrieved Code Context ===\n\n")

	for i, c := range chunks {
		fmt.Fprintf(&sb, "--- Chunk %d | Type: %s | Language: %s | Lines: %d-%d ---\n",
			i+1, c.Type, c.Language, c.StartLine, c.EndLine)

		if c.Summary != nil && *c.Summary != "" {
			fmt.Fprintf(&sb, "Summary: %s\n", *c.Summary)
		}

		sb.WriteString("Content:\n")
		sb.WriteString(c.Content)
		sb.WriteString("\n")

		if edges := callEdgesBySymbol[c.SymbolID]; len(edges) > 0 {
			sb.WriteString("Outgoing Calls:\n")
			for _, e := range edges {
				fmt.Fprintf(&sb, "  -> %s\n", e.CalleeSymbolID)
			}
		}

		if edges := importEdgesByFile[c.FileID]; len(edges) > 0 {
			sb.WriteString("File Imports:\n")
			for _, e := range edges {
				entry := fmt.Sprintf("  - %s", e.ImportPath)
				if e.Alias != nil && *e.Alias != "" {
					entry += fmt.Sprintf(" (alias: %s)", *e.Alias)
				}
				if e.IsExternal {
					entry += " [external]"
				}
				sb.WriteString(entry + "\n")
			}
		}

		sb.WriteString("\n")
	}

	return &RetrievalContext{Formatted: sb.String(), Chunks: chunks}, nil
}
