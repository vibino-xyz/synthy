package rabbimq

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/vibino-xyz/synthy/internal/core/repository"
	"google.golang.org/protobuf/proto"
)

// Repository-event topology. Mirrors wolfy's publisher exactly (same exchange,
// queue and dead-letter args) so both producers and synthy's own consumer agree
// on the declaration — RabbitMQ rejects re-declaring a queue with different
// arguments.
const (
	RepositoryEventExchange    = "repository_event_exchange"
	RepositoryEventDLXExchange = "repository_event_dlx_exchange"
)

type repositoryEventPublisher struct {
	ch       *amqp.Channel
	exchange string
}

func NewRepositoryEventPublisher(conn *amqp.Connection) (repository.RepositoryEventPublisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	if err := ch.ExchangeDeclare(RepositoryEventExchange, "direct", true, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("failed to declare main exchange: %w", err)
	}
	if err := ch.ExchangeDeclare(RepositoryEventDLXExchange, "direct", true, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("failed to declare DLX exchange: %w", err)
	}

	args := amqp.Table{
		"x-dead-letter-exchange":    RepositoryEventDLXExchange,
		"x-dead-letter-routing-key": "failed",
	}
	queue, err := ch.QueueDeclare(EventsQueueName, true, false, false, false, args)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}
	if err := ch.QueueBind(queue.Name, "", RepositoryEventExchange, false, nil); err != nil {
		return nil, fmt.Errorf("failed to bind queue to main exchange: %w", err)
	}

	return &repositoryEventPublisher{ch: ch, exchange: RepositoryEventExchange}, nil
}

func (r *repositoryEventPublisher) Publish(ctx context.Context, message *repository.IngestionMessage) error {
	proto0, err := message.ToProto()
	if err != nil {
		return fmt.Errorf("failed to map message to proto: %w", err)
	}

	body, err := proto.Marshal(proto0)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := r.ch.PublishWithContext(
		ctx,
		r.exchange,
		"",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/x-protobuf",
			Body:        body,
			Headers: amqp.Table{
				"event_type": int32(proto0.EventType),
			},
		},
	); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}
	return nil
}
