package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/lam0glia/chat-system/domain"
	"github.com/lam0glia/chat-system/http/middleware"
	"github.com/lam0glia/chat-system/queue"
	"github.com/lam0glia/chat-system/use_case"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Chat struct {
	queueConn     *amqp.Connection
	upgrader      websocket.Upgrader
	messageReader domain.MessageReader
	messageWriter domain.MessageWriter
	uidGenerator  domain.UIDGenerator
}

func (h *Chat) WebSocket(c *gin.Context) {
	from := middleware.GetUserIDFromContext(c)

	q, err := queue.NewMessage(h.queueConn)
	if err != nil {
		abortWithInternalError(c, err)
		return
	}

	sendMessageUseCase := use_case.NewSendMessage(
		q,
		h.uidGenerator,
	)

	ws, err := newChatWS(c, h.upgrader, from, sendMessageUseCase)
	if err != nil {
		log.Printf("err: upgrade: %s", err)
		return
	}

	defer ws.close()

	ctx := c.Request.Context()

	connClosed := make(chan bool)

	go ws.read(ctx, connClosed)

	go ws.write(ctx)

	go ws.ping(ctx)

	<-connClosed
	// when the request ends, the request context is canceled and
	// the write and ping goroutines are finished
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
	messageReader domain.MessageReader,
	queueConn *amqp.Connection,
	uidGenerator domain.UIDGenerator,
	messageWriter domain.MessageWriter,
) *Chat {
	return &Chat{
		upgrader: websocket.Upgrader{
			HandshakeTimeout: 10 * time.Second,
		},
		messageReader: messageReader,
		queueConn:     queueConn,
		uidGenerator:  uidGenerator,
		messageWriter: messageWriter,
	}
}

func abortWithInternalError(c *gin.Context, err error) {
	log.Println(err)
	c.AbortWithStatus(http.StatusInternalServerError)
}
