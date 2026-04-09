package rabbimq

import (
	"context"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/vibino-xyz/synthy/internal/core/embedding"
	"google.golang.org/protobuf/proto"
)

type embeddingEventPublisher struct {
	ch       *amqp.Channel
	exchange string
}

const (
	EmbeddingQueueName   = "embedding_event_queue"
	EmbeddingExchange    = "embedding_event_exchange"
	EmbeddingDLXExchange = "embedding_event_dlx_exchange"
)

func NewEmbeddingEventPublisher(conn *amqp.Connection) (embedding.EmbeddingPublisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	err = ch.ExchangeDeclare(
		EmbeddingExchange,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare main exchange: %w", err)
	}

	err = ch.ExchangeDeclare(
		EmbeddingDLXExchange,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare DLX exchange: %w", err)
	}

	failureQueue, err := ch.QueueDeclare(
		EmbeddingQueueName+"_error",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare failure queue: %w", err)
	}

	err = ch.QueueBind(
		failureQueue.Name,
		"failed",
		EmbeddingDLXExchange,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to bind failure queue: %w", err)
	}

	args := amqp.Table{
		"x-dead-letter-exchange":    EmbeddingDLXExchange,
		"x-dead-letter-routing-key": "failed",
	}
	q, err := ch.QueueDeclare(
		EmbeddingQueueName,
		true,
		false,
		false,
		false,
		args,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	err = ch.QueueBind(
		q.Name,
		"",
		EmbeddingExchange,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to bind queue to exchange: %w", err)
	}

	return &embeddingEventPublisher{
		ch:       ch,
		exchange: EmbeddingExchange,
	}, nil
}

func (e *embeddingEventPublisher) PublishEmbedding(ctx context.Context, req embedding.EmbeddingPublisherRequest) error {
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
