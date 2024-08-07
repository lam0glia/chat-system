package route

import (
	"github.com/gin-gonic/gin"
	"github.com/lam0glia/chat-system/bootstrap"
	"github.com/lam0glia/chat-system/http/middleware"
)

const v1Prefix = "/v1"

func Setup(app *bootstrap.App) *gin.Engine {
	if app.Env.EnvironmentName == bootstrap.ProductionEnvironmentName {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	eng := gin.Default()

	eng.SetTrustedProxies(nil)

	v1 := eng.Group(v1Prefix, middleware.NewUser)
	{
		chatRouter(v1, app)
	}

	return eng
}
