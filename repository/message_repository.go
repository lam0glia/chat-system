package repository

import (
	"context"
	"time"

	"github.com/gocql/gocql"
	"github.com/lam0glia/chat-system/domain"
)

type message struct {
	db *gocql.Session
}

func (r *message) Insert(ctx context.Context, message *domain.Message) error {
	return r.db.Query(
		"INSERT INTO messages (id, content, from_id, to_id, created_at) VALUES (?, ?, ?, ?, ?)",
		message.ID,
		message.Content,
		message.FromID,
		message.ToID,
		message.CreatedAt,
	).WithContext(ctx).Exec()
}

func (r *message) List(ctx context.Context, before time.Time, skip int) ([]domain.Message, error) {
	r.db.Query(
		"SELECT id, sender_id, receiver_id, content, created_at WHERE id < ?",
	)

	return nil, nil
}

func NewMessage(session *gocql.Session) *message {
	return &message{
		db: session,
	}
}
