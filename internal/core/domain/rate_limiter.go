// Package domain concentra entidades e estruturas centrais do rate limiter.
package domain

import "time"

type RateLimitRule struct {
	Requests      int
	Window        time.Duration
	BlockDuration time.Duration
}

type RateLimitRequest struct {
	IP    string
	Token string
}

type Decision struct {
	Allowed      bool
	Identifier   string
	AppliedRule  RateLimitRule
	CurrentCount int64
}
