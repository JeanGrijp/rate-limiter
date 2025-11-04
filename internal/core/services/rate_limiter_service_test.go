package services

import (
	"context"
	"testing"
	"time"

	"github.com/JeanGrijp/rate-limiter/internal/core/domain"
)

func TestRateLimiter_AllowsWithinIPLimit(t *testing.T) {
	storage := newMockStorage()
	service := newTestLimiter(t, storage, Config{
		DefaultIPRule: domain.RateLimitRule{
			Requests:      3,
			Window:        time.Second,
			BlockDuration: time.Minute,
		},
	})

	ctx := context.Background()

	for i := 0; i < 3; i++ {
		decision, err := service.Allow(ctx, domain.RateLimitRequest{IP: "192.168.1.1"})
		if err != nil {
			t.Fatalf("unexpected error at attempt %d: %v", i+1, err)
		}
		if !decision.Allowed {
			t.Fatalf("expected request %d to be allowed", i+1)
		}
	}
}

func TestRateLimiter_BlocksAfterExceedingIPLimit(t *testing.T) {
	storage := newMockStorage()
	service := newTestLimiter(t, storage, Config{
		DefaultIPRule: domain.RateLimitRule{
			Requests:      2,
			Window:        time.Second,
			BlockDuration: time.Minute,
		},
	})

	ctx := context.Background()

	for i := 0; i < 2; i++ {
		if _, err := service.Allow(ctx, domain.RateLimitRequest{IP: "10.0.0.1"}); err != nil {
			t.Fatalf("unexpected error on warmup %d: %v", i+1, err)
		}
	}

	decision, err := service.Allow(ctx, domain.RateLimitRequest{IP: "10.0.0.1"})
	if err == nil || !domain.IsBlockedError(err) {
		t.Fatalf("expected blocked error, got decision=%+v err=%v", decision, err)
	}
	if decision.Allowed {
		t.Fatalf("expected decision.Allowed=false after exceeding limit")
	}

	// Once blocked, the next call should be short-circuited by IsBlocked.
	_, err = service.Allow(ctx, domain.RateLimitRequest{IP: "10.0.0.1"})
	if err == nil || !domain.IsBlockedError(err) {
		t.Fatalf("expected blocked error on subsequent call, got %v", err)
	}
}

func TestRateLimiter_UsesTokenOverride(t *testing.T) {
	storage := newMockStorage()
	tokenRule := domain.RateLimitRule{
		Requests:      5,
		Window:        time.Second,
		BlockDuration: time.Minute,
	}

	service := newTestLimiter(t, storage, Config{
		DefaultIPRule: domain.RateLimitRule{
			Requests:      1,
			Window:        time.Second,
			BlockDuration: time.Minute,
		},
		TokenRules: map[string]domain.RateLimitRule{
			"abc123": tokenRule,
		},
	})

	ctx := context.Background()

	for i := 0; i < tokenRule.Requests; i++ {
		decision, err := service.Allow(ctx, domain.RateLimitRequest{IP: "203.0.113.10", Token: "abc123"})
		if err != nil {
			t.Fatalf("unexpected error for token request %d: %v", i+1, err)
		}
		if !decision.Allowed {
			t.Fatalf("expected token request %d to be allowed", i+1)
		}
		if decision.AppliedRule != tokenRule {
			t.Fatalf("expected token rule to be applied, got %+v", decision.AppliedRule)
		}
	}
}

func TestRateLimiter_DefaultTokenRule(t *testing.T) {
	storage := newMockStorage()
	defaultTokenRule := domain.RateLimitRule{
		Requests:      2,
		Window:        time.Second,
		BlockDuration: time.Minute,
	}

	service := newTestLimiter(t, storage, Config{
		DefaultIPRule: domain.RateLimitRule{
			Requests:      1,
			Window:        time.Second,
			BlockDuration: time.Minute,
		},
		DefaultTokenRule: defaultTokenRule,
	})

	ctx := context.Background()

	// First request should be allowed under the default token rule.
	if decision, err := service.Allow(ctx, domain.RateLimitRequest{IP: "198.51.100.5", Token: "dynamic"}); err != nil || !decision.Allowed {
		t.Fatalf("expected first token request to be allowed, decision=%+v err=%v", decision, err)
	}

	// Second request still allowed.
	if decision, err := service.Allow(ctx, domain.RateLimitRequest{IP: "198.51.100.5", Token: "dynamic"}); err != nil || !decision.Allowed {
		t.Fatalf("expected second token request to be allowed, decision=%+v err=%v", decision, err)
	}

	// Third request should trigger a block under the token rule.
	if _, err := service.Allow(ctx, domain.RateLimitRequest{IP: "198.51.100.5", Token: "dynamic"}); err == nil || !domain.IsBlockedError(err) {
		t.Fatalf("expected blocked error on third token request, got %v", err)
	}
}

// newTestLimiter is a helper that fails the test immediately if creation fails.
func newTestLimiter(t *testing.T, storage *mockStorage, cfg Config) *RateLimiterService {
	t.Helper()
	service, err := NewRateLimiterService(storage, cfg)
	if err != nil {
		t.Fatalf("failed to create rate limiter service: %v", err)
	}
	return service
}

type mockStorage struct {
	counts map[string]int64
	blocks map[string]time.Time
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		counts: make(map[string]int64),
		blocks: make(map[string]time.Time),
	}
}

func (m *mockStorage) Increment(_ context.Context, key string, _ time.Duration) (int64, error) {
	m.counts[key]++
	return m.counts[key], nil
}

func (m *mockStorage) IsBlocked(_ context.Context, key string) (bool, error) {
	expiration, ok := m.blocks[key]
	if !ok {
		return false, nil
	}
	if time.Now().After(expiration) {
		delete(m.blocks, key)
		return false, nil
	}
	return true, nil
}

func (m *mockStorage) SetBlock(_ context.Context, key string, duration time.Duration) error {
	if duration <= 0 {
		delete(m.blocks, key)
		return nil
	}
	m.blocks[key] = time.Now().Add(duration)
	return nil
}
