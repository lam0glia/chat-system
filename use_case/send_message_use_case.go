package use_case

import (
	"context"
	"fmt"

	"github.com/lam0glia/chat-system/domain"
)

type sendMessage struct {
	chatStreamDispatcher domain.ChatStream
	chatRepositoryWriter domain.ChatRepository
	uidGenerator         domain.UIDGenerator
}

func (uc *sendMessage) Execute(ctx context.Context, messageRequest *domain.SendMessageRequest) error {
	id, err := uc.uidGenerator.NextID()
	if err != nil {
		return fmt.Errorf("generate new unique id: %w", err)
	}

	message := domain.NewMessage(
		id,
		messageRequest.From,
		messageRequest.To,
		messageRequest.Content,
	)

	if err = uc.chatRepositoryWriter.InsertMessage(ctx, message); err != nil {
		return fmt.Errorf("insert message: %w", err)
	}

	if err = uc.chatStreamDispatcher.DispatchMessage(message); err != nil {
		return fmt.Errorf("publish event: %w", err)
	}

	return nil
}

func NewSendMessage(
	chatStreamDispatcher domain.ChatStream,
	chatRepositoryWriter domain.ChatRepository,
	uidGenerator domain.UIDGenerator,
) *sendMessage {
	return &sendMessage{
		chatStreamDispatcher: chatStreamDispatcher,
		chatRepositoryWriter: chatRepositoryWriter,
		uidGenerator:         uidGenerator,
	}
}
