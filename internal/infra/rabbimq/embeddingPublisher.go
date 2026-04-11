package rabbimq

import (
	"context"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/vibino-xyz/synthy/internal/core/embedding"
	"google.golang.org/protobuf/proto"
)

type embeddingPublisher struct {
	ch       *amqp.Channel
	exchange string
}

const (
	EmbeddingQueueName   = "embedding_queue"
	SummaryQueueName     = "summary_queue"
	EmbeddingExchange    = "embedding_exchange"
	EmbeddingDLXExchange = "embedding_dlx_exchange"
)

func NewEmbeddingPublisher(conn *amqp.Connection) (embedding.EmbeddingPublisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Main fanout exchange — every bound queue receives every message.
	if err := ch.ExchangeDeclare(
		EmbeddingExchange,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return nil, fmt.Errorf("failed to declare main exchange: %w", err)
	}

	// Dead-letter exchange for unprocessable messages from both queues.
	if err := ch.ExchangeDeclare(
		EmbeddingDLXExchange,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return nil, fmt.Errorf("failed to declare DLX exchange: %w", err)
	}

	if err := declareQueueWithDLX(ch, EmbeddingQueueName, EmbeddingDLXExchange); err != nil {
		return nil, fmt.Errorf("failed to set up %s: %w", EmbeddingQueueName, err)
	}

	if err := declareQueueWithDLX(ch, SummaryQueueName, EmbeddingDLXExchange); err != nil {
		return nil, fmt.Errorf("failed to set up %s: %w", SummaryQueueName, err)
	}

	return &embeddingPublisher{
		ch:       ch,
		exchange: EmbeddingExchange,
	}, nil
}

// declareQueueWithDLX declares a durable queue that forwards rejected messages
// to dlxExchange, and also declares the corresponding _error queue.
func declareQueueWithDLX(ch *amqp.Channel, queueName, dlxExchange string) error {
	errorQueue, err := ch.QueueDeclare(
		queueName+"_error",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("declare error queue: %w", err)
	}

	if err := ch.QueueBind(errorQueue.Name, queueName+"_failed", dlxExchange, false, nil); err != nil {
		return fmt.Errorf("bind error queue: %w", err)
	}

	q, err := ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-dead-letter-exchange":    dlxExchange,
			"x-dead-letter-routing-key": queueName + "_failed",
		},
	)
	if err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	// Routing key is ignored by fanout exchanges; use empty string.
	if err := ch.QueueBind(q.Name, "", EmbeddingExchange, false, nil); err != nil {
		return fmt.Errorf("bind queue to exchange: %w", err)
	}

	return nil
}

func (e *embeddingPublisher) PublishEmbedding(ctx context.Context, req embedding.EmbeddingPublisherRequest) error {
	protoMsg, err := req.ToProto()
	if err != nil {
		return fmt.Errorf("failed to convert to proto message: %w", err)
	}

	messageBytes, err := proto.Marshal(protoMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal proto message: %w", err)
	}

	err = e.ch.PublishWithContext(ctx,
		e.exchange,
		"",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/x-protobuf",
			Body:        messageBytes,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish embedding message: %w", err)
	}

	slog.InfoContext(ctx, "Published embedding event", "chunk_id", req.ChunkId, "repository_id", req.RepositoryId)

	return nil
}
