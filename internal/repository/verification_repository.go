package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type VerificationRepository struct {
	redis *redis.Client
}

func NewVerificationRepository(redis *redis.Client) *VerificationRepository {
	return &VerificationRepository{redis: redis}
}

func (r *VerificationRepository) SaveVerificationCode(ctx context.Context, email, code string, ttl time.Duration) error {
	key := fmt.Sprintf("verify:%s", email)
	return r.redis.Set(ctx, key, code, ttl).Err()
}

func (r *VerificationRepository) GetVerificationCode(ctx context.Context, email string) (string, error) {
	key := fmt.Sprintf("verify:%s", email)
	return r.redis.Get(ctx, key).Result()
}

func (r *VerificationRepository) DeleteVerificationCode(ctx context.Context, email string) error {
	key := fmt.Sprintf("verify:%s", email)
	return r.redis.Del(ctx, key).Err()
}