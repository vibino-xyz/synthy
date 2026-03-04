package rabbimq

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

func NewConn() (*amqp.Connection, error) {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return nil, err
	}

	return conn, nil
}
