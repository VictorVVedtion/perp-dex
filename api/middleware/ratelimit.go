package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	// Configuration
	config *RateLimitConfig

	// Buckets by key (IP or user ID)
	buckets   map[string]*Bucket
	bucketsMu sync.RWMutex

	// Order submission limits (stricter)
	orderBuckets   map[string]*Bucket
	orderBucketsMu sync.RWMutex

	// Daily counters
	dailyCounters   map[string]*DailyCounter
	dailyCountersMu sync.RWMutex

	// Cleanup ticker
	cleanupTicker *time.Ticker
	stopCh        chan struct{}
}

// RateLimitConfig contains rate limiting configuration
type RateLimitConfig struct {
	// IP-based limits
	IPRequestsPerSecond    int           // General requests per second per IP
	IPBurst                int           // Burst capacity for IP
	IPBlockDuration        time.Duration // How long to block after limit exceeded

	// User-based limits (stricter, identified users)
	UserRequestsPerSecond  int           // General requests per second per user
	UserBurst              int           // Burst capacity for user

	// Order-specific limits
	OrdersPerSecond        int           // Order submissions per second
	OrdersPerDay           int           // Orders per day per user
	OrderBurst             int           // Burst for orders

	// Cleanup
	CleanupInterval        time.Duration // How often to clean up old buckets
	BucketTTL              time.Duration // Time before unused bucket is removed
}

// DefaultRateLimitConfig returns default configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		IPRequestsPerSecond:   100,
		IPBurst:               200,
		IPBlockDuration:       time.Minute,

		UserRequestsPerSecond: 200,
		UserBurst:             400,

		OrdersPerSecond:       10,
		OrdersPerDay:          10000,
		OrderBurst:            20,

		CleanupInterval:       time.Minute * 5,
		BucketTTL:             time.Hour,
	}
}

// Bucket represents a token bucket for rate limiting
type Bucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64   // tokens per second
	lastUpdate time.Time
	blocked    bool
	blockedUntil time.Time
	mu         sync.Mutex
}

