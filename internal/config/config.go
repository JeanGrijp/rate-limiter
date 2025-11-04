// Package config centraliza o carregamento de configurações da aplicação.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/JeanGrijp/rate-limiter/internal/core/domain"
)

type Config struct {
	Server      ServerConfig
	Storage     StorageConfig
	RateLimiter RateLimiterConfig
}

type ServerConfig struct {
	Port string
}

type StorageConfig struct {
	Type  string
	Redis RedisConfig
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type RateLimiterConfig struct {
	IPRule           domain.RateLimitRule
	DefaultTokenRule domain.RateLimitRule
	TokenRules       map[string]domain.RateLimitRule
}

func Load() (Config, error) {
	_ = godotenv.Load()

	server := ServerConfig{Port: getEnv("SERVER_PORT", "8080")}

	storageType := getEnv("STORAGE_TYPE", "redis")

	redisConfig, err := buildRedisConfig()
	if err != nil {
		return Config{}, err
	}

	rateLimiterConfig, err := buildRateLimiterConfig()
	if err != nil {
		return Config{}, err
	}

	return Config{
		Server: server,
		Storage: StorageConfig{
			Type:  storageType,
			Redis: redisConfig,
		},
		RateLimiter: rateLimiterConfig,
	}, nil
}

func buildRedisConfig() (RedisConfig, error) {
	host := getEnv("REDIS_HOST", "localhost")
	port, err := strconv.Atoi(getEnv("REDIS_PORT", "6379"))
	if err != nil {
		return RedisConfig{}, fmt.Errorf("invalid REDIS_PORT: %w", err)
	}
	db, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		return RedisConfig{}, fmt.Errorf("invalid REDIS_DB: %w", err)
	}

	return RedisConfig{
		Host:     host,
		Port:     port,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       db,
	}, nil
}

func buildRateLimiterConfig() (RateLimiterConfig, error) {
	ipRequests, err := strconv.Atoi(getEnv("RATE_LIMIT_IP_REQUESTS", "10"))
	if err != nil {
		return RateLimiterConfig{}, fmt.Errorf("invalid RATE_LIMIT_IP_REQUESTS: %w", err)
	}
	ipWindowSeconds, err := strconv.Atoi(getEnv("RATE_LIMIT_IP_WINDOW_SECONDS", "1"))
	if err != nil {
		return RateLimiterConfig{}, fmt.Errorf("invalid RATE_LIMIT_IP_WINDOW_SECONDS: %w", err)
	}
	ipBlockMinutes, err := strconv.Atoi(getEnv("RATE_LIMIT_IP_BLOCK_DURATION_MINUTES", "5"))
	if err != nil {
		return RateLimiterConfig{}, fmt.Errorf("invalid RATE_LIMIT_IP_BLOCK_DURATION_MINUTES: %w", err)
	}

	defaultTokenRule, err := buildOptionalTokenRule()
	if err != nil {
		return RateLimiterConfig{}, err
	}

	tokenRules, err := buildTokenOverrides()
	if err != nil {
		return RateLimiterConfig{}, err
	}

	return RateLimiterConfig{
		IPRule: domain.RateLimitRule{
			Requests:      ipRequests,
			Window:        time.Duration(ipWindowSeconds) * time.Second,
			BlockDuration: time.Duration(ipBlockMinutes) * time.Minute,
		},
		DefaultTokenRule: defaultTokenRule,
		TokenRules:       tokenRules,
	}, nil
}

func buildOptionalTokenRule() (domain.RateLimitRule, error) {
	requestsStr := os.Getenv("RATE_LIMIT_TOKEN_DEFAULT_REQUESTS")
	if strings.TrimSpace(requestsStr) == "" {
		return domain.RateLimitRule{}, nil
	}

	requests, err := strconv.Atoi(requestsStr)
	if err != nil {
		return domain.RateLimitRule{}, fmt.Errorf("invalid RATE_LIMIT_TOKEN_DEFAULT_REQUESTS: %w", err)
	}

	windowSeconds, err := strconv.Atoi(getEnv("RATE_LIMIT_TOKEN_DEFAULT_WINDOW_SECONDS", "1"))
	if err != nil {
		return domain.RateLimitRule{}, fmt.Errorf("invalid RATE_LIMIT_TOKEN_DEFAULT_WINDOW_SECONDS: %w", err)
	}

	blockMinutes, err := strconv.Atoi(getEnv("RATE_LIMIT_TOKEN_DEFAULT_BLOCK_DURATION_MINUTES", "5"))
	if err != nil {
		return domain.RateLimitRule{}, fmt.Errorf("invalid RATE_LIMIT_TOKEN_DEFAULT_BLOCK_DURATION_MINUTES: %w", err)
	}

	return domain.RateLimitRule{
		Requests:      requests,
		Window:        time.Duration(windowSeconds) * time.Second,
		BlockDuration: time.Duration(blockMinutes) * time.Minute,
	}, nil
}

func buildTokenOverrides() (map[string]domain.RateLimitRule, error) {
	raw := strings.TrimSpace(os.Getenv("TOKENS"))
	if raw == "" {
		return map[string]domain.RateLimitRule{}, nil
	}

	overrides := make(map[string]domain.RateLimitRule)
	items := strings.Split(raw, ",")

	for _, item := range items {
		parts := strings.Split(strings.TrimSpace(item), ":")
		if len(parts) != 4 {
			return nil, fmt.Errorf("token override must follow TOKEN:REQUESTS:WINDOW_SECONDS:BLOCK_DURATION_MINUTES: %s", item)
		}

		token := strings.TrimSpace(parts[0])
		requests, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid requests for token %s: %w", token, err)
		}
		windowSeconds, err := strconv.Atoi(parts[2])
		if err != nil {
			return nil, fmt.Errorf("invalid window seconds for token %s: %w", token, err)
		}
		blockMinutes, err := strconv.Atoi(parts[3])
		if err != nil {
			return nil, fmt.Errorf("invalid block minutes for token %s: %w", token, err)
		}

		overrides[token] = domain.RateLimitRule{
			Requests:      requests,
			Window:        time.Duration(windowSeconds) * time.Second,
			BlockDuration: time.Duration(blockMinutes) * time.Minute,
		}
	}

	return overrides, nil
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
