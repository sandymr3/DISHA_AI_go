package ai

import (
	"context"
	"sync"
	"time"
)

// tokenBucket implements a simple token bucket for rate limiting.
type tokenBucket struct {
	tokens     float64
	capacity   float64
	refillRate float64 // tokens per second
	lastRefill time.Time
	lastAccess time.Time
}

// RateLimiter provides per-student rate limiting using token buckets.
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*tokenBucket
	capacity int
	windowH  int
}

// NewRateLimiter creates a new rate limiter. The cleanup goroutine runs until ctx is cancelled.
func NewRateLimiter(ctx context.Context, capacity int, windowHours int) *RateLimiter {
	rl := &RateLimiter{
		buckets:  make(map[string]*tokenBucket),
		capacity: capacity,
		windowH:  windowHours,
	}
	go rl.cleanupLoop(ctx)
	return rl
}

// Allow checks if a request from studentID is permitted.
// Returns (allowed, retryAfter). retryAfter is 0 if allowed.
func (rl *RateLimiter) Allow(studentID string) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.buckets[studentID]
	if !ok {
		b = &tokenBucket{
			tokens:     float64(rl.capacity) - 1, // consume one token
			capacity:   float64(rl.capacity),
			refillRate: float64(rl.capacity) / (float64(rl.windowH) * 3600.0),
			lastRefill: now,
			lastAccess: now,
		}
		rl.buckets[studentID] = b
		return true, 0
	}

	// Refill tokens based on elapsed time
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * b.refillRate
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
	b.lastRefill = now
	b.lastAccess = now

	if b.tokens < 1 {
		// Calculate retry-after: time to refill 1 token
		deficit := 1.0 - b.tokens
		retryAfter := time.Duration(deficit/b.refillRate) * time.Second
		return false, retryAfter
	}

	b.tokens--
	return true, 0
}

// cleanupLoop removes stale buckets (last access > 2 hours).
func (rl *RateLimiter) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rl.cleanup()
		}
	}
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	staleThreshold := 2 * time.Hour
	for id, b := range rl.buckets {
		if time.Since(b.lastAccess) > staleThreshold {
			delete(rl.buckets, id)
		}
	}
}
