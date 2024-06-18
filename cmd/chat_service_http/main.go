package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/lam0glia/chat-system/bootstrap"
	"github.com/lam0glia/chat-system/domain"
	"github.com/lam0glia/chat-system/queue"
	"github.com/lam0glia/chat-system/repository"
	"github.com/lam0glia/chat-system/use_case"
	"github.com/sony/sonyflake"
)

var (
	sendMessageUseCase domain.SendMessageUseCase
	updatePresence     domain.UpdatePresenceUseCase
	uidGenerator       domain.UIDGenerator
	messageQueue       domain.MessageQueue
	messageReader      domain.MessageReader
	presenceWriter     domain.PresenceWriter
	presenceConsumer   domain.PresenceConsumer
	app                *bootstrap.App
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func init() {
	var err error
	app, err = bootstrap.NewApp()
	panicOnError(err, "failed to bootstrap app")

	start, err := time.Parse("2006-01-02", app.Env.UIDGeneratorStartTime)
	panicOnError(err, "Failed to parse start time")

	settings := sonyflake.Settings{
		StartTime: start,
	}

	uidGenerator, err = sonyflake.New(settings)
	panicOnError(err, "Failed to configure sonyflake")

	messageRepo := repository.NewMessage(app.CassandraSession)

	sendMessageUseCase = use_case.NewSendMessage(
		messageRepo,
		messageQueue,
		uidGenerator,
	)

	messageReader = messageRepo

	presenceRepository := repository.NewPresence(app.RedisClient)

	presenceWriter = presenceRepository

	presenceQueue, err := queue.NewPresence(app.RabbitMQConnection)
	panicOnError(err, "failed to initialize presence queue")

	updatePresence = use_case.NewUpdatePresence(
		presenceRepository,
		presenceRepository,
		presenceQueue,
	)

	presenceConsumer = presenceQueue

	gin.SetMode(gin.DebugMode)
}

func main() {
	eng := gin.Default()

	eng.SetTrustedProxies(nil)

	eng.GET("/ws", wsHandler)
	eng.POST("/users", registerUserHandler)
	eng.GET("/messages", listMessagesHandler)
	eng.GET("/ws/presence", presenceWSHandler)

	eng.Run(fmt.Sprintf(":%d", app.Env.HTTPPortNumber))
}

func wsHandler(c *gin.Context) {
	from, err := strconv.ParseUint(c.Query("from"), 10, 64)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		abortWithInternalError(c, err)
		return
	}

	defer conn.Close()

	ctx := c.Request.Context()
	close := make(chan bool, 1)

	var wg sync.WaitGroup

	wg.Add(1)
	go func(ctx context.Context, close chan bool) {
		defer func() {
			close <- true
		}()

		defer wg.Done()

		for {
			_, r, err := conn.NextReader()
			if err != nil {
				if _, is := err.(*websocket.CloseError); !is {
					log.Printf("read message from peer returned an error: %s", err)
				}

				break
			}

			message := domain.SentMessageRequest{
				From: from,
			}

			if err = json.NewDecoder(r).Decode(&message); err != nil {
				log.Printf("failed to decode message: %s", err)
			} else if err = sendMessageUseCase.Execute(ctx, &message); err != nil {
				log.Printf("failed to send message: %s", err)
			}
		}
	}(ctx, close)

	wg.Add(1)
	go func(close chan bool) {
		defer wg.Done()

		err := messageQueue.Consume(from, conn, close)
		if err != nil {
			log.Println("Failed to consume messages: ", err.Error())
		}
	}(close)

	wg.Wait()
}

func registerUserHandler(c *gin.Context) {
	uid, err := uidGenerator.NextID()
	if err != nil {
		abortWithInternalError(c, err)
		return
	}

	err = messageQueue.NewUserQueue(uid)
	if err != nil {
		abortWithInternalError(c, err)
		return
	}

	c.Status(http.StatusCreated)
}

func listMessagesHandler(c *gin.Context) {
	var params domain.ListMessageRequest
	err := c.ShouldBindQuery(&params)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	messages, err := messageReader.List(
		c.Request.Context(),
		params.From,
		params.To,
		params.BeforeID,
		10)
	if err != nil {
		abortWithInternalError(c, err)
	}

	c.JSON(http.StatusOK, messages)
}

func presenceWSHandler(c *gin.Context) {
	from, err := strconv.ParseUint(c.Query("from"), 10, 64)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	ctx := c.Request.Context()

	presence := domain.Presence{
		Online: true,
		UserID: from,
	}

	defer func() {
		presence.Online = false

		var err error
		if err = updatePresence.Execute(ctx, &presence); err != nil {
			log.Println("err: update presence")
		}
	}()

	if err = updatePresence.Execute(ctx, &presence); err != nil {
		abortWithInternalError(c, err)
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		abortWithInternalError(c, err)
		return
	}

	defer conn.Close()

	ticker := time.NewTicker(5 * time.Second)

	conn.SetPongHandler(func(appData string) error {
		log.Println("Pong received")

		var err error
		if err = presenceWriter.Update(ctx, from, true); err != nil {
			log.Printf("err: update presence: %s", err)
		}

		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		ticker.Reset(5 * time.Second)

		return nil
	})

	stop := make(chan struct{})

	conn.SetReadDeadline(time.Now().Add(35 * time.Second))

	var wg sync.WaitGroup

	wg.Add(1)
	go func(conn *websocket.Conn, stop chan struct{}) {
		defer wg.Done()
		defer log.Println("Closed ping goroutine")

		for {
			select {
			case <-stop:
				return
			default:
				<-ticker.C
				err = conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
				ticker.Reset(30 * time.Second)
				if err != nil {
					if err != websocket.ErrCloseSent {
						log.Printf("err: send ping message: %s", err)
						ticker.Reset(5 * time.Second)
					}
				} else {
					log.Println("Ping sent")
				}
			}

		}
	}(conn, stop)

	wg.Add(1)
	go func(c *websocket.Conn, stop chan struct{}) {
		defer func() {
			log.Println("Closed message goroutine")
			close(stop)
			ticker.Reset(1)
			wg.Done()
		}()

		for {
			if _, _, err := c.NextReader(); err != nil {
				if closeErr, is := err.(*websocket.CloseError); is {
					log.Printf("Close message received: [%d] %s", closeErr.Code, closeErr.Text)
				} else {
					log.Printf("err: read message: %s", err)
				}

				break
			}
		}
	}(conn, stop)

	go presenceConsumer.Consume(ctx, from, conn, stop)

	wg.Wait()
}

func abortWithInternalError(c *gin.Context, err error) {
	log.Panicln(err)
	c.AbortWithStatus(http.StatusInternalServerError)
}

func panicOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}
