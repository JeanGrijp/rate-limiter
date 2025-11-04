// Package ports define contratos que conectam o domínio a implementações externas.
package ports

import (
	"context"
	"time"
)

type Storage interface {
	Increment(ctx context.Context, key string, window time.Duration) (int64, error)
	IsBlocked(ctx context.Context, key string) (bool, error)
	SetBlock(ctx context.Context, key string, duration time.Duration) error
}
