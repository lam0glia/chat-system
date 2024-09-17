package queue

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
	"github.com/lam0glia/chat-system/domain"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Chat struct {
	ch        *amqp.Channel
	queueName string
}

func NewChat(conn *amqp.Connection, userID uint64) (*Chat, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open channel: %w", err)
	}

	chat := &Chat{
		ch:        ch,
		queueName: fmt.Sprintf("%d", userID),
	}

	_, err = ch.QueueDeclare(
		chat.queueName, // name
		true,           // durable
		false,          // delete when unused
		false,          // exclusive
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("setup queue: %w", err)
	}

	return chat, nil
}

func (q *Chat) PublishMessage(msg *domain.Message) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	err = q.ch.Publish(
		"",                          // exchange
		fmt.Sprintf("%d", msg.ToID), // routing key
		false,                       // mandatory
		false,                       // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        b,
		})
	if err != nil {
		return fmt.Errorf("publish message: %w", err)
	}

	return nil
}

// desacoplar websocket.Conn
// Call Close to stop consuming
func (q *Chat) ConsumeMessages(conn *websocket.Conn) error {
	msgs, err := q.ch.Consume(
		q.queueName, // queue
		"",          // consumer
		false,       // auto-ack
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)
	if err != nil {
		return err
	}

	for d := range msgs {
		var msg domain.Message

		if err = json.Unmarshal(d.Body, &msg); err != nil {
			log.Printf("err: json decode: %s", err)

			d.Reject(false)

			continue
		}

		if err = conn.WriteJSON(msg); err != nil {
			log.Printf("err: write json: %s", err)
			continue
		}

		if err = d.Ack(false); err != nil {
			log.Printf("err: ack: %s", err)
			continue
		}
	}

	return nil
}

func (q *Chat) Close() {
	err := q.ch.Close()

	log.Println("err:", err)
}
