package domain

import (
	"context"

	"github.com/gorilla/websocket"
)

type Message struct {
	ID         *int   `json:"id"`
	SenderID   int    `json:"from"`
	ReceiverID int    `json:"to"`
	Content    string `json:"content"`
}

type MessageWriter interface {
	Insert(ctx context.Context, message *Message) error
}

type SendMessageUseCase interface {
	Execute(ctx context.Context, message *Message) error
}

type MessageQueue interface {
	NewUserQueue(userID int) error
	Send(ctx context.Context, msg *Message) error
	Consume(userID int, conn *websocket.Conn) error
}
