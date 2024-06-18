package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/lam0glia/chat-system/domain"
	"github.com/lam0glia/chat-system/http/middleware"
)

type Chat struct {
	upgrader           websocket.Upgrader
	sendMessageUseCase domain.SendMessageUseCase
	messageQueue       domain.MessageQueue
	messageReader      domain.MessageReader
}

func (h *Chat) WebSocket(c *gin.Context) {
	from := middleware.GetUserIDFromContext(c)

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		abortWithInternalError(c, err)
		return
	}

	defer conn.Close()

	ctx := c.Request.Context()
	close := make(chan bool, 1)

	var wg sync.WaitGroup

	wg.Add(1)
	go func(ctx context.Context, close chan bool) {
		defer func() {
			close <- true
		}()

		defer wg.Done()

		for {
			_, r, err := conn.NextReader()
			if err != nil {
				if _, is := err.(*websocket.CloseError); !is {
					log.Printf("read message from peer returned an error: %s", err)
				}

				break
			}

			message := domain.SentMessageRequest{
				From: from,
			}

			if err = json.NewDecoder(r).Decode(&message); err != nil {
				log.Printf("failed to decode message: %s", err)
			} else if err = h.sendMessageUseCase.Execute(ctx, &message); err != nil {
				log.Printf("failed to send message: %s", err)
			}
		}
	}(ctx, close)

	wg.Add(1)
	go func(close chan bool) {
		defer wg.Done()

		err := h.messageQueue.Consume(from, conn, close)
		if err != nil {
			log.Println("Failed to consume messages: ", err.Error())
		}
	}(close)

	wg.Wait()
}

func (h *Chat) ListMessages(c *gin.Context) {
	var params domain.ListMessageRequest
	err := c.ShouldBindQuery(&params)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	from := middleware.GetUserIDFromContext(c)

	messages, err := h.messageReader.List(
		c.Request.Context(),
		from,
		params.To,
		params.BeforeID,
		10)
	if err != nil {
		abortWithInternalError(c, err)
	}

	c.JSON(http.StatusOK, messages)
}

func NewChat(
	sendMessageUseCase domain.SendMessageUseCase,
	messageQueue domain.MessageQueue,
	messageReader domain.MessageReader,
) *Chat {
	return &Chat{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  5120,
			WriteBufferSize: 5120,
		},
		sendMessageUseCase: sendMessageUseCase,
		messageQueue:       messageQueue,
		messageReader:      messageReader,
	}
}
