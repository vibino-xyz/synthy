package embedding

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SummarySubscriber interface {
	Subscribe(ctx context.Context) (<-chan *ProcessSummaryMessage, error)
	Close() error
}

type ProcessSummaryMessage struct {
	Message  *EmbeddingProcessMessage
	Delivery amqp.Delivery
}

func (m *ProcessSummaryMessage) Ack() error {
	return m.Delivery.Ack(false)
}

func (m *ProcessSummaryMessage) Nack(requeue bool) error {
	return m.Delivery.Nack(false, requeue)
}
