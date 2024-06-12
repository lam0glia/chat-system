package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/gocql/gocql"
	"github.com/gorilla/websocket"
	"github.com/lam0glia/chat-system/domain"
	"github.com/lam0glia/chat-system/queue"
	"github.com/lam0glia/chat-system/repository"
	"github.com/lam0glia/chat-system/service"
	"github.com/lam0glia/chat-system/use_case"

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

	uidGenerator = service.NewRandInt()

	sendMessageUseCase = use_case.NewSendMessage(
		repository.NewMessage(session),
		messageQueue,
		uidGenerator,
	)
}

func main() {
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/users", registerUserHandler)

	log.Println("Listening port 8080")
	http.ListenAndServe(":8080", nil)
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	from, err := strconv.Atoi(query.Get("from"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ch, err := queueConnection.Channel()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	defer ch.Close()

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("failed to oppen websocket connection: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
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

			message := domain.Message{
				SenderID: from,
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

func registerUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	uid, err := uidGenerator.NewUID(context.TODO())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = messageQueue.NewUserQueue(uid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func panicOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}
