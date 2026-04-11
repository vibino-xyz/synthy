package rabbimq

import (
	"context"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
	contracts "github.com/vibino-xyz/protos/contracts/build"
	"github.com/vibino-xyz/synthy/internal/core/embedding"
	"google.golang.org/protobuf/proto"
)

type EmbeddingSubscriberAdapter struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

// pub is required to ensure the exchange and queues are declared before this
// subscriber starts consuming.
func NewEmbeddingSubscriber(conn *amqp.Connection, _ embedding.EmbeddingPublisher) (embedding.EmbeddingSubscriber, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	err = ch.Qos(1, 0, false)
	if err != nil {
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	return &EmbeddingSubscriberAdapter{
		conn: conn,
		ch:   ch,
	}, nil
}

func (e *EmbeddingSubscriberAdapter) Subscribe(ctx context.Context) (<-chan *embedding.ProcessEmbeddingMessage, error) {
	slog.InfoContext(ctx, "Starting embedding event subscriber", "queue", EmbeddingQueueName)

	deliveries, err := e.ch.Consume(
		EmbeddingQueueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start consuming: %w", err)
	}

	messages := make(chan *embedding.ProcessEmbeddingMessage)

	go func() {
		defer close(messages)

		for {
			select {
			case <-ctx.Done():
				slog.InfoContext(ctx, "Stopping embedding event subscriber", "queue", EmbeddingQueueName)
				return
			case delivery, ok := <-deliveries:
				if !ok {
					slog.WarnContext(ctx, "RabbitMQ delivery channel closed")
					return
				}

				var msg contracts.EmbeddingEventMessage
				if err := proto.Unmarshal(delivery.Body, &msg); err != nil {
					slog.ErrorContext(ctx, "failed to unmarshal proto message", "error", err, "delivery_tag", delivery.DeliveryTag)
					delivery.Nack(false, false)
					continue
				}

				em := new(embedding.EmbeddingProcessMessage)
				_ = em.FromProto(&msg)

				embeddingEventMessage := &embedding.ProcessEmbeddingMessage{
					Message:  em,
					Delivery: delivery,
				}

				select {
				case messages <- embeddingEventMessage:
				case <-ctx.Done():
					delivery.Nack(false, true)
					return
				}
			}
		}
	}()

	return messages, nil
}

func (e *EmbeddingSubscriberAdapter) Close() error {
	if e.ch != nil {
		return e.ch.Close()
	}
	return nil
}
