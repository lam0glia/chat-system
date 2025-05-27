package websocket_buffer

import (
	"log"

	"github.com/lam0glia/chat-system/domain"
	"github.com/lam0glia/chat-system/internal"
)

type WriteBuffer struct {
	conn   domain.WebsocketConnection
	buffer chan any
}

func (b *WriteBuffer) DeliveryToClient() {
	var err error

	for body := range b.buffer {
		if err = b.conn.WriteJSON(body); err != nil {
			log.Printf("err: write json: %s", err)
		}
	}

	internal.LogGoroutineClosed("WriteBuffer.DeliveryToClient")
}

func (b *WriteBuffer) Write(body any) {
	b.buffer <- body
}

func (b *WriteBuffer) Setup(conn domain.WebsocketConnection) {
	b.buffer = make(chan any)
	b.conn = conn
}
