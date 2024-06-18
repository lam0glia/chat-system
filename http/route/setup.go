package route

import (
	"github.com/gin-gonic/gin"
	"github.com/lam0glia/chat-system/bootstrap"
	"github.com/lam0glia/chat-system/http/handler"
	"github.com/lam0glia/chat-system/http/middleware"
)

const v1Prefix = "/v1"

func Setup(handler *handler.Handler, envName string) *gin.Engine {
	if envName == bootstrap.ProductionEnvironmentName {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	eng := gin.Default()

	eng.SetTrustedProxies(nil)

	v1 := eng.Group(v1Prefix, middleware.NewUser)
	{
		chatRouter(v1, handler.Chat)
		presenceRouter(v1, handler.Presence)
	}

	return eng
}
