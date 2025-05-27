package service

import (
	"context"
	"fmt"

	"github.com/lam0glia/chat-system/domain"
)

type presenceService struct {
	repository domain.PresenceRepository
	channel    domain.StreamChannel
}

func (s *presenceService) SetChannel(channel domain.StreamChannel) {
	s.channel = channel
}

func (s *presenceService) SetUserOnline(ctx context.Context, userID uint64) error {
	err := s.repository.UpdateUserStatus(ctx, userID, domain.PresenceStatusOnline)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	body := domain.Presence{
		Status: domain.PresenceStatusOnline,
		UserID: userID,
	}

	if err = s.channel.Publish(
		domain.ChannelExchangePresence,
		"",
		body,
	); err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	return nil
}

func (s *presenceService) RefreshUserPresence(ctx context.Context, userID uint64) error {
	return s.repository.SetKeyExpiration(ctx, userID)
}

func (s *presenceService) SubscribeUserPresenceUpdate(buff domain.WebsocketWriteBuffer) error {
	return s.channel.Subscribe(buff)
}

func (s *presenceService) SetUserOffline(ctx context.Context, userID uint64) error {
	err := s.repository.DeleteUserStatus(ctx, userID)
	if err != nil {
		return fmt.Errorf("delete user status: %w", err)
	}

	body := domain.Presence{
		Status: domain.PresenceStatusOffline,
		UserID: userID,
	}

	if err = s.channel.Publish(
		domain.ChannelExchangePresence,
		"",
		body,
	); err != nil {
		return fmt.Errorf("publish message to channel: %w", err)
	}

	s.channel.Close()

	return nil
}

func NewPresence(repository domain.PresenceRepository) *presenceService {
	return &presenceService{
		repository: repository,
	}
}
