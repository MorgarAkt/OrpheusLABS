package services

import (
	"fmt"
	"time"

	"github.com/streadway/amqp"
)

type RabbitMQClient struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	replyTo string
}

func NewRabbitMQClient(amqpURL string) (*RabbitMQClient, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	replyQueue, err := ch.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare reply queue: %w", err)
	}

	return &RabbitMQClient{
		conn:    conn,
		channel: ch,
		replyTo: replyQueue.Name,
	}, nil
}

func (c *RabbitMQClient) Call(queue string, request []byte, timeout time.Duration) ([]byte, error) {
	correlationID := fmt.Sprintf("%d", time.Now().UnixNano())

	err := c.channel.Publish(
		"",
		queue,
		false,
		false,
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: correlationID,
			ReplyTo:       c.replyTo,
			Body:          request,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to publish message: %w", err)
	}

	msgs, err := c.channel.Consume(c.replyTo, "", true, true, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to consume response: %w", err)
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case msg := <-msgs:
			if msg.CorrelationId == correlationID {
				return msg.Body, nil
			}
		case <-timer.C:
			return nil, fmt.Errorf("timeout waiting for response")
		}
	}
}

func (c *RabbitMQClient) Close() {
	c.channel.Close()
	c.conn.Close()
}
