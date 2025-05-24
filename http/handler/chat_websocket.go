package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/lam0glia/chat-system/domain"
	"github.com/lam0glia/chat-system/stream"
)

const (
	pingTickerDuration   = 5 * time.Second
	pongDeadlineDuration = 30 * time.Second
)

type chatWS struct {
	conn               *websocket.Conn
	userID             uint64
	sendMessageUseCase domain.SendMessageUseCase
	pingTicker         *time.Ticker
	consumer           *stream.Chat
	done               chan bool
	presenceService    domain.PresenceService
}

func newChatWS(
	c *gin.Context,
	upgrader websocket.Upgrader,
	userID uint64,
	sendMessageUseCase domain.SendMessageUseCase,
	consumer *stream.Chat,
	presenceService domain.PresenceService,
) (*chatWS, error) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return nil, fmt.Errorf("upgrade http connection: %w", err)
	}

	ticker := time.NewTicker(pingTickerDuration)

	conn.SetPongHandler(func(string) error {
		now := time.Now()

		// log.Printf("Pong received at: %s", time.Now().String())

		if err := presenceService.RefreshUserPresence(c.Request.Context(), userID); err != nil {
			log.Printf("err: refresh presence: %s", err.Error())
			return err
		}

		conn.SetReadDeadline(now.Add(pongDeadlineDuration))
		ticker.Reset(pingTickerDuration)

		return nil
	})

	return &chatWS{
		conn:               conn,
		userID:             userID,
		sendMessageUseCase: sendMessageUseCase,
		pingTicker:         ticker,
		consumer:           consumer,
		done:               make(chan bool),
		presenceService:    presenceService,
	}, nil
}

func (ws *chatWS) readFromClient(ctx context.Context) {
	defer func() {
		logGoroutineDone("ReadFromClient")
		ws.done <- true
	}()

	for {
		_, r, err := ws.conn.NextReader()

		if err != nil {
			if closeErr, is := err.(*websocket.CloseError); is {
				log.Printf("Close message received: [%d] %s", closeErr.Code, closeErr.Text)
			} else {
				log.Printf("err: read peer: %s", err)
			}

			break
		}

		message := domain.SendMessageRequest{
			From: ws.userID,
		}

		if err := json.NewDecoder(r).Decode(&message); err != nil {
			log.Printf("err: decode json: %s", err)
		} else if err = ws.sendMessageUseCase.Execute(ctx, &message); err != nil {
			log.Printf("err: send message: %s", err)
		}
	}
}

func (ws *chatWS) writeToClient(ctx context.Context) {
	defer func() {
		logGoroutineDone("WriteToClient")
		ws.consumer.Close()
	}()

	go func() {
		defer logGoroutineDone("ConsumeMessages")

		err := ws.consumer.ConsumeMessages(ws.conn)
		if err != nil {
			log.Printf("err: consume messages: %s", err)
		}
	}()

	<-ctx.Done()
}

func (ws *chatWS) ping(ctx context.Context) {
	defer logGoroutineDone("Ping")

	go func() {
		// Reset ticker to break the ping loop
		defer ws.pingTicker.Reset(1)

		<-ctx.Done()
	}()

	for {
		<-ws.pingTicker.C
		err := ws.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
		ws.pingTicker.Reset(30 * time.Second)
		if err != nil {
			if err == websocket.ErrCloseSent {
				ws.pingTicker.Stop()
				break
			}

			log.Printf("err: send ping message: %s", err)
			ws.pingTicker.Reset(pingTickerDuration)
		}
		// else {
		// 	log.Printf("Ping sent at: %s", tickTime.String())
		// }
	}
}

func (ws *chatWS) writePresenceUpdates() {
	defer logGoroutineDone("SubscribeUserPresenceUpdate")

	var err error
	if err = ws.presenceService.SubscribeUserPresenceUpdate(ws.conn); err != nil {
		log.Printf("err: subscribe presence: %s", err.Error())
	}
}

func logGoroutineDone(name string) {
	log.Printf("%s goroutine done", name)
}

func (ws *chatWS) close() {
	ws.conn.Close()
}
