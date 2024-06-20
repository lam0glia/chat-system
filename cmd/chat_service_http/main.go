package main

import (
	"fmt"
	"log"
	"time"

	"github.com/lam0glia/chat-system/bootstrap"
	"github.com/lam0glia/chat-system/http/handler"
	"github.com/lam0glia/chat-system/http/route"
	"github.com/lam0glia/chat-system/queue"
	"github.com/lam0glia/chat-system/repository"
	"github.com/lam0glia/chat-system/use_case"
	"github.com/sony/sonyflake"
)

var (
	h   *handler.Handler
	env *bootstrap.Env
)

func init() {
	app, err := bootstrap.NewApp()
	panicOnError(err, "Failed to bootstrap app")

	start, err := time.Parse("2006-01-02", app.Env.UIDGeneratorStartTime)
	panicOnError(err, "Failed to parse start time")

	settings := sonyflake.Settings{
		StartTime: start,
	}

	uidGenerator, err := sonyflake.New(settings)
	panicOnError(err, "Failed to configure sonyflake")

	messageRepo := repository.NewMessage(app.CassandraSession)
	presenceRepository := repository.NewPresence(app.RedisClient)

	messageQueue, err := queue.NewMessage(app.RabbitMQConnection)
	panicOnError(err, "Failed to initialize message queue")

	presenceQueue, err := queue.NewPresence(app.RabbitMQConnection)
	panicOnError(err, "failed to initialize presence queue")

	sendMessageUseCase := use_case.NewSendMessage(
		messageRepo,
		messageQueue,
		uidGenerator,
	)
	updatePresenceUseCase := use_case.NewUpdatePresence(
		presenceRepository,
		presenceRepository,
		presenceQueue,
	)

	chatHandler := handler.NewChat(
		sendMessageUseCase,
		messageRepo,
		app.RabbitMQConnection,
	)
	presenceHandler := handler.NewPresence(
		updatePresenceUseCase,
		presenceRepository,
		presenceQueue,
	)

	h = handler.NewHandler(
		chatHandler,
		presenceHandler,
	)

	env = app.Env
}

func main() {
	eng := route.Setup(h, env.EnvironmentName)

	eng.Run(fmt.Sprintf(":%d", env.HTTPPortNumber))
}

func panicOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}
