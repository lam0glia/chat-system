package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/gorilla/websocket"
	"github.com/lam0glia/chat-system/domain"
	"github.com/lam0glia/chat-system/queue"
	"github.com/lam0glia/chat-system/repository"
	"github.com/lam0glia/chat-system/use_case"
	"github.com/sony/sonyflake"

	amqp "github.com/rabbitmq/amqp091-go"
)

const keyspace = "chat"

var (
	sendMessageUseCase domain.SendMessageUseCase
	uidGenerator       domain.UIDGenerator
	messageQueue       domain.MessageQueue
	messageReader      domain.MessageReader
	session            *gocql.Session
	queueConnection    *amqp.Connection
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func init() {
	cluster := gocql.NewCluster("172.17.0.1")

	cluster.Keyspace = keyspace

	var err error
	session, err = cluster.CreateSession()
	panicOnError(err, "Failed to initialize database session")

	queueConnection, err = amqp.Dial("amqp://user:password@localhost:5672/")
	panicOnError(err, "Failed to connect to RabbitMQ")

	messageQueue, err = queue.NewMessage(queueConnection)
	panicOnError(err, "Failed to initialize message queue")

	start, err := time.Parse("2006-01-02", "2024-06-13")
	panicOnError(err, "Failed to parse start time")

	settings := sonyflake.Settings{
		StartTime: start,
	}

	uidGenerator, err = sonyflake.New(settings)
	if err != nil {
		log.Println("cavalo")
	}
	panicOnError(err, "Failed to configure sonyflake")

	messageRepo := repository.NewMessage(session)

	sendMessageUseCase = use_case.NewSendMessage(
		messageRepo,
		messageQueue,
		uidGenerator,
	)

	messageReader = messageRepo

	gin.SetMode(gin.DebugMode)
}

func main() {
	eng := gin.Default()

	eng.SetTrustedProxies(nil)

	eng.GET("/ws", wsHandler)
	eng.POST("/users", registerUserHandler)
	eng.GET("/messages", listMessagesHandler)

	eng.Run(":8080")
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

func abortWithInternalError(c *gin.Context, err error) {
	log.Panicln(err)
	c.AbortWithStatus(http.StatusInternalServerError)
}

func panicOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}