// DailyCounter tracks daily request counts
type DailyCounter struct {
	count     int
	limit     int
	date      string
	mu        sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *RateLimitConfig) *RateLimiter {
	if config == nil {
		config = DefaultRateLimitConfig()
	}

	rl := &RateLimiter{
		config:        config,
		buckets:       make(map[string]*Bucket),
		orderBuckets:  make(map[string]*Bucket),
		dailyCounters: make(map[string]*DailyCounter),
		cleanupTicker: time.NewTicker(config.CleanupInterval),
		stopCh:        make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// Stop stops the rate limiter
func (rl *RateLimiter) Stop() {
	close(rl.stopCh)
	rl.cleanupTicker.Stop()
}

// cleanupLoop periodically cleans up expired buckets
func (rl *RateLimiter) cleanupLoop() {
	for {
		select {
		case <-rl.cleanupTicker.C:
			rl.cleanup()
		case <-rl.stopCh:
			return
		}
	}
}

// cleanup removes expired buckets
func (rl *RateLimiter) cleanup() {
	now := time.Now()
	threshold := now.Add(-rl.config.BucketTTL)

	rl.bucketsMu.Lock()
	for key, bucket := range rl.buckets {
		bucket.mu.Lock()
		if bucket.lastUpdate.Before(threshold) {
			delete(rl.buckets, key)
		}
		bucket.mu.Unlock()
	}
	rl.bucketsMu.Unlock()

	rl.orderBucketsMu.Lock()
	for key, bucket := range rl.orderBuckets {
		bucket.mu.Lock()
		if bucket.lastUpdate.Before(threshold) {
			delete(rl.orderBuckets, key)
		}
		bucket.mu.Unlock()
	}
	rl.orderBucketsMu.Unlock()
}

// getBucket gets or creates a bucket for a key
func (rl *RateLimiter) getBucket(key string, maxTokens, refillRate float64) *Bucket {
	rl.bucketsMu.RLock()
	bucket, ok := rl.buckets[key]
	rl.bucketsMu.RUnlock()

	if ok {
		return bucket
	}

	rl.bucketsMu.Lock()
	defer rl.bucketsMu.Unlock()

	// Double-check after acquiring write lock
	if bucket, ok := rl.buckets[key]; ok {
		return bucket
	}

	bucket = &Bucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastUpdate: time.Now(),
	}
	rl.buckets[key] = bucket
	return bucket
}

// getOrderBucket gets or creates an order bucket for a key
func (rl *RateLimiter) getOrderBucket(key string) *Bucket {
	rl.orderBucketsMu.RLock()
	bucket, ok := rl.orderBuckets[key]
	rl.orderBucketsMu.RUnlock()

	if ok {
		return bucket
	}

	rl.orderBucketsMu.Lock()
	defer rl.orderBucketsMu.Unlock()

	// Double-check
	if bucket, ok := rl.orderBuckets[key]; ok {
		return bucket
	}

	bucket = &Bucket{
		tokens:     float64(rl.config.OrderBurst),
		maxTokens:  float64(rl.config.OrderBurst),
		refillRate: float64(rl.config.OrdersPerSecond),
		lastUpdate: time.Now(),
	}
	rl.orderBuckets[key] = bucket
	return bucket
}

// getDailyCounter gets or creates a daily counter for a key
func (rl *RateLimiter) getDailyCounter(key string, limit int) *DailyCounter {
	today := time.Now().Format("2006-01-02")
	counterKey := key + ":" + today

	rl.dailyCountersMu.RLock()
	counter, ok := rl.dailyCounters[counterKey]
	rl.dailyCountersMu.RUnlock()

	if ok {
		return counter
	}

	rl.dailyCountersMu.Lock()
	defer rl.dailyCountersMu.Unlock()

	// Double-check
	if counter, ok := rl.dailyCounters[counterKey]; ok {
		return counter
	}

	counter = &DailyCounter{
		count: 0,
		limit: limit,
		date:  today,
	}
	rl.dailyCounters[counterKey] = counter
	return counter
}

// AllowIP checks if a request from an IP is allowed
func (rl *RateLimiter) AllowIP(ip string) (bool, *RateLimitInfo) {
	bucket := rl.getBucket("ip:"+ip, float64(rl.config.IPBurst), float64(rl.config.IPRequestsPerSecond))
	return rl.tryConsume(bucket, 1)
}

// AllowUser checks if a request from a user is allowed
func (rl *RateLimiter) AllowUser(userID string) (bool, *RateLimitInfo) {
	bucket := rl.getBucket("user:"+userID, float64(rl.config.UserBurst), float64(rl.config.UserRequestsPerSecond))
	return rl.tryConsume(bucket, 1)
}

// AllowOrder checks if an order submission is allowed
func (rl *RateLimiter) AllowOrder(userID string) (bool, *RateLimitInfo) {
	// Check rate limit
	bucket := rl.getOrderBucket("order:" + userID)
	allowed, info := rl.tryConsume(bucket, 1)
	if !allowed {
		return false, info
	}

	// Check daily limit
	counter := rl.getDailyCounter("order:"+userID, rl.config.OrdersPerDay)
	counter.mu.Lock()
	defer counter.mu.Unlock()

	if counter.count >= counter.limit {
		return false, &RateLimitInfo{
			Allowed:        false,
			Remaining:      0,
			Limit:          counter.limit,
			RetryAfter:     rl.secondsUntilMidnight(),
			LimitType:      "daily",
		}
	}

	counter.count++
	return true, &RateLimitInfo{
		Allowed:   true,
		Remaining: counter.limit - counter.count,
		Limit:     counter.limit,
		LimitType: "daily",
	}
}

// tryConsume tries to consume a token from a bucket
func (rl *RateLimiter) tryConsume(bucket *Bucket, tokens float64) (bool, *RateLimitInfo) {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	now := time.Now()

	// Check if blocked
	if bucket.blocked && now.Before(bucket.blockedUntil) {
		return false, &RateLimitInfo{
			Allowed:    false,
			Remaining:  0,
			Limit:      int(bucket.maxTokens),
			RetryAfter: int(bucket.blockedUntil.Sub(now).Seconds()) + 1,
			LimitType:  "blocked",
		}
	}
	bucket.blocked = false

	// Refill tokens
	elapsed := now.Sub(bucket.lastUpdate).Seconds()
	bucket.tokens += elapsed * bucket.refillRate
	if bucket.tokens > bucket.maxTokens {
		bucket.tokens = bucket.maxTokens
	}
	bucket.lastUpdate = now

	// Try to consume
	if bucket.tokens >= tokens {
		bucket.tokens -= tokens
		return true, &RateLimitInfo{
			Allowed:   true,
			Remaining: int(bucket.tokens),
			Limit:     int(bucket.maxTokens),
			LimitType: "rate",
		}
	}

	// Not enough tokens, block the bucket
	bucket.blocked = true
	bucket.blockedUntil = now.Add(rl.config.IPBlockDuration)

	retryAfter := int((tokens - bucket.tokens) / bucket.refillRate) + 1
	return false, &RateLimitInfo{
		Allowed:    false,
		Remaining:  0,
		Limit:      int(bucket.maxTokens),
		RetryAfter: retryAfter,
		LimitType:  "rate",
	}
}

// secondsUntilMidnight returns seconds until midnight
func (rl *RateLimiter) secondsUntilMidnight() int {
	now := time.Now()
	midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	return int(midnight.Sub(now).Seconds())
}

// RateLimitInfo contains rate limit information
type RateLimitInfo struct {
	Allowed    bool   `json:"allowed"`
	Remaining  int    `json:"remaining"`
	Limit      int    `json:"limit"`
	RetryAfter int    `json:"retry_after,omitempty"`
	LimitType  string `json:"limit_type"`
}

// ============ HTTP Middleware ============

// RateLimitMiddleware creates an HTTP middleware for rate limiting
func RateLimitMiddleware(rl *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client IP
			ip := getClientIP(r)

			// Check IP rate limit
			allowed, info := rl.AllowIP(ip)
			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", info.Limit))
				w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", info.Remaining))
				if info.RetryAfter > 0 {
					w.Header().Set("Retry-After", fmt.Sprintf("%d", info.RetryAfter))
				}
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"error":       "rate_limit_exceeded",
					"message":     "Too many requests, please slow down",
					"retry_after": info.RetryAfter,
				})
				return
			}

			// Add rate limit headers
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", info.Limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", info.Remaining))

			// Check user rate limit if authenticated
			userID := getUserFromContext(r.Context())
			if userID != "" {
				allowed, userInfo := rl.AllowUser(userID)
				if !allowed {
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", userInfo.Limit))
					w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", userInfo.Remaining))
					if userInfo.RetryAfter > 0 {
						w.Header().Set("Retry-After", fmt.Sprintf("%d", userInfo.RetryAfter))
					}
					w.WriteHeader(http.StatusTooManyRequests)
					_ = json.NewEncoder(w).Encode(map[string]interface{}{
						"error":       "rate_limit_exceeded",
						"message":     "User rate limit exceeded",
						"retry_after": userInfo.RetryAfter,
					})
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// OrderRateLimitMiddleware creates an HTTP middleware for order rate limiting
func OrderRateLimitMiddleware(rl *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := getUserFromContext(r.Context())
			if userID == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":   "unauthorized",
					"message": "Authentication required for order submission",
				})
				return
			}

			allowed, info := rl.AllowOrder(userID)
			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", info.Limit))
				w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", info.Remaining))
				if info.RetryAfter > 0 {
					w.Header().Set("Retry-After", fmt.Sprintf("%d", info.RetryAfter))
				}
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"error":       "order_limit_exceeded",
					"message":     fmt.Sprintf("Order %s limit exceeded", info.LimitType),
					"retry_after": info.RetryAfter,
					"limit_type":  info.LimitType,
				})
				return
			}

			// Add rate limit headers
			w.Header().Set("X-RateLimit-Order-Remaining", fmt.Sprintf("%d", info.Remaining))

			next.ServeHTTP(w, r)
		})
	}
}

