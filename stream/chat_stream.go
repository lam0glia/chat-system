package stream

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/lam0glia/chat-system/domain"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Chat struct {
	ch *amqp.Channel
	id string
}

func NewChat(conn *amqp.Connection, userID uint64) (*Chat, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open channel: %w", err)
	}

	chat := &Chat{
		ch: ch,
		id: fmt.Sprintf("%d", userID),
	}

	_, err = ch.QueueDeclare(
		chat.id, // name
		true,    // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("setup queue: %w", err)
	}

	return chat, nil
}

func (s *Chat) DispatchMessage(msg *domain.Message) error {
	err := s.publish(fmt.Sprintf("%d", msg.ToID), *msg)
	if err != nil {
		return fmt.Errorf("publish message: %w", err)
	}

	return nil
}

func (s *Chat) publish(key string, decodedBody any) error {
	body, err := json.Marshal(decodedBody)
	if err != nil {
		return fmt.Errorf("json encode body: %w", err)
	}

	err = s.ch.Publish(
		"",    // exchange
		key,   // routing key
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}

// Call Close to stop consuming
func (s *Chat) ConsumeMessages(conn domain.WebsocketConnection) error {
	msgs, err := s.ch.Consume(
		s.id,  // queue
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return err
	}

	for d := range msgs {
		var msg domain.MessageReceivedResponse

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
	q.ch.Close()
}
