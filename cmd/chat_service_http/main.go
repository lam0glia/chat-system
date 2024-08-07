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

	"github.com/lam0glia/chat-system/bootstrap"
	"github.com/lam0glia/chat-system/http/route"
)

var app *bootstrap.App

func init() {
	var err error

	app, err = bootstrap.NewApp()
	if err != nil {
		log.Panicf("Failed to bootstrap app: %s", err)
	}
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)

	defer cancel()

	handler := route.Setup(app)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", app.Env.HTTPPortNumber),
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
