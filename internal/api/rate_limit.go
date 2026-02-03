package api

import (
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RateLimiter tracks Slack API rate limits and provides backoff.
type RateLimiter struct {
	mu        sync.Mutex
	remaining int
	resetAt   time.Time
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{remaining: 100}
}

// Update reads rate limit headers from a response.
func (rl *RateLimiter) Update(resp *http.Response) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Slack uses Retry-After header (seconds) on 429 responses
	if resp.StatusCode == 429 {
		if v := resp.Header.Get("Retry-After"); v != "" {
			if secs, err := strconv.Atoi(v); err == nil {
				rl.remaining = 0
				rl.resetAt = time.Now().Add(time.Duration(secs) * time.Second)
				return
			}
		}
		// Default backoff if no Retry-After
		rl.remaining = 0
		rl.resetAt = time.Now().Add(5 * time.Second)
		return
	}

	// Not rate limited; assume we have capacity
	rl.remaining = 1
}

// Wait blocks until it's safe to make another request.
func (rl *RateLimiter) Wait() {
	rl.mu.Lock()
	remaining := rl.remaining
	resetAt := rl.resetAt
	rl.mu.Unlock()

	if remaining > 0 {
		return
	}

	waitDuration := time.Until(resetAt)
	if waitDuration > 0 {
		time.Sleep(waitDuration)
	}
}

// ShouldRetry returns true if the response indicates rate limiting.
func (rl *RateLimiter) ShouldRetry(resp *http.Response) bool {
	return resp.StatusCode == 429
}
