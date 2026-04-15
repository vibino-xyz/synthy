package rabbimq

import (
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

func NewConn() (*amqp.Connection, error) {
	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		return nil, err
	}

	return conn, nil
}
