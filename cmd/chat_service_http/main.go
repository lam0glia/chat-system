package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
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

		_, p, err := conn.ReadMessage()
		if err != nil {
			log.Printf("failed to read message: %s", err.Error())
			return
		}

		fmt.Println(string(p))

		conn.WriteMessage(1, []byte("I'm just a reply message"))
	})

	log.Println("Listening port 8080")
	http.ListenAndServe(":8080", nil)
}
