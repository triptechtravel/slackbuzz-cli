package api

import (
	"net/http"
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter()
	if rl == nil {
		t.Fatal("NewRateLimiter() returned nil")
	}
	if rl.remaining != 100 {
		t.Errorf("initial remaining = %d, want 100", rl.remaining)
	}
	if !rl.resetAt.IsZero() {
		t.Errorf("initial resetAt = %v, want zero time", rl.resetAt)
	}
}

func TestUpdate_429(t *testing.T) {
	rl := NewRateLimiter()

	resp := &http.Response{
		StatusCode: 429,
		Header:     http.Header{"Retry-After": []string{"10"}},
	}
	rl.Update(resp)

	if rl.remaining != 0 {
		t.Errorf("remaining = %d, want 0", rl.remaining)
	}
	if rl.resetAt.IsZero() {
		t.Error("resetAt should be set after 429")
	}
}

func TestUpdate_429_NoHeader(t *testing.T) {
	rl := NewRateLimiter()

	resp := &http.Response{
		StatusCode: 429,
		Header:     http.Header{},
	}
	rl.Update(resp)

	if rl.remaining != 0 {
		t.Errorf("remaining = %d, want 0", rl.remaining)
	}
	// Should default to 5 second backoff
	if time.Until(rl.resetAt) < 3*time.Second {
		t.Error("resetAt should be ~5s in the future for default backoff")
	}
}

func TestUpdate_200(t *testing.T) {
	rl := NewRateLimiter()
	// Simulate previous rate limit
	rl.remaining = 0

	resp := &http.Response{
		StatusCode: 200,
		Header:     http.Header{},
	}
	rl.Update(resp)

	if rl.remaining != 1 {
		t.Errorf("remaining = %d, want 1", rl.remaining)
	}
}

func TestWait_HasRemaining(t *testing.T) {
	rl := NewRateLimiter() // remaining = 100

	start := time.Now()
	rl.Wait()
	elapsed := time.Since(start)

	if elapsed > 100*time.Millisecond {
		t.Errorf("Wait() took %v with remaining > 0, expected near-instant return", elapsed)
	}
}

func TestWait_ZeroRemainingPastReset(t *testing.T) {
	rl := NewRateLimiter()
	rl.remaining = 0
	rl.resetAt = time.Now().Add(-1 * time.Second)

	start := time.Now()
	rl.Wait()
	elapsed := time.Since(start)

	if elapsed > 100*time.Millisecond {
		t.Errorf("Wait() took %v with past reset time, expected near-instant return", elapsed)
	}
}

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		{"429 Too Many Requests", 429, true},
		{"200 OK", 200, false},
		{"500 Internal Server Error", 500, false},
		{"403 Forbidden", 403, false},
		{"401 Unauthorized", 401, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewRateLimiter()
			resp := &http.Response{StatusCode: tt.statusCode}
			got := rl.ShouldRetry(resp)
			if got != tt.want {
				t.Errorf("ShouldRetry(status=%d) = %v, want %v", tt.statusCode, got, tt.want)
			}
		})
	}
}
