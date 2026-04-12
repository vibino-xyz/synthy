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

type SummarySubscriberAdapter struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

// pub is required to ensure the exchange and queues are declared before this
// subscriber starts consuming.
func NewSummarySubscriber(conn *amqp.Connection, _ embedding.EmbeddingPublisher) (embedding.SummarySubscriber, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	if err := ch.Qos(1, 0, false); err != nil {
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	return &SummarySubscriberAdapter{
		conn: conn,
		ch:   ch,
	}, nil
}

func (s *SummarySubscriberAdapter) Subscribe(ctx context.Context) (<-chan *embedding.ProcessSummaryMessage, error) {
	slog.InfoContext(ctx, "Starting summary event subscriber", "queue", SummaryQueueName)

	deliveries, err := s.ch.Consume(
		SummaryQueueName,
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

	messages := make(chan *embedding.ProcessSummaryMessage)

	go func() {
		defer close(messages)

		for {
			select {
			case <-ctx.Done():
				slog.InfoContext(ctx, "Stopping summary event subscriber", "queue", SummaryQueueName)
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

				select {
				case messages <- &embedding.ProcessSummaryMessage{
					Message:  em,
					Delivery: delivery,
				}:
				case <-ctx.Done():
					delivery.Nack(false, true)
					return
				}
			}
		}
	}()

	return messages, nil
}

func (s *SummarySubscriberAdapter) Close() error {
	if s.ch != nil {
		return s.ch.Close()
	}
	return nil
}
