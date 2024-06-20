package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/lam0glia/chat-system/domain"
	"github.com/lam0glia/chat-system/http/middleware"
	"github.com/lam0glia/chat-system/queue"
	"github.com/lam0glia/chat-system/worker"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Chat struct {
	queueConn          *amqp.Connection
	upgrader           websocket.Upgrader
	sendMessageUseCase domain.SendMessageUseCase
	messageReader      domain.MessageReader
}

type chatWS struct {
	conn         *websocket.Conn
	stop         chan struct{}
	userID       uint64
	wg           sync.WaitGroup
	messageQueue domain.MessageQueue
	writerWorker worker.Worker
}

func (h *Chat) WebSocket(c *gin.Context) {
	from := middleware.GetUserIDFromContext(c)

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		abortWithInternalError(c, err)
		return
	}

	ctx := c.Request.Context()

	ws, err := newChatWS(conn, h.queueConn, from)
	if err != nil {
		conn.Close()
		abortWithInternalError(c, err)
		return
	}

	ws.wg.Add(1)
	go ws.read(ctx, h.sendMessageUseCase)

	ws.wg.Add(1)
	go ws.write()

	ws.wg.Wait()
}

func (ws *chatWS) write() {
	defer ws.done("write")

	ws.writerWorker.Run()

	<-ws.writerWorker.Done()
}

func (ws *chatWS) read(ctx context.Context, uc domain.SendMessageUseCase) {
	defer func() {
		ws.conn.Close()
		close(ws.stop)
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
		} else if err = uc.Execute(ctx, &message); err != nil {
			log.Printf("err: send message: %s", err)
			break
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
	queueConn *amqp.Connection,
	userID uint64,
) (*chatWS, error) {
	q, err := queue.NewMessage(queueConn)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize message queues: %w", err)
	}

	writer, err := worker.NewMessageWriter(conn, q, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to create writer worker: %w", err)
	}

	return &chatWS{
		conn:         conn,
		stop:         make(chan struct{}),
		userID:       userID,
		messageQueue: q,
		writerWorker: writer,
	}, nil
}

func NewChat(
	sendMessageUseCase domain.SendMessageUseCase,
	messageReader domain.MessageReader,
	queueConn *amqp.Connection,
) *Chat {
	return &Chat{
		upgrader: websocket.Upgrader{
			ReadBufferSize:   5120,
			WriteBufferSize:  5120,
			Error:            nil,
			HandshakeTimeout: 10 * time.Second,
		},
		sendMessageUseCase: sendMessageUseCase,
		messageReader:      messageReader,
		queueConn:          queueConn,
	}
}