// Helper context key for user ID
type contextKey string

const userContextKey contextKey = "user_id"

// SetUserContext sets the user ID in context
func SetUserContext(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userContextKey, userID)
}

// getUserFromContext gets the user ID from context
func getUserFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value(userContextKey).(string); ok {
		return userID
	}
	return ""
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check for forwarded headers
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to remote address
	ip := r.RemoteAddr
	for i := len(ip) - 1; i >= 0; i-- {
		if ip[i] == ':' {
			return ip[:i]
		}
	}
	return ip
}

// ============ Statistics ============

// Stats returns rate limiter statistics
type Stats struct {
	TotalBuckets      int `json:"total_buckets"`
	OrderBuckets      int `json:"order_buckets"`
	DailyCounters     int `json:"daily_counters"`
	BlockedBuckets    int `json:"blocked_buckets"`
}

// GetStats returns current rate limiter statistics
func (rl *RateLimiter) GetStats() *Stats {
	rl.bucketsMu.RLock()
	totalBuckets := len(rl.buckets)
	blockedCount := 0
	for _, b := range rl.buckets {
		b.mu.Lock()
		if b.blocked && time.Now().Before(b.blockedUntil) {
			blockedCount++
		}
		b.mu.Unlock()
	}
	rl.bucketsMu.RUnlock()

	rl.orderBucketsMu.RLock()
	orderBuckets := len(rl.orderBuckets)
	rl.orderBucketsMu.RUnlock()

	rl.dailyCountersMu.RLock()
	dailyCounters := len(rl.dailyCounters)
	rl.dailyCountersMu.RUnlock()

	return &Stats{
		TotalBuckets:   totalBuckets,
		OrderBuckets:   orderBuckets,
		DailyCounters:  dailyCounters,
		BlockedBuckets: blockedCount,
	}
}
