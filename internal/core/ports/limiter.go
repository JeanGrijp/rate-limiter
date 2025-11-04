// Package ports define contratos que conectam o domínio a implementações externas.
package ports

import (
	"context"

	"github.com/JeanGrijp/rate-limiter/internal/core/domain"
)

type RateLimiter interface {
	Allow(ctx context.Context, req domain.RateLimitRequest) (domain.Decision, error)
}
