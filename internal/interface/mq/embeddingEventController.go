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
			slog.ErrorContext(ctx, "failed to get chunk by ID", "error", err)
			continue
		}

		resp, err := c.embeddingClient.GenerateEmbeddings(ctx, chunk.Content)
		if err != nil {
			slog.ErrorContext(ctx, "failed to generate embeddings", "error", err)
			continue
		}

		slog.InfoContext(ctx, "Generated embeddings for chunk", "chunk_id", chunk.ID, "embedding_length", len(resp), "embeddings", resp)

		id, err := c.idxConnection.UpsertEmbeddings(ctx, resp, map[string]any{
			"chunk_id":      chunk.ID,
			"repository_id": chunk.RepositoryID,
			"file_id":       chunk.FileID,
			"symbol_id":     chunk.SymbolID,
			"chunk_type":    string(chunk.Type),
			"language":      string(chunk.Language),
			"start_line":    chunk.StartLine,
			"end_line":      chunk.EndLine,
		}, "__default__")
		if err != nil {
			slog.ErrorContext(ctx, "failed to upsert embeddings", "error", err)
			continue
		}
		slog.InfoContext(ctx, "Upserted embeddings to Pinecone", "chunk_id", chunk.ID, "vector_id", id)

		if err := c.chunkRepository.UpdateChunk(ctx, chunk.ID, chunker.UpdateChunkRequest{EmbeddingID: &id}); err != nil {
			slog.ErrorContext(ctx, "failed to update chunk embedding ID", "error", err)
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
