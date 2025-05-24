package domain

import (
	"context"
)

const PresenceStatusOnline = "online"
const PresenceStatusOffline = "offline"

type Presence struct {
	Status string `json:"status"`
	UserID uint64 `json:"userId"`
}

type UpdatePresenceUseCase interface {
	Execute(ctx context.Context, presence *Presence) error
}

type PresenceService interface {
	// Call "SetUserOffline" to close Channel
	SetChannel(channel StreamChannel)
	SetUserOnline(ctx context.Context, userID uint64) error
	RefreshUserPresence(ctx context.Context, userID uint64) error
	SubscribeUserPresenceUpdate(conn WebsocketConnection) error
	SetUserOffline(ctx context.Context, userID uint64) error
}

type PresenceRepository interface {
	UpdateUserStatus(ctx context.Context, userID uint64, status string) error
	DeleteUserStatus(ctx context.Context, userID uint64) error
	SetKeyExpiration(ctx context.Context, userID uint64) error
}
