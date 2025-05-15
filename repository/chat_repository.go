package repository

import (
	"context"
	"fmt"

	"github.com/gocql/gocql"
	"github.com/lam0glia/chat-system/domain"
)

const pairFormat = "%d.%d"

type chat struct {
	db *gocql.Session
}

func (r *chat) pair(fromID, toID uint64) string {
	if fromID < toID {
		return fmt.Sprintf(pairFormat, fromID, toID)
	} else {
		return fmt.Sprintf(pairFormat, toID, fromID)
	}
}

func (r *chat) InsertMessage(ctx context.Context, message *domain.Message) error {
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

func (r *chat) ListMessages(
	ctx context.Context,
	fromID,
	toID uint64,
	beforeID *uint64,
	limit int,
) ([]domain.Message, error) {
	query := `SELECT
			id, content, created_at, from_id, to_id
		FROM
			messages
		WHERE
			pair = ?
			%s
		ORDER BY id ASC LIMIT ?`

	var values []any

	values = append(values, r.pair(fromID, toID))

	var beforeCondition string

	if beforeID != nil {
		beforeCondition = "AND id < ?"
		values = append(values, beforeID)
	}

	values = append(values, limit)

	query = fmt.Sprintf(query, beforeCondition)

	scanner := r.db.Query(
		query,
		values...,
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

func NewChat(session *gocql.Session) *chat {
	return &chat{
		db: session,
	}
}
