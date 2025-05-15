package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/lam0glia/chat-system/domain"
	"github.com/lam0glia/chat-system/http/middleware"
	"github.com/lam0glia/chat-system/stream"
	"github.com/lam0glia/chat-system/use_case"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Chat struct {
	queueConn      *amqp.Connection
	upgrader       websocket.Upgrader
	chatRepository domain.ChatRepository
	uidGenerator   domain.UIDGenerator
}

func (h *Chat) WebSocket(c *gin.Context) {
	from := middleware.GetUserIDFromContext(c)

	chatStream, err := stream.NewChat(h.queueConn, from)
	if err != nil {
		abortWithInternalError(c, err)
		return
	}

	defer chatStream.Close()

	sendMessageUseCase := use_case.NewSendMessage(
		chatStream,
		h.chatRepository,
		h.uidGenerator,
	)

	ws, err := newChatWS(c, h.upgrader, from, sendMessageUseCase, chatStream)
	if err != nil {
		log.Printf("err: upgrade: %s", err)
		return
	}

	defer ws.close()

	ctx := c.Request.Context()

	go ws.readFromClient(ctx)

	go ws.writeToClient(ctx)

	go ws.ping(ctx)

	<-ws.done
	// when the request ends, its context is canceled,
	// closing write and ping goroutines
}

func (h *Chat) ListMessages(c *gin.Context) {
	var params domain.ListMessageRequest
	err := c.ShouldBindQuery(&params)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	from := middleware.GetUserIDFromContext(c)

	messages, err := h.chatRepository.ListMessages(
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
	queueConn *amqp.Connection,
	uidGenerator domain.UIDGenerator,
	chatRepository domain.ChatRepository,
) *Chat {
	return &Chat{
		upgrader: websocket.Upgrader{
			HandshakeTimeout: 10 * time.Second,
		},
		queueConn:      queueConn,
		uidGenerator:   uidGenerator,
		chatRepository: chatRepository,
	}
}

func abortWithInternalError(c *gin.Context, err error) {
	log.Println(err)
	c.AbortWithStatus(http.StatusInternalServerError)
}
