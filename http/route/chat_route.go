package route

import (
	"github.com/gin-gonic/gin"
	"github.com/lam0glia/chat-system/bootstrap"
	"github.com/lam0glia/chat-system/http/handler"
	"github.com/lam0glia/chat-system/repository"
)

func chatRouter(r gin.IRouter, app *bootstrap.App) {
	messageRepo := repository.NewMessage(app.CassandraSession)

	h := handler.NewChat(
		messageRepo,
		app.RabbitMQConnection,
		app.SonyFlake,
		messageRepo,
	)

	chat := r.Group("/chat")

	chat.GET("/ws", h.WebSocket)
	chat.GET("/messages", h.ListMessages)
}
