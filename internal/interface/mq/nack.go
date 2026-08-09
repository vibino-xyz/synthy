package mq

import (
	"context"
	"log/slog"
)

// nacker is the subset of a delivery wrapper the failure handling needs. Every
// subscriber message type (embedding, summary, ingestion) satisfies it.
type nacker interface {
	Nack(requeue bool) error
}

// fail rejects a delivery that could not be processed.
//
// Nothing here requeues. A message that failed once — a chunk that cannot be
// embedded, a repo that cannot be cloned — fails identically on every redelivery,
// and a message left neither acked nor nacked is worse still: it stays unacked
// and RabbitMQ hands it back on the next reconnect, forever. Both queues declare
// a dead-letter exchange, so rejecting without requeue parks the message there
// for inspection instead of looping.
func fail(ctx context.Context, msg nacker, event string, err error, attrs ...any) {
	slog.ErrorContext(ctx, event, append(attrs, "error", err)...)

	if nackErr := msg.Nack(false); nackErr != nil {
		slog.ErrorContext(ctx, "failed to nack message, it will be redelivered",
			"event", event, "error", nackErr)
	}
}
