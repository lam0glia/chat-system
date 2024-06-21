package use_case

import (
	"context"
	"fmt"

	"github.com/lam0glia/chat-system/domain"
)

type sendMessage struct {
	messageWriter domain.MessageWriter
	messageQueue  domain.MessageQueueProducer
	uidGenerator  domain.UIDGenerator
}

func (uc *sendMessage) Execute(ctx context.Context, messageRequest *domain.SentMessageRequest) error {
	id, err := uc.uidGenerator.NextID()
	if err != nil {
		return fmt.Errorf("failed to generate a new unique id: %w", err)
	}

	message := domain.NewMessage(
		id,
		messageRequest.From,
		messageRequest.To,
		messageRequest.Content,
	)

	if err = uc.messageWriter.Insert(ctx, message); err != nil {
		return fmt.Errorf("failed to insert message: %w", err)
	}

	if err = uc.messageQueue.Publish(ctx, message); err != nil {
		return fmt.Errorf("faield to send message to queue: %w", err)
	}

	return nil
}

func NewSendMessage(
	messageWriter domain.MessageWriter,
	messageQueue domain.MessageQueueProducer,
	uidGenerator domain.UIDGenerator,
) *sendMessage {
	return &sendMessage{
		messageWriter: messageWriter,
		messageQueue:  messageQueue,
		uidGenerator:  uidGenerator,
	}
}
