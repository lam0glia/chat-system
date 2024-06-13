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

	uidGenerator, err := sonyflake.New(settings)
	panicOnError(err, "Failed to configure sonyflake")

	sendMessageUseCase = use_case.NewSendMessage(
		repository.NewMessage(session),
		messageQueue,
		uidGenerator,
	)

	gin.SetMode(gin.DebugMode)
}

func main() {
	eng := gin.Default()

	eng.SetTrustedProxies(nil)

	eng.GET("/ws", wsHandler)
	eng.POST("/users", registerUserHandler)

	eng.Run(":8080")
}

func wsHandler(c *gin.Context) {
	from, err := strconv.ParseUint(c.Query("from"), 10, 64)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	ch, err := queueConnection.Channel()
	if err != nil {
		abortWithInternalError(c, err)
		return
	}

	defer ch.Close()

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		abortWithInternalError(c, err)
		return
	}

	defer conn.Close()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			_, reader, err := conn.NextReader()
			if err != nil {
				log.Printf("failed to read message received from peer: %s", err.Error())
				break
			}

			message := domain.SentMessageRequest{
				From: from,
			}

			if err = json.NewDecoder(reader).Decode(&message); err != nil {
				log.Printf("failed to decode message from stream: %s", err)
			} else if err = sendMessageUseCase.Execute(context.TODO(), &message); err != nil {
				log.Println(err.Error())
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		err := messageQueue.Consume(from, conn)
		if err != nil {
			log.Println("Failed to consume messages: ", err.Error())
		}
	}()

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

func abortWithInternalError(c *gin.Context, err error) {
	log.Panicln(err)
	c.AbortWithStatus(http.StatusInternalServerError)
}

func panicOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}
