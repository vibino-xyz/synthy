package mq

import (
	"context"
	"log/slog"

	"github.com/vibino-xyz/synthy/internal/core/repository"
)

type RepositoryEventController struct {
	ingestionEventSubscriber repository.IngestionSubscriber
}

func NewRepositoryEventController(ingestionEventSubscriber repository.IngestionSubscriber) *RepositoryEventController {
	return &RepositoryEventController{
		ingestionEventSubscriber: ingestionEventSubscriber,
	}
}

func (c *RepositoryEventController) Start(ctx context.Context) error {
	messages, err := c.ingestionEventSubscriber.Subscribe(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to subscribe to repository events", "error", err)
		return err
	}

	for msg := range messages {
		slog.InfoContext(ctx, "Received repository event message", "event", msg.Message)
		if ackErr := msg.Ack(); ackErr != nil {
			slog.ErrorContext(ctx, "failed to ack repository event", "error", ackErr)
		} else {
			slog.InfoContext(ctx, "Successfully processed repository event message")
		}
	}
	return nil
}

func (c *RepositoryEventController) Stop() error {
	return c.ingestionEventSubscriber.Close()
}
