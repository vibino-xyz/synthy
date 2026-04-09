package rabbimq

import (
	"context"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
	contracts "github.com/vibino-xyz/protos/contracts/build"
	"github.com/vibino-xyz/synthy/internal/core/repository"
	"google.golang.org/protobuf/proto"
)

const EventsQueueName = "repository_event_queue"

type IngestionSubscriberAdapter struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewIngestionSubscriber(conn *amqp.Connection) (repository.IngestionSubscriber, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	err = ch.Qos(1, 0, false)
	if err != nil {
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	return &IngestionSubscriberAdapter{
		conn: conn,
		ch:   ch,
	}, nil
}

func (r *IngestionSubscriberAdapter) Subscribe(ctx context.Context) (<-chan *repository.ProcessIngestionMessage, error) {
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

	messages := make(chan *repository.ProcessIngestionMessage)

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

				var msg contracts.RepositoryEventMessage
				if err := proto.Unmarshal(delivery.Body, &msg); err != nil {
					slog.ErrorContext(ctx, "failed to unmarshal proto message", "error", err, "delivery_tag", delivery.DeliveryTag)
					delivery.Nack(false, false)
					continue
				}

				im := new(repository.IngestionMessage)
				_ = im.FromProto(&msg)

				repositoryEventMessage := &repository.ProcessIngestionMessage{
					Message:  im,
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

func (r *IngestionSubscriberAdapter) Close() error {
	if r.ch != nil {
		return r.ch.Close()
	}
	return nil
}
