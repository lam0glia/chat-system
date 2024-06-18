package route

import (
	"github.com/gin-gonic/gin"
	"github.com/lam0glia/chat-system/http/handler"
)

func chatRouter(r gin.IRouter, h *handler.Chat) {
	chat := r.Group("/chat")

	chat.GET("/ws", h.WebSocket)
	chat.GET("/chat/messages")
}
