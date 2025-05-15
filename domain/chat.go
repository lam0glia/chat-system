package domain

import (
	"context"
	"time"
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

type SendMessageRequest struct {
	From    uint64
	To      uint64 `json:"to"`
	Content string `json:"content"`
}

type MessageReceivedResponse struct {
	ID        uint64    `json:"id"`
	FromID    uint64    `json:"from"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type ListMessageRequest struct {
	BeforeID *uint64 `form:"beforeId"`
	To       uint64  `form:"to" binding:"required"`
}

type ChatRepository interface {
	InsertMessage(ctx context.Context, message *Message) error
	ListMessages(
		ctx context.Context,
		fromID,
		toID uint64,
		beforeID *uint64,
		limit int,
	) ([]Message, error)
}

type SendMessageUseCase interface {
	Execute(ctx context.Context, message *SendMessageRequest) error
}

type ChatStram interface {
	DispatchMessage(*Message) error
	ConsumeMessages(WebsocketConnection) error
}
