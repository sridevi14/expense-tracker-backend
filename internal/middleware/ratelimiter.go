package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"expense-tracker/internal/response"
)

func RateLimiter(rdb *redis.Client) gin.HandlerFunc {
	const (
		maxRequests = 100
		window      = 1 * time.Minute
	)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := fmt.Sprintf("rate_limit:%s", ip)

		ctx := context.Background()

		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}

		if count == 1 {
			rdb.Expire(ctx, key, window)
		}

		if count > int64(maxRequests) {
			response.Error(c, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests, please try again later")
			c.Abort()
			return
		}

		c.Next()
	}
}
