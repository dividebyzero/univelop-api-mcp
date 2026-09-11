package client

import (
	"context"
	"math"
	"sync"
	"time"
)

// RateLimiter implements a token-bucket rate limiter.
type RateLimiter struct {
	mu     sync.Mutex
	rate   float64
	burst  float64
	tokens float64
	last   time.Time
}

// NewRateLimiter creates a rate limiter allowing rate requests per second.
// Returns nil if rate <= 0 (no limiting).
func NewRateLimiter(rate float64) *RateLimiter {
	if rate <= 0 {
		return nil
	}
	return &RateLimiter{
		rate:   rate,
		burst:  math.Max(rate, 1),
		tokens: math.Max(rate, 1),
		last:   time.Now(),
	}
}

// Wait blocks until a token is available or the context is cancelled.
func (l *RateLimiter) Wait(ctx context.Context) error {
	for {
		l.mu.Lock()
		now := time.Now()
		elapsed := now.Sub(l.last).Seconds()
		l.tokens = math.Min(l.burst, l.tokens+elapsed*l.rate)
		l.last = now

		if l.tokens >= 1 {
			l.tokens--
			l.mu.Unlock()
			return nil
		}

		// Need to wait
		need := 1 - l.tokens
		wait := time.Duration((need / l.rate) * float64(time.Second))
		l.mu.Unlock()

		// Guard against degenerate wait times
		if wait < time.Millisecond {
			wait = time.Millisecond
		}
		if wait > 5*time.Second {
			wait = 5 * time.Second
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
}