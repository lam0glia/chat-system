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

func (q *message) Consume(userID int, conn *websocket.Conn) error {
	msgs, err := q.ch.Consume(
		q.getQueueName(userID), // queue
		"",                     // consumer
		true,                   // auto-ack
		false,                  // exclusive
		false,                  // no-local
		false,                  // no-wait
		nil,                    // args
	)
	if err != nil {
		return err
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
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever

	return nil
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
