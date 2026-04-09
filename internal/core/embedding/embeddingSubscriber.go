package embedding

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type EmbeddingSubscriber interface {
	Subscribe(ctx context.Context) (<-chan *ProcessEmbeddingMessage, error)
	Close() error
}

type ProcessEmbeddingMessage struct {
	Message  *EmbeddingProcessMessage
	Delivery amqp.Delivery
}

func (m *ProcessEmbeddingMessage) Ack() error {
	return m.Delivery.Ack(false)
}

func (m *ProcessEmbeddingMessage) Nack(requeue bool) error {
	return m.Delivery.Nack(false, requeue)
}
