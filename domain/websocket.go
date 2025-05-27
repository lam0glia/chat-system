package domain

type WebsocketConnection interface {
	WriteJSON(any) error
}

type WebsocketWriteBuffer interface {
	Setup(WebsocketConnection)
	DeliveryToClient()
	Write(body any)
}
