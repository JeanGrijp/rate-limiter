package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/JeanGrijp/rate-limiter/internal/core/domain"
	"github.com/JeanGrijp/rate-limiter/internal/core/ports"
)

// Config agrega os limites utilizados pelo serviço de rate limiting.
type Config struct {
	DefaultIPRule    domain.RateLimitRule
	DefaultTokenRule domain.RateLimitRule
	TokenRules       map[string]domain.RateLimitRule
}

// RateLimiterService implementa a lógica central de rate limiting.
type RateLimiterService struct {
	storage ports.Storage
	config  Config
}

// NewRateLimiterService cria uma nova instância do serviço.
func NewRateLimiterService(storage ports.Storage, cfg Config) (*RateLimiterService, error) {
	if storage == nil {
		return nil, fmt.Errorf("storage is required")
	}
	if cfg.DefaultIPRule.Requests <= 0 || cfg.DefaultIPRule.Window <= 0 {
		return nil, fmt.Errorf("default IP rule must have positive values")
	}
	if cfg.TokenRules == nil {
		cfg.TokenRules = make(map[string]domain.RateLimitRule)
	}

	return &RateLimiterService{storage: storage, config: cfg}, nil
}

// Allow avalia se a requisição pode prosseguir de acordo com as regras configuradas.
func (s *RateLimiterService) Allow(ctx context.Context, req domain.RateLimitRequest) (domain.Decision, error) {
	rule, keys, err := s.resolveRule(req)
	if err != nil {
		return domain.Decision{}, err
	}

	blocked, err := s.storage.IsBlocked(ctx, keys.blockKey)
	if err != nil {
		return domain.Decision{}, err
	}
	if blocked {
		return domain.Decision{Allowed: false, Identifier: keys.identifier, AppliedRule: rule}, domain.ErrBlocked
	}

	currentCount, err := s.storage.Increment(ctx, keys.counterKey, rule.Window)
	if err != nil {
		return domain.Decision{}, err
	}

	if int(currentCount) > rule.Requests {
		if setErr := s.storage.SetBlock(ctx, keys.blockKey, rule.BlockDuration); setErr != nil {
			return domain.Decision{}, setErr
		}
		return domain.Decision{Allowed: false, Identifier: keys.identifier, AppliedRule: rule, CurrentCount: currentCount}, domain.ErrBlocked
	}

	return domain.Decision{Allowed: true, Identifier: keys.identifier, AppliedRule: rule, CurrentCount: currentCount}, nil
}

type resolvedKeys struct {
	counterKey string
	blockKey   string
	identifier string
}

func (s *RateLimiterService) resolveRule(req domain.RateLimitRequest) (domain.RateLimitRule, resolvedKeys, error) {
	token := strings.TrimSpace(req.Token)
	if token != "" {
		if rule, ok := s.config.TokenRules[token]; ok {
			return rule, buildKeys("token", token), nil
		}
		if s.config.DefaultTokenRule.Requests > 0 && s.config.DefaultTokenRule.Window > 0 {
			return s.config.DefaultTokenRule, buildKeys("token", token), nil
		}
	}

	ip := strings.TrimSpace(req.IP)
	if ip == "" {
		return domain.RateLimitRule{}, resolvedKeys{}, fmt.Errorf("ip address is required when token has no override")
	}

	return s.config.DefaultIPRule, buildKeys("ip", ip), nil
}

func buildKeys(prefix, identifier string) resolvedKeys {
	identifier = strings.ToLower(strings.TrimSpace(identifier))
	return resolvedKeys{
		counterKey: fmt.Sprintf("ratelimit:%s:%s", prefix, identifier),
		blockKey:   fmt.Sprintf("ratelimit:%s:%s:block", prefix, identifier),
		identifier: identifier,
	}
}
