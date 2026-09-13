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

// ---------- Email verification (existing) ----------

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

// ---------- Phone OTP verification (new) ----------
//
// Keyed by userID (not phone number) since a user must already be logged
// in to request phone verification as part of the author application flow.
// This also naturally prevents someone from spamming OTP requests against
// an arbitrary phone number they don't own.

func (r *VerificationRepository) SavePhoneOTP(ctx context.Context, userID, code string, ttl time.Duration) error {
	key := fmt.Sprintf("phone_verify:%s", userID)
	return r.redis.Set(ctx, key, code, ttl).Err()
}

func (r *VerificationRepository) GetPhoneOTP(ctx context.Context, userID string) (string, error) {
	key := fmt.Sprintf("phone_verify:%s", userID)
	return r.redis.Get(ctx, key).Result()
}

func (r *VerificationRepository) DeletePhoneOTP(ctx context.Context, userID string) error {
	key := fmt.Sprintf("phone_verify:%s", userID)
	return r.redis.Del(ctx, key).Err()
}

// ---------- Phone OTP resend cooldown ----------
// Prevents hammering the (future) real SMS provider / terminal log.

func (r *VerificationRepository) SetPhoneOTPCooldown(ctx context.Context, userID string, ttl time.Duration) error {
	key := fmt.Sprintf("phone_verify_cooldown:%s", userID)
	return r.redis.Set(ctx, key, "1", ttl).Err()
}

func (r *VerificationRepository) IsPhoneOTPOnCooldown(ctx context.Context, userID string) (bool, error) {
	key := fmt.Sprintf("phone_verify_cooldown:%s", userID)
	n, err := r.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}