package mq

import (
	"context"
	"log/slog"

	"github.com/vibino-xyz/synthy/internal/app/events"
)

type RepositoryEventController struct {
	repositoryEventSubscriber events.RepositoryEventSubscriber
}

func NewRepositoryEventController(repositoryEventSubscriber events.RepositoryEventSubscriber) *RepositoryEventController {
	return &RepositoryEventController{
		repositoryEventSubscriber: repositoryEventSubscriber,
	}
}

func (c *RepositoryEventController) Start(ctx context.Context) error {
	messages, err := c.repositoryEventSubscriber.Subscribe(ctx)
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
	return c.repositoryEventSubscriber.Close()
}
