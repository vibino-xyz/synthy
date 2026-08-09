package mq

import (
	"context"
	"log/slog"

	"github.com/vibino-xyz/synthy/internal/core/chunker"
	"github.com/vibino-xyz/synthy/internal/core/embedding"
	"github.com/vibino-xyz/synthy/internal/core/llm"
)

type SummaryEventController struct {
	summarySubscriber embedding.SummarySubscriber
	chunkRepository   chunker.ChunkRepository
	llmClient         llm.LLMClient
}

func NewSummaryEventController(summarySubscriber embedding.SummarySubscriber, chunkRepository chunker.ChunkRepository, llmClient llm.LLMClient) *SummaryEventController {
	return &SummaryEventController{
		summarySubscriber: summarySubscriber,
		chunkRepository:   chunkRepository,
		llmClient:         llmClient,
	}
}

func (c *SummaryEventController) Start(ctx context.Context) error {
	messages, err := c.summarySubscriber.Subscribe(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to subscribe to summary events", "error", err)
		return err
	}

	for msg := range messages {
		slog.InfoContext(ctx, "Received summary event message", "event", msg.Message)

		chunk, err := c.chunkRepository.GetChunkByID(ctx, msg.Message.ChunkID)
		if err != nil {
			fail(ctx, msg, "failed to get chunk by ID", err, "chunk_id", msg.Message.ChunkID)
			continue
		}

		resp, err := c.llmClient.GenerateSummary(ctx, chunk.Content)
		if err != nil {
			fail(ctx, msg, "failed to generate summary", err, "chunk_id", chunk.ID)
			continue
		}

		slog.InfoContext(ctx, "Generated summary", "chunk_id", chunk.ID)

		if err := c.chunkRepository.UpdateChunk(ctx, chunk.ID, chunker.UpdateChunkRequest{Summary: &resp}); err != nil {
			fail(ctx, msg, "failed to update chunk summary", err, "chunk_id", chunk.ID)
			continue
		}
		slog.InfoContext(ctx, "Updated chunk summary in database", "chunk_id", chunk.ID)

		if ackErr := msg.Ack(); ackErr != nil {
			slog.ErrorContext(ctx, "failed to ack summary event", "error", ackErr)
		} else {
			slog.InfoContext(ctx, "Successfully processed summary event message")
		}
	}
	return nil
}

func (c *SummaryEventController) Stop() error {
	return c.summarySubscriber.Close()
}
