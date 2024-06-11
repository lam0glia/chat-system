package use_case

import (
	"context"
	"fmt"

	"github.com/lam0glia/chat-system/domain"
)

type sendMessage struct {
	messageWriter domain.MessageWriter
}

func (uc *sendMessage) Execute(ctx context.Context, message *domain.Message) error {
	var err error

	if err = uc.messageWriter.Insert(ctx, message); err != nil {
		return fmt.Errorf("failed to insert message: %w", err)
	}

	return nil
}

func NewSendMessage(
	messageWriter domain.MessageWriter,
) *sendMessage {
	return &sendMessage{
		messageWriter: messageWriter,
	}
}
