package repository

import (
	"context"
	"math/rand"

	"github.com/gocql/gocql"
	"github.com/lam0glia/chat-system/domain"
)

type message struct {
	db *gocql.Session
}

func (r *message) Insert(ctx context.Context, message *domain.Message) error {
	message.ID = rand.Intn(1000000)

	return r.db.Query(
		"INSERT INTO messages (id, content, receiver_id, sender_id) VALUES (?, ?, ?, ?)",
		message.ID,
		message.Content,
		message.ReceiverID,
		message.SenderID,
	).WithContext(ctx).Exec()
}

func NewMessage(session *gocql.Session) *message {
	return &message{
		db: session,
	}
}