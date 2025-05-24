package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type presence struct {
	db *redis.Client
}

const onlineStatus = "online"
const onlinePresenceDuration = 40 * time.Second

func (r *presence) SetKeyExpiration(ctx context.Context, userID uint64) error {
	key := r.getKey(userID)

	return r.db.Expire(ctx, key, onlinePresenceDuration).Err()
}

func (r *presence) UpdateUserStatus(ctx context.Context, userID uint64, status string) error {
	key := r.getKey(userID)

	return r.db.Set(
		ctx,
		key,
		onlineStatus,
		onlinePresenceDuration).Err()
}

func (r *presence) DeleteUserStatus(ctx context.Context, userID uint64) error {
	return r.db.Del(ctx, r.getKey(userID)).Err()
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
