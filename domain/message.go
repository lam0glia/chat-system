package domain

import "context"

type Message struct {
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
