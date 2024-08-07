package bootstrap

import (
	"fmt"

	"github.com/gocql/gocql"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"github.com/sony/sonyflake"
)

type App struct {
	Env                *Env
	CassandraSession   *gocql.Session
	RabbitMQConnection *amqp.Connection
	RedisClient        *redis.Client
	SonyFlake          *sonyflake.Sonyflake
}

func NewApp() (*App, error) {
	var (
		err error
		app App
	)

	app.Env, err = newEnv()
	if err != nil {
		return nil, fmt.Errorf("load env: %w", err)
	}

	app.CassandraSession, err = newCassandra(app.Env.CassandraHosts...)
	if err != nil {
		return nil, fmt.Errorf("create cassandra session: %w", err)
	}

	app.RabbitMQConnection, err = amqp.Dial(app.Env.RabbitMQURL)
	if err != nil {
		return nil, fmt.Errorf("create rabbitmq connection: %w", err)
	}

	app.RedisClient, err = newRedis(app.Env.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("create redis connection: %w", err)
	}

	app.SonyFlake, err = newUIDGenerator(app.Env.UIDGeneratorStartTime)
	if err != nil {
		return nil, fmt.Errorf("new uid generator: %w", err)
	}

	return &app, nil
}
