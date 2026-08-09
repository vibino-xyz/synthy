package mq

import (
	"context"
	"log/slog"

	"github.com/vibino-xyz/synthy/internal/core/chunker"
	"github.com/vibino-xyz/synthy/internal/core/embedding"
	"github.com/vibino-xyz/synthy/internal/core/llm"
)

type EmbeddingEventController struct {
	embeddingSubscriber embedding.EmbeddingSubscriber
	embeddingClient     llm.EmbeddingClient
	chunkRepository     chunker.ChunkRepository
	idxConnection       embedding.PineconeRepository
}

func NewEmbeddingEventController(embeddingSubscriber embedding.EmbeddingSubscriber, embeddingClient llm.EmbeddingClient, chunkRepository chunker.ChunkRepository, idxConnection embedding.PineconeRepository) *EmbeddingEventController {
	return &EmbeddingEventController{
		embeddingSubscriber: embeddingSubscriber,
		embeddingClient:     embeddingClient,
		chunkRepository:     chunkRepository,
		idxConnection:       idxConnection,
	}
}

func (c *EmbeddingEventController) Start(ctx context.Context) error {
	messages, err := c.embeddingSubscriber.Subscribe(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to subscribe to embedding events", "error", err)
		return err
	}

	for msg := range messages {
		slog.InfoContext(ctx, "Received embedding event message", "event", msg.Message)

		chunk, err := c.chunkRepository.GetChunkByID(ctx, msg.Message.ChunkID)
		if err != nil {
			fail(ctx, msg, "failed to get chunk by ID", err, "chunk_id", msg.Message.ChunkID)
			continue
		}

		resp, err := c.embeddingClient.GenerateEmbeddings(ctx, chunk.Content, llm.InputTypeDocument)
		if err != nil {
			fail(ctx, msg, "failed to generate embeddings", err, "chunk_id", chunk.ID)
			continue
		}

		slog.InfoContext(ctx, "Generated embeddings for chunk", "chunk_id", chunk.ID, "embedding_length", len(resp))

		id, err := c.idxConnection.UpsertEmbeddings(ctx, resp, map[string]any{
			"chunk_id":      chunk.ID,
			"repository_id": chunk.RepositoryID,
			"file_id":       chunk.FileID,
			"symbol_id":     chunk.SymbolID,
			"chunk_type":    string(chunk.Type),
			"language":      string(chunk.Language),
			"start_line":    chunk.StartLine,
			"end_line":      chunk.EndLine,
		}, embedding.DefaultNamespace)

		if err != nil {
			fail(ctx, msg, "failed to upsert embeddings", err, "chunk_id", chunk.ID)
			continue
		}
		slog.InfoContext(ctx, "Upserted embeddings to Pinecone", "chunk_id", chunk.ID, "vector_id", id)

		if err := c.chunkRepository.UpdateChunk(ctx, chunk.ID, chunker.UpdateChunkRequest{EmbeddingID: &id}); err != nil {
			// The vector is already in Pinecone but the chunk does not point at
			// it; drop it so the retry does not leave an orphan behind.
			if delErr := c.idxConnection.DeleteVectors(ctx, []string{id}, embedding.DefaultNamespace); delErr != nil {
				slog.ErrorContext(ctx, "failed to roll back orphaned vector",
					"chunk_id", chunk.ID, "vector_id", id, "error", delErr)
			}
			fail(ctx, msg, "failed to update chunk embedding ID", err, "chunk_id", chunk.ID)
			continue
		}
		slog.InfoContext(ctx, "Updated chunk embedding ID in database", "chunk_id", chunk.ID, "embedding_id", id)

		if ackErr := msg.Ack(); ackErr != nil {
			slog.ErrorContext(ctx, "failed to ack embedding event", "error", ackErr)
		} else {
			slog.InfoContext(ctx, "Successfully processed embedding event message")
		}
	}
	return nil
}

func (c *EmbeddingEventController) Stop() error {
	return c.embeddingSubscriber.Close()
}
