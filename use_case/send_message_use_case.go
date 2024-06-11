package use_case

import (
	"context"
	"fmt"

	"github.com/lam0glia/chat-system/domain"
)

type sendMessage struct {
	messageWriter domain.MessageWriter
	uidGenerator  domain.UIDGenerator
}

func (uc *sendMessage) Execute(ctx context.Context, message *domain.Message) error {
	id, err := uc.uidGenerator.NewUID(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate a new unique id: %w", err)
	}

	message.ID = &id

	if err = uc.messageWriter.Insert(ctx, message); err != nil {
		return fmt.Errorf("failed to insert message: %w", err)
	}

	return nil
}

func NewSendMessage(
	messageWriter domain.MessageWriter,
	uidGenerator domain.UIDGenerator,
) *sendMessage {
	return &sendMessage{
		messageWriter: messageWriter,
		uidGenerator:  uidGenerator,
	}
}
