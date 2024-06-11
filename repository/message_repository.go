package repository

import (
	"context"

	"github.com/lam0glia/chat-system/domain"
)

type message struct {
}

func (r *message) Insert(ctx context.Context, message *domain.Message) error {
	return nil
}

func NewMessage() *message {
	return &message{}
}
