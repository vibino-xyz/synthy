package repository

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type IngestionSubscriber interface {
	Subscribe(ctx context.Context) (<-chan *ProcessIngestionMessage, error)
	Close() error
}

type ProcessIngestionMessage struct {
	Message  *IngestionMessage
	Delivery amqp.Delivery
}

func (m *ProcessIngestionMessage) Ack() error {
	return m.Delivery.Ack(false)
}

func (m *ProcessIngestionMessage) Nack(requeue bool) error {
	return m.Delivery.Nack(false, requeue)
}
