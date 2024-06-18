package route

import (
	"github.com/gin-gonic/gin"
	"github.com/lam0glia/chat-system/http/handler"
)

func presenceRouter(r gin.IRouter, h *handler.Presence) {
	r.GET("/presence/ws", h.WebSocket)
}
