package route

import (
	"github.com/gin-gonic/gin"
	"github.com/lam0glia/chat-system/bootstrap"
	"github.com/lam0glia/chat-system/event"
	"github.com/lam0glia/chat-system/http/handler"
	"github.com/lam0glia/chat-system/repository"
	"github.com/lam0glia/chat-system/service"
	"github.com/lam0glia/chat-system/websocket_buffer"
)

func chatRouter(r gin.IRouter, app *bootstrap.App) {
	chatRepository := repository.NewChat(app.CassandraSession)
	messageBroker := event.NewRabbitMQ(app.RabbitMQConnection)
	presenceRepository := repository.NewPresence(app.RedisClient)
	presenceService := service.NewPresence(presenceRepository)
	writeBuffer := &websocket_buffer.WriteBuffer{}

	h := handler.NewChat(
		app.RabbitMQConnection,
		app.SonyFlake,
		chatRepository,
		messageBroker,
		presenceService,
		writeBuffer,
	)

	chat := r.Group("/chat")

	chat.GET("/ws", h.WebSocket)
	chat.GET("/messages", h.ListMessages)
}
