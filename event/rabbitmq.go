package event

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/lam0glia/chat-system/domain"
	"github.com/lam0glia/chat-system/internal"
	amqp "github.com/rabbitmq/amqp091-go"
)

type rabbitMQ struct {
	connection *amqp.Connection
}

type rabbitMQChannel struct {
	channel   *amqp.Channel
	queueName string
}

func (c *rabbitMQChannel) Close() {
	c.channel.Close()
}

func (r *rabbitMQ) NewChannel() (domain.StreamChannel, error) {
	ch, err := r.connection.Channel()
	if err != nil {
		return nil, err
	}

	err = ch.ExchangeDeclare(
		"presence", // nome do exchange
		"fanout",   // tipo: fanout
		true,       // durable
		false,      // auto-delete
		false,      // internal
		false,      // no-wait
		nil,        // args
	)
	if err != nil {
		log.Printf("deu ruim pra criar o exchange: %s", err.Error())
		return nil, err
	}

	q, err := ch.QueueDeclare(
		"",    // nome vazio cria uma fila exclusiva e aleatória
		false, // durable
		true,  // auto-delete
		true,  // exclusive
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		log.Printf("deu ruim pra criar a fila: %s", err.Error())
		return nil, err
	}

	err = ch.QueueBind(
		q.Name,     // nome da fila
		"",         // routing key (ignorada no fanout)
		"presence", // nome do exchange
		false,
		nil,
	)
	if err != nil {
		log.Printf("deu ruim pra criar o bind: %s", err.Error())
		return nil, err
	}

	return &rabbitMQChannel{
		channel:   ch,
		queueName: q.Name,
	}, nil
}

func (c *rabbitMQChannel) Publish(exchange, key string, body any) error {
	encodedBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("json encode body: %w", err)
	}

	err = c.channel.Publish(
		exchange, // exchange
		key,      // routing key
		false,    // mandatory
		false,    // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        encodedBody,
		})
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}

func (c *rabbitMQChannel) Subscribe(buff domain.WebsocketWriteBuffer) error {
	msgs, err := c.channel.Consume(
		c.queueName, // queue
		"",          // consumer
		false,       // auto-ack
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)
	if err != nil {
		return err
	}

	defer internal.LogGoroutineClosed("RabbitMQChannel.Subscribe")

	for d := range msgs {
		var msg any

		if err = json.Unmarshal(d.Body, &msg); err != nil {
			log.Printf("err: json decode: %s", err)

			d.Reject(false)

			continue
		}

		buff.Write(msg)

		if err = d.Ack(false); err != nil {
			log.Printf("err: ack: %s", err)
			continue
		}
	}

	return nil
}

func NewRabbitMQ(conn *amqp.Connection) *rabbitMQ {
	return &rabbitMQ{
		connection: conn,
	}
}
