package mq

import (
	"context"
	"log/slog"

	"github.com/vibino-xyz/synthy/internal/core/embedding"
)

type SummaryEventController struct {
	summarySubscriber embedding.SummarySubscriber
}

func NewSummaryEventController(summarySubscriber embedding.SummarySubscriber) *SummaryEventController {
	return &SummaryEventController{
		summarySubscriber: summarySubscriber,
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
