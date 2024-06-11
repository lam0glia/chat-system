package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gocql/gocql"
	"github.com/gorilla/websocket"
	"github.com/lam0glia/chat-system/domain"
	"github.com/lam0glia/chat-system/repository"
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
	)
}

func main() {
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("failed to oppen websocket connection: %s", err.Error())
			return
		}

		defer conn.Close()

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
	})

	log.Println("Listening port 8080")
	http.ListenAndServe(":8080", nil)
}
