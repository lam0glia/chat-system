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

const exchangeName = "messages"

type message struct {
	ch *amqp.Channel
}

func (q *message) Send(ctx context.Context, msg *domain.Message) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to encode json: %w", err)
	}

	err = q.ch.PublishWithContext(ctx,
		exchangeName,                // exchange
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

func (q *message) NewUserQueue(userID uint64) error {
	qName := q.getQueueName(userID)

	_, err := q.ch.QueueDeclare(
		qName, // name
		true,  // durable
		false, // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", qName, err)
	}

	err = q.ch.QueueBind(
		qName,
		fmt.Sprintf("%d", userID),
		exchangeName,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue %s: %w", qName, err)
	}

	return err
}

func (q *message) Consume(userID uint64, conn *websocket.Conn) error {
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
		exchangeName,              // exchange
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue to an exchange: %w", err)
	}

	msgs, err := q.ch.Consume(
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

	var forever chan struct{}

	go func() {
		for d := range msgs {
			var m domain.Message

			err := json.Unmarshal(d.Body, &m)
			if err != nil {
				log.Println("Failed to decode json: " + err.Error())
				continue
			}

			if err = conn.WriteJSON(m); err != nil {
				log.Println("Failed to write message: " + err.Error())
				continue
			}

			d.Ack(false)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever

	return nil
}

func (q *message) getQueueName(receiverID uint64) string {
	return fmt.Sprintf("msg-%d", receiverID)
}

func NewMessage(
	conn *amqp.Connection,
) (*message, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %s", err)
	}

	err = ch.ExchangeDeclare(
		exchangeName,        // name
		amqp.ExchangeDirect, // type
		true,                // durable
		false,               // auto-deleted
		false,               // internal
		false,               // no-wait
		nil,                 // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare messages exchange: %w", err)
	}

	return &message{
		ch: ch,
	}, nil
}
