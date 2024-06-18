package handler

import (
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/lam0glia/chat-system/domain"
)

type Presence struct {
	upgrader         websocket.Upgrader
	updatePresence   domain.UpdatePresenceUseCase
	presenceWriter   domain.PresenceWriter
	presenceConsumer domain.PresenceConsumer
}

func (h *Presence) WebSocket(c *gin.Context) {
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
		if err = h.updatePresence.Execute(ctx, &presence); err != nil {
			log.Println("err: update presence")
		}
	}()

	if err = h.updatePresence.Execute(ctx, &presence); err != nil {
		abortWithInternalError(c, err)
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		abortWithInternalError(c, err)
		return
	}

	defer conn.Close()

	ticker := time.NewTicker(5 * time.Second)

	conn.SetPongHandler(func(appData string) error {
		log.Println("Pong received")

		var err error
		if err = h.presenceWriter.Update(ctx, from, true); err != nil {
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

	go h.presenceConsumer.Consume(ctx, from, conn, stop)

	wg.Wait()
}

func NewPresence(
	updatePresence domain.UpdatePresenceUseCase,
	presenceWriter domain.PresenceWriter,
	presenceConsumer domain.PresenceConsumer,
) *Presence {
	return &Presence{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  128,
			WriteBufferSize: 128,
		},
		updatePresence:   updatePresence,
		presenceWriter:   presenceWriter,
		presenceConsumer: presenceConsumer,
	}
}
