package domain

type WebsocketConnection interface {
	WriteJSON(any) error
}
