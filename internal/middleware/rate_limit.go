package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type RateLimiter struct {
	limiters sync.Map
	rate     rate.Limit
	burst    int
}

func NewRateLimiter(rps float64, burst int) *RateLimiter {
	return &RateLimiter{
		rate:  rate.Limit(rps),
		burst: burst,
	}
}

func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	if v, ok := rl.limiters.Load(key); ok {
		return v.(*rate.Limiter)
	}
	limiter := rate.NewLimiter(rl.rate, rl.burst)
	rl.limiters.Store(key, limiter)
	return limiter
}

func (rl *RateLimiter) RateLimitByIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := rl.getLimiter(ip)

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": time.Second.String(),
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func (rl *RateLimiter) RateLimitByAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("Authorization")
		if key == "" {
			key = c.GetHeader("X-Api-Key")
		}
		if key == "" {
			key = c.ClientIP()
		}

		limiter := rl.getLimiter(key)

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
