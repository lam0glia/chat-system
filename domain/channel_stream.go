package domain

const ChannelExchangePresence = "presence"

type ChannelFactory interface {
	NewChannel() (StreamChannel, error)
}

type StreamChannel interface {
	Publish(exchange, key string, body any) error
	Subscribe(buff WebsocketWriteBuffer) error
	Close()
}
