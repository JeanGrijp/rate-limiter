package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	httpHandlers "github.com/JeanGrijp/rate-limiter/internal/adapters/http/handlers"
	httpMiddleware "github.com/JeanGrijp/rate-limiter/internal/adapters/http/middleware"
	redisstorage "github.com/JeanGrijp/rate-limiter/internal/adapters/storage/redis"
	"github.com/JeanGrijp/rate-limiter/internal/config"
	"github.com/JeanGrijp/rate-limiter/internal/core/domain"
	"github.com/JeanGrijp/rate-limiter/internal/core/ports"
	"github.com/JeanGrijp/rate-limiter/internal/core/services"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	storage, closeFn, err := initStorage(cfg.Storage)
	if err != nil {
		log.Fatalf("failed to init storage: %v", err)
	}
	defer closeFn()

	limiter, err := services.NewRateLimiterService(storage, services.Config{
		DefaultIPRule:    cfg.RateLimiter.IPRule,
		DefaultTokenRule: cfg.RateLimiter.DefaultTokenRule,
		TokenRules:       cloneRules(cfg.RateLimiter.TokenRules),
	})
	if err != nil {
		log.Fatalf("failed to create limiter: %v", err)
	}

	r := chi.NewRouter()
	r.Use(httpMiddleware.NewRateLimiterMiddleware(limiter))
	r.Get("/test", httpHandlers.TestHandler)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Server.Port),
		Handler: r,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("shutdown signal received")
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

func initStorage(cfg config.StorageConfig) (ports.Storage, func(), error) {
	switch cfg.Type {
	case "redis":
		redisCfg := redisstorage.Config{
			Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		}
		storage, err := redisstorage.New(redisCfg)
		if err != nil {
			return nil, nil, err
		}
		return storage, func() {
			if err := storage.Close(); err != nil {
				log.Printf("failed to close redis storage: %v", err)
			}
		}, nil
	default:
		return nil, nil, fmt.Errorf("unsupported storage type: %s", cfg.Type)
	}
}

func cloneRules(src map[string]domain.RateLimitRule) map[string]domain.RateLimitRule {
	if src == nil {
		return nil
	}
	clone := make(map[string]domain.RateLimitRule, len(src))
	for k, v := range src {
		clone[k] = v
	}
	return clone
}
