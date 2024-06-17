package domain

import (
	"context"
)

type Presence struct {
	Online bool   `json:"online"`
	UserID uint64 `json:"userId"`
}

type PresenceWriter interface {
	Update(ctx context.Context, userID uint64, online bool) error
}
