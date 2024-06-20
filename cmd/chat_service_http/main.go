package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
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

	presenceQueue, err := queue.NewPresence(app.RabbitMQConnection)
	panicOnError(err, "failed to initialize presence queue")

	updatePresenceUseCase := use_case.NewUpdatePresence(
		presenceRepository,
		presenceRepository,
		presenceQueue,
	)

	chatHandler := handler.NewChat(
		messageRepo,
		app.RabbitMQConnection,
		uidGenerator,
		messageRepo,
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
	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)

	defer cancel()

	handler := route.Setup(h, env.EnvironmentName)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", env.HTTPPortNumber),
		Handler: handler,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("err: serve http: %s", err.Error())
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		<-ctx.Done()

		log.Println("Shutting down http server...")
		// todo: "Shutdown does not attempt to close nor wait for hijacked connections such as WebSockets"
		server.Shutdown(context.Background())
	}()

	wg.Wait()
}

func panicOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}
