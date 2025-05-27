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
	queueConn            *amqp.Connection
	upgrader             websocket.Upgrader
	chatRepository       domain.ChatRepository
	uidGenerator         domain.UIDGenerator
	presenceService      domain.PresenceService
	channelFactory       domain.ChannelFactory
	websocketWriteBuffer domain.WebsocketWriteBuffer
}

/*
IMPORTANT: there are two channels that use the same websocket connection
to write to the client and its not concurrency safe.
*/
func (h *Chat) WebSocket(c *gin.Context) {
	userID := middleware.GetUserIDFromContext(c)

	chatStream, err := stream.NewChat(h.queueConn, userID)
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

	channel, err := h.channelFactory.NewChannel()
	if err != nil {
		abortWithInternalError(c, err)
		return
	}

	ws, err := newChatWS(
		c,
		h.upgrader,
		userID,
		sendMessageUseCase,
		chatStream,
		h.presenceService,
		h.websocketWriteBuffer,
	)
	if err != nil {
		log.Printf("err: upgrade: %s", err)
		return
	}

	defer ws.close()

	h.presenceService.SetChannel(channel)

	ctx := c.Request.Context()

	defer func() {
		if err = h.presenceService.SetUserOffline(ctx, userID); err != nil {
			log.Printf("err: set user status offline: %s", err.Error())
		}
	}()

	if err = h.presenceService.SetUserOnline(ctx, userID); err != nil {
		log.Printf("err: set user status online: %s", err.Error())
	}

	go ws.readFromClient(ctx)

	// TODO: Close goroutine
	go h.websocketWriteBuffer.DeliveryToClient()

	// TODO: Handle error
	go h.presenceService.SubscribeUserPresenceUpdate(h.websocketWriteBuffer)

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
	channelFactory domain.ChannelFactory,
	presenceService domain.PresenceService,
	websocketWriteBuffer domain.WebsocketWriteBuffer,
) *Chat {
	return &Chat{
		upgrader: websocket.Upgrader{
			HandshakeTimeout: 10 * time.Second,
		},
		queueConn:            queueConn,
		uidGenerator:         uidGenerator,
		chatRepository:       chatRepository,
		channelFactory:       channelFactory,
		presenceService:      presenceService,
		websocketWriteBuffer: websocketWriteBuffer,
	}
}

func abortWithInternalError(c *gin.Context, err error) {
	log.Println(err)
	c.AbortWithStatus(http.StatusInternalServerError)
}
