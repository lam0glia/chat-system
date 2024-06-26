package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/lam0glia/chat-system/domain"
	"github.com/lam0glia/chat-system/http/middleware"
	"github.com/lam0glia/chat-system/queue"
	"github.com/lam0glia/chat-system/use_case"
	"github.com/lam0glia/chat-system/worker"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Chat struct {
	queueConn     *amqp.Connection
	upgrader      websocket.Upgrader
	messageReader domain.MessageReader
	messageWriter domain.MessageWriter
	uidGenerator  domain.UIDGenerator
}

type chatWS struct {
	conn               *websocket.Conn
	userID             uint64
	wg                 sync.WaitGroup
	sendMessageUseCase domain.SendMessageUseCase
}

func (h *Chat) WebSocket(c *gin.Context) {
	from := middleware.GetUserIDFromContext(c)

	consumer, producer, err := queue.NewMessage(h.queueConn)
	if err != nil {
		abortWithInternalError(c, err)
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("err: %s", err)
		return
	}

	defer conn.Close()

	ctx := c.Request.Context()

	writer, err := worker.NewMessageWriter(ctx, conn, consumer, from)
	if err != nil {
		abortWithInternalError(c, err)
		return
	}

	sendMessageUseCase := use_case.NewSendMessage(
		h.messageWriter,
		producer,
		h.uidGenerator,
	)

	ws := newChatWS(conn, from, sendMessageUseCase)

	ws.wg.Add(1)
	go ws.read(ctx, consumer)

	ws.wg.Add(1)
	go ws.write(writer)

	ws.wg.Wait()
}

func (ws *chatWS) write(w worker.Worker) {
	defer ws.done("write")

	w.Run()
}

func (ws *chatWS) read(
	ctx context.Context,
	consumer domain.MessageQueueConsumer,
) {
	defer func() {
		// close the writer worker channel
		consumer.Close()
		ws.conn.Close()
		ws.done("read")
	}()

	for {
		_, r, err := ws.conn.NextReader()
		if err != nil {
			if _, is := err.(*websocket.CloseError); !is {
				log.Printf("err: read peer: %s", err)
			}

			break
		}

		message := domain.SentMessageRequest{
			From: ws.userID,
		}

		if err := json.NewDecoder(r).Decode(&message); err != nil {
			log.Printf("err: decode json: %s", err)
		} else if err = ws.sendMessageUseCase.Execute(ctx, &message); err != nil {
			log.Printf("err: send message: %s", err)
		}
	}
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

func (ws *chatWS) done(name string) {
	log.Printf("Closed %s goroutine", name)
	ws.wg.Done()
}

func newChatWS(
	conn *websocket.Conn,
	userID uint64,
	sendMessageUseCase domain.SendMessageUseCase,
) *chatWS {
	return &chatWS{
		conn:               conn,
		userID:             userID,
		sendMessageUseCase: sendMessageUseCase,
	}
}

func NewChat(
	messageReader domain.MessageReader,
	queueConn *amqp.Connection,
	uidGenerator domain.UIDGenerator,
	messageWriter domain.MessageWriter,
) *Chat {
	return &Chat{
		upgrader: websocket.Upgrader{
			ReadBufferSize:   5120,
			WriteBufferSize:  5120,
			Error:            nil,
			HandshakeTimeout: 10 * time.Second,
		},
		messageReader: messageReader,
		queueConn:     queueConn,
		uidGenerator:  uidGenerator,
		messageWriter: messageWriter,
	}
}
