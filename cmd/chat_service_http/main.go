package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gocql/gocql"
	"github.com/gorilla/websocket"
	"github.com/lam0glia/chat-system/domain"
	"github.com/lam0glia/chat-system/repository"
	"github.com/lam0glia/chat-system/service"
	"github.com/lam0glia/chat-system/use_case"
)

const keyspace = "chat"

var (
	sendMessageUseCase domain.SendMessageUseCase
	session            *gocql.Session
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var writeMessage = []byte("Hi, how are you doing?")

func init() {
	cluster := gocql.NewCluster("172.17.0.1")

	cluster.Keyspace = keyspace

	var err error
	session, err = cluster.CreateSession()
	if err != nil {
		log.Fatalln(err)
	}

	sendMessageUseCase = use_case.NewSendMessage(
		repository.NewMessage(session),
		service.NewRandInt(),
	)
}

func main() {
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	http.HandleFunc("/ws", wsHandler)

	log.Println("Listening port 8080")
	http.ListenAndServe(":8080", nil)
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
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

			var message domain.Message

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

		for {
			time.Sleep(5 * time.Second)

			if err := conn.WriteMessage(1, writeMessage); err != nil {
				log.Printf("failed to write message: %s", err.Error())
				break
			}
		}
	}()

	wg.Wait()
}
