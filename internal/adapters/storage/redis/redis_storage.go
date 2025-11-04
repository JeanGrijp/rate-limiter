// Package redis disponibiliza a implementação do storage baseada em Redis.
package redis

import (
	"context"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"

	"github.com/JeanGrijp/rate-limiter/internal/core/ports"
)

type Storage struct {
	client *redis.Client
}

var _ ports.Storage = (*Storage)(nil)

type Config struct {
	Addr     string
	Password string
	DB       int
}

func New(cfg Config) (*Storage, error) {
	if cfg.Addr == "" {
		return nil, fmt.Errorf("redis address is required")
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &Storage{client: client}, nil
}

func (s *Storage) Close() error {
	return s.client.Close()
}

func (s *Storage) Increment(ctx context.Context, key string, window time.Duration) (int64, error) {
	pipe := s.client.TxPipeline()
	counter := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return counter.Val(), nil
}

func (s *Storage) IsBlocked(ctx context.Context, key string) (bool, error) {
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func (s *Storage) SetBlock(ctx context.Context, key string, duration time.Duration) error {
	if duration <= 0 {
		return s.client.Del(ctx, key).Err()
	}
	return s.client.Set(ctx, key, "1", duration).Err()
}
