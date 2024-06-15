package domain

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
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
	From     uint64 `form:"from"`
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

type MessageQueue interface {
	NewUserQueue(userID uint64) error
	Send(ctx context.Context, msg *Message) error
	Consume(userID uint64, conn *websocket.Conn, close chan bool) error
}
