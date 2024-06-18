package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

const customUserIDHeader = "X-User-Id"
const userIDContextKey = "x-user-id"

func NewUser(c *gin.Context) {
	h := c.GetHeader(customUserIDHeader)

	userID, err := strconv.ParseUint(h, 10, 64)
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	c.Set(userIDContextKey, userID)

	c.Next()
}

func GetUserIDFromContext(c *gin.Context) uint64 {
	return c.GetUint64(userIDContextKey)
}
