package rabbimq

import (
	"context"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
	repositoryv1 "github.com/vibino-xyz/protos/contracts/build/go/repository/v1"
	"github.com/vibino-xyz/synthy/internal/app/events"
	"google.golang.org/protobuf/proto"
)

const EventsQueueName = "repository_event_queue"

type RepositoryEventSubscriberAdapter struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewRepositoryEventSubscriber(conn *amqp.Connection) (events.RepositoryEventSubscriber, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	err = ch.Qos(1, 0, false)
	if err != nil {
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	return &RepositoryEventSubscriberAdapter{
		conn: conn,
		ch:   ch,
	}, nil
}

func (r *RepositoryEventSubscriberAdapter) Subscribe(ctx context.Context) (<-chan *events.ProcessRepositoryEventMessage, error) {
	slog.InfoContext(ctx, "Starting repository event subscriber", "queue", EventsQueueName)

	deliveries, err := r.ch.Consume(
		EventsQueueName,
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

	messages := make(chan *events.ProcessRepositoryEventMessage)

	go func() {
		defer close(messages)

		for {
			select {
			case <-ctx.Done():
				slog.InfoContext(ctx, "Stopping event subscriber", "queue", EventsQueueName)
				return
			case delivery, ok := <-deliveries:
				if !ok {
					slog.WarnContext(ctx, "RabbitMQ delivery channel closed")
					return
				}

				var msg repositoryv1.RepositoryEventMessage
				if err := proto.Unmarshal(delivery.Body, &msg); err != nil {
					slog.ErrorContext(ctx, "failed to unmarshal proto message", "error", err, "delivery_tag", delivery.DeliveryTag)
					delivery.Nack(false, false)
					continue
				}

				repositoryEventMessage := &events.ProcessRepositoryEventMessage{
					Message:  events.FromProtos(&msg),
					Delivery: delivery,
				}

				select {
				case messages <- repositoryEventMessage:
				case <-ctx.Done():
					delivery.Nack(false, true)
					return

				}

			}
		}
	}()

	return messages, nil
}

func (r *RepositoryEventSubscriberAdapter) Close() error {
	if r.ch != nil {
		return r.ch.Close()
	}
	return nil
}
