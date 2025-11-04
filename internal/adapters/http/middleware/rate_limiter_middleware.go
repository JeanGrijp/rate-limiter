// Package middleware disponibiliza middlewares HTTP específicos da aplicação.
package middleware

import (
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/JeanGrijp/rate-limiter/internal/core/domain"
	"github.com/JeanGrijp/rate-limiter/internal/core/ports"
)

const rateLimitExceededMessage = "you have reached the maximum number of requests or actions allowed within a certain time frame"

func NewRateLimiterMiddleware(limiter ports.RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limiter == nil {
				next.ServeHTTP(w, r)
				return
			}

			ip := extractIP(r)
			token := strings.TrimSpace(r.Header.Get("API_KEY"))

			decision, err := limiter.Allow(r.Context(), domain.RateLimitRequest{IP: ip, Token: token})
			if err != nil {
				if domain.IsBlockedError(err) {
					writeTooManyRequests(w)
					return
				}

				log.Printf("rate limiter failed: %v", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			if !decision.Allowed {
				writeTooManyRequests(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func extractIP(r *http.Request) string {
	xForwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if xForwardedFor != "" {
		parts := strings.Split(xForwardedFor, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	xRealIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if xRealIP != "" {
		return xRealIP
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}

	return host
}

func writeTooManyRequests(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusTooManyRequests)
	_, _ = w.Write([]byte(rateLimitExceededMessage))
}
