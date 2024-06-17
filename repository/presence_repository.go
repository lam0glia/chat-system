package repository

import (
	"context"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

type presence struct {
	db *redis.Client
}

const onlineStatus = "online"

func (r *presence) Update(ctx context.Context, userID uint64, online bool) (err error) {
	key := fmt.Sprintf("%d", userID)
	if online {
		err = r.db.Set(ctx, key, onlineStatus, 0).Err()
	} else {
		err = r.db.Del(ctx, key).Err()
	}

	return err
}

func (r *presence) IsOnline(ctx context.Context, userID uint64) (bool, error) {
	value, err := r.db.Get(ctx, r.getKey(userID)).Result()
	if err != nil {
		if err == redis.Nil {
			err = nil
		}

		return false, err
	}

	return value == onlineStatus, nil
}

// como não tem uma relação de quais usuários devem receber a atualização
// de presença, recupero todos que estão online
func (r *presence) GetInterestedUsers(ctx context.Context, userID uint64) ([]uint64, error) {
	keys, err := r.db.Keys(ctx, "*").Result()
	if err != nil {
		return nil, err
	}

	results := make([]uint64, len(keys)-1)

	for _, key := range keys {
		id, _ := strconv.ParseUint(key, 10, 64)

		if userID != id {
			results = append(results, id)
		}
	}

	return results, nil
}

func (r *presence) getKey(userID uint64) string {
	return fmt.Sprintf("%d", userID)
}

func NewPresence(client *redis.Client) *presence {
	return &presence{
		db: client,
	}
}
