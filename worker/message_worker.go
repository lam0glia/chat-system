package worker

import (
	"context"
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
	"github.com/lam0glia/chat-system/domain"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Worker interface {
	Run()
}

type messageWriter struct {
	conn     *websocket.Conn
	delivery <-chan amqp.Delivery
}

func (w *messageWriter) Run() {
	for d := range w.delivery {
		var m domain.Message

		err := json.Unmarshal(d.Body, &m)
		if err != nil {
			log.Printf("err: decode json: %s", err)
			continue
		}

		if err = w.conn.WriteJSON(m); err != nil {
			log.Printf("err: write ws: %s", err)
			continue
		}

		d.Ack(false)
	}
}

func NewMessageWriter(
	ctx context.Context,
	conn *websocket.Conn,
	consumer domain.MessageQueueConsumer,
	userID uint64,
) (*messageWriter, error) {
	d, err := consumer.NewConsumer(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &messageWriter{
		conn:     conn,
		delivery: d,
	}, nil
}
