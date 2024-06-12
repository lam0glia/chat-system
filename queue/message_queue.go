package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lam0glia/chat-system/domain"

	amqp "github.com/rabbitmq/amqp091-go"
)

type message struct {
	ch *amqp.Channel
}

func (q *message) Send(ctx context.Context, msg *domain.Message) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to encode json: %w", err)
	}

	err = q.ch.PublishWithContext(ctx,
		"",                             // exchange
		q.getQueueName(msg.ReceiverID), // routing key
		false,                          // mandatory
		false,                          // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        b,
		})

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

func (q *message) NewUserQueue(userID int) error {
	_, err := q.ch.QueueDeclare(
		q.getQueueName(userID), // name
		false,                  // durable
		false,                  // delete when unused
		false,                  // exclusive
		false,                  // no-wait
		nil,                    // arguments
	)

	return err
}

func (q *message) getQueueName(receiverID int) string {
	return fmt.Sprintf("msg-%d", receiverID)
}

func NewMessage(
	conn *amqp.Connection,
) (*message, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	return &message{
		ch: ch,
	}, nil
}
