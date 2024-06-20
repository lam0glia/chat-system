package domain

import (
	"context"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Message struct {
	ID        uint64    `json:"id"`
	FromID    uint64    `json:"from"`
	ToID      uint64    `json:"to"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

func NewMessage(id, fromID, toID uint64, content string) *Message {
	return &Message{
		ID:        id,
		FromID:    fromID,
		ToID:      toID,
		Content:   content,
		CreatedAt: time.Now(),
	}
}

type SentMessageRequest struct {
	From    uint64
	To      uint64 `json:"to"`
	Content string `json:"content"`
}

type ListMessageRequest struct {
	BeforeID uint64 `form:"beforeId"`
	To       uint64 `form:"to"`
}

type MessageWriter interface {
	Insert(ctx context.Context, message *Message) error
}

type MessageReader interface {
	List(ctx context.Context, fromID, toID, beforeID uint64, limit int) ([]Message, error)
}

type SendMessageUseCase interface {
	Execute(ctx context.Context, message *SentMessageRequest) error
}

type MessageQueueProducer interface {
	Publish(ctx context.Context, msg *Message) error
}

type MessageQueueConsumer interface {
	NewConsumer(ctx context.Context, userID uint64) (<-chan amqp.Delivery, error)
}
