package repository

import (
	"context"
	"fmt"

	"github.com/gocql/gocql"
	"github.com/lam0glia/chat-system/domain"
)

const pairFormat = "%d%d"

type message struct {
	db *gocql.Session
}

func (r *message) pair(fromID, toID uint64) string {
	if fromID < toID {
		return fmt.Sprintf(pairFormat, fromID, toID)
	} else {
		return fmt.Sprintf(pairFormat, toID, fromID)
	}
}

func (r *message) Insert(ctx context.Context, message *domain.Message) error {

	return r.db.Query(
		"INSERT INTO messages (id, content, from_id, to_id, created_at, pair) VALUES (?, ?, ?, ?, ?, ?)",
		message.ID,
		message.Content,
		message.FromID,
		message.ToID,
		message.CreatedAt,
		r.pair(message.FromID, message.ToID),
	).WithContext(ctx).Exec()
}

func (r *message) List(ctx context.Context, fromID, toID, beforeID uint64, limit int) ([]domain.Message, error) {
	scanner := r.db.Query(
		`SELECT
			id, content, created_at, from_id, to_id
		FROM
			messages
		WHERE
			pair = ?
			AND id < ?
		ORDER BY id DESC LIMIT ?;`,
		r.pair(fromID, toID),
		beforeID,
		limit,
	).WithContext(ctx).Iter().Scanner()

	var (
		messages []domain.Message
		err      error
	)

	for scanner.Next() {
		var message domain.Message

		err = scanner.Scan(
			&message.ID,
			&message.Content,
			&message.CreatedAt,
			&message.FromID,
			&message.ToID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %s", err)
		}

		messages = append(messages, message)
	}

	if err = scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to close scanner: %s", err)
	}

	return messages, nil
}

func NewMessage(session *gocql.Session) *message {
	return &message{
		db: session,
	}
}
