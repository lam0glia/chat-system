package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lam0glia/chat-system/domain"

	amqp "github.com/rabbitmq/amqp091-go"
)

const messageExchange = "messages"

type message struct {
	ch *amqp.Channel
}

func (q *message) Publish(ctx context.Context, msg *domain.Message) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to encode json: %w", err)
	}

	err = q.ch.PublishWithContext(ctx,
		messageExchange,             // exchange
		fmt.Sprintf("%d", msg.ToID), // routing key
		false,                       // mandatory
		false,                       // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        b,
		})

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

func (q *message) declareConsumer(userID uint64) (string, error) {
	qDeclared, err := q.ch.QueueDeclare(
		"",    // name
		false, // durable
		false, // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return qDeclared.Name, fmt.Errorf("declare queue: %w", err)
	}

	err = q.ch.QueueBind(
		qDeclared.Name,            // queue name
		fmt.Sprintf("%d", userID), // routing key
		messageExchange,           // exchange
		false,
		nil,
	)
	if err != nil {
		return qDeclared.Name, fmt.Errorf("bind queue to an exchange: %w", err)
	}

	return qDeclared.Name, nil
}

func (q *message) NewConsumer(ctx context.Context, userID uint64) (<-chan amqp.Delivery, error) {
	name, err := q.declareConsumer(userID)
	if err != nil {
		return nil, fmt.Errorf("declare consumer: %w", err)
	}

	consumer := fmt.Sprintf("%d", userID)

	msgs, err := q.ch.ConsumeWithContext(ctx,
		name,     // queue
		consumer, // consumer
		false,    // auto-ack
		false,    // exclusive
		false,    // no-local
		false,    // no-wait
		nil,      // args
	)
	if err != nil {
		return nil, fmt.Errorf("consume: %w", err)
	}

	return msgs, nil
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
