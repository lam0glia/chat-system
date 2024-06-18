package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Chat     *Chat
	Presence *Presence
}

func NewHandler(
	Chat *Chat,
	Presence *Presence,
) *Handler {
	return &Handler{
		Chat:     Chat,
		Presence: Presence,
	}
}

func abortWithInternalError(c *gin.Context, err error) {
	log.Panicln(err)
	c.AbortWithStatus(http.StatusInternalServerError)
}
