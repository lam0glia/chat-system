package bootstrap

import (
	"context"
	"fmt"

	"github.com/gocql/gocql"
	"github.com/redis/go-redis/v9"
)

const keyspace = "chat"

func newCassandra(hosts ...string) (*gocql.Session, error) {
	cluster := gocql.NewCluster(hosts...)

	cluster.Keyspace = keyspace

	return cluster.CreateSession()
}

func newRedis(url string) (*redis.Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}

	client := redis.NewClient(opts)

	if err = client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}

	return client, nil
}
