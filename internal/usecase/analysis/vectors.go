package analysis

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vibino-xyz/synthy/internal/core/chunker"
	"github.com/vibino-xyz/synthy/internal/core/embedding"
)

// pineconeDeleteBatch is the number of ids sent per delete call. Pinecone caps
// delete-by-id at 1000.
const pineconeDeleteBatch = 1000

// deleteRepositoryVectors removes every vector belonging to a repository.
func (p *AnalysisPipeline) deleteRepositoryVectors(ctx context.Context, repositoryID string) error {
	chunks, err := p.chunkRepo.GetChunksByRepositoryID(ctx, repositoryID)
	if err != nil {
		return fmt.Errorf("list chunks for vector cleanup: %w", err)
	}
	return p.deleteVectorsForChunks(ctx, chunks, "repository_id", repositoryID)
}

// deleteFileVectors removes the vectors of one file's chunks.
func (p *AnalysisPipeline) deleteFileVectors(ctx context.Context, fileID string) error {
	chunks, err := p.chunkRepo.GetChunksByFileID(ctx, fileID)
	if err != nil {
		return fmt.Errorf("list chunks for vector cleanup: %w", err)
	}
	return p.deleteVectorsForChunks(ctx, chunks, "file_id", fileID)
}

// deleteVectorsForChunks deletes the Pinecone vectors the given chunks point at.
//
// Chunks with no embedding id were never embedded — queued but not yet processed,
// or failed — and have nothing to delete.
func (p *AnalysisPipeline) deleteVectorsForChunks(
	ctx context.Context, chunks []*chunker.CodeChunk, scopeKey, scopeValue string,
) error {
	if p.vectors == nil {
		return nil // no vector store wired (tests)
	}

	ids := make([]string, 0, len(chunks))
	for _, c := range chunks {
		if c.EmbeddingID != nil && *c.EmbeddingID != "" {
			ids = append(ids, *c.EmbeddingID)
		}
	}
	if len(ids) == 0 {
		return nil
	}

	for start := 0; start < len(ids); start += pineconeDeleteBatch {
		end := min(start+pineconeDeleteBatch, len(ids))
		if err := p.vectors.DeleteVectors(ctx, ids[start:end], embedding.DefaultNamespace); err != nil {
			return fmt.Errorf("delete vectors (%s=%s): %w", scopeKey, scopeValue, err)
		}
	}

	slog.InfoContext(ctx, "deleted embeddings", scopeKey, scopeValue, "vectors", len(ids))
	return nil
}
