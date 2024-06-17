package domain

import (
	"context"

	"github.com/gorilla/websocket"
)

type Presence struct {
	Online bool   `json:"online"`
	UserID uint64 `json:"userId"`
}

type UpdatePresenceUseCase interface {
	Execute(ctx context.Context, presence *Presence) error
}

type PresenceWriter interface {
	Update(ctx context.Context, userID uint64, online bool) error
}

type PresenceReader interface {
	IsOnline(ctx context.Context, userID uint64) (bool, error)
	GetInterestedUsers(ctx context.Context, userID uint64) ([]uint64, error)
}

type PresencePublisher interface {
	Publish(ctx context.Context, presence *Presence, toUserID uint64) error
}

type PresenceConsumer interface {
	Consume(ctx context.Context, userID uint64, conn *websocket.Conn, close chan struct{}) error
}
