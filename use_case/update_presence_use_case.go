package use_case

import (
	"context"
	"fmt"

	"github.com/lam0glia/chat-system/domain"
)

type updatePresence struct {
	presenceWriter    domain.PresenceWriter
	presenceReader    domain.PresenceReader
	presencePublisher domain.PresencePublisher
}

func (uc *updatePresence) Execute(ctx context.Context, presence *domain.Presence) error {
	err := uc.presenceWriter.Update(ctx, presence.UserID, true)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	ids, err := uc.presenceReader.GetInterestedUsers(ctx, presence.UserID)
	if err != nil {
		return fmt.Errorf("failed to get interested users: %w", err)
	}

	for _, id := range ids {
		err = uc.presencePublisher.Publish(ctx, presence, id)
		if err != nil {
			return fmt.Errorf("failed to publish presence message: %w", err)
		}
	}

	return nil
}

func NewUpdatePresence(
	presenceWriter domain.PresenceWriter,
	presenceReader domain.PresenceReader,
	presencePublisher domain.PresencePublisher,
) *updatePresence {
	return &updatePresence{
		presencePublisher: presencePublisher,
		presenceWriter:    presenceWriter,
		presenceReader:    presenceReader,
	}
}
