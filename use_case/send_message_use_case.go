package use_case

import (
	"context"
	"fmt"

	"github.com/lam0glia/chat-system/domain"
)

type sendMessage struct {
	messageEventProducer domain.MessageEventProducer
	uidGenerator         domain.UIDGenerator
}

func (uc *sendMessage) Execute(ctx context.Context, messageRequest *domain.SentMessageRequest) error {
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

	if err = uc.messageEventProducer.PublishMessage(message); err != nil {
		return fmt.Errorf("publish event: %w", err)
	}

	return nil
}

func NewSendMessage(
	producer domain.MessageEventProducer,
	uidGenerator domain.UIDGenerator,
) *sendMessage {
	return &sendMessage{
		messageEventProducer: producer,
		uidGenerator:         uidGenerator,
	}
}
