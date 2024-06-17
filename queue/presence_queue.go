package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
	"github.com/lam0glia/chat-system/domain"
	amqp "github.com/rabbitmq/amqp091-go"
)

type presence struct {
	ch *amqp.Channel
}

const presenceExchange = "presence"

func (q *presence) Consume(ctx context.Context, userID uint64, conn *websocket.Conn, close chan struct{}) error {
	qDeclared, err := q.ch.QueueDeclare(
		"",    // name
		false, // durable
		false, // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	err = q.ch.QueueBind(
		qDeclared.Name,            // queue name
		fmt.Sprintf("%d", userID), // routing key
		presenceExchange,          // exchange
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue to an exchange: %w", err)
	}

	msgs, err := q.ch.ConsumeWithContext(ctx,
		qDeclared.Name, // queue
		"",             // consumer
		false,          // auto-ack
		false,          // exclusive
		false,          // no-local
		false,          // no-wait
		nil,            // args
	)
	if err != nil {
		return fmt.Errorf("failed to consume messages: %w", err)
	}

	defer log.Println("Closed consumer queue goroutine")

	go func() {
		for d := range msgs {
			var presence domain.Presence

			err := json.Unmarshal(d.Body, &presence)
			if err != nil {
				log.Println("failed to decode json: " + err.Error())
				continue
			}

			if err = conn.WriteJSON(presence); err != nil {
				log.Println("failed to write message: " + err.Error())
				break
			}

			d.Ack(false)
		}
	}()

	<-close

	return nil
}

func (q *presence) Publish(ctx context.Context, presence *domain.Presence, toUserID uint64) error {
	b, err := json.Marshal(presence)
	if err != nil {
		return fmt.Errorf("failed to encode json: %w", err)
	}

	err = q.ch.PublishWithContext(ctx,
		presenceExchange,
		fmt.Sprintf("%d", toUserID),
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        b,
		})
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

func NewPresence(
	conn *amqp.Connection,
) (*presence, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	err = ch.ExchangeDeclare(
		presenceExchange,    // name
		amqp.ExchangeDirect, // type
		true,                // durable
		false,               // auto-deleted
		false,               // internal
		false,               // no-wait
		nil,                 // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	return &presence{
		ch: ch,
	}, nil
}
