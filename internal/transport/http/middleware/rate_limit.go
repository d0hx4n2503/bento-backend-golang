package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/redis/go-redis/v9"
)

const (
	requestsPerIPPerMinute  = 100_000
	requestsGlobalPerMinute = 500_000
	rateLimitWindow         = time.Minute
)

var rateLimitScript = redis.NewScript(`
local ip_count = redis.call('INCR', KEYS[1])
if ip_count == 1 then redis.call('PEXPIRE', KEYS[1], ARGV[1]) end
local global_count = redis.call('INCR', KEYS[2])
if global_count == 1 then redis.call('PEXPIRE', KEYS[2], ARGV[1]) end
if ip_count > tonumber(ARGV[2]) or global_count > tonumber(ARGV[3]) then return 0 end
return 1
`)

// RateLimit applies the API's per-IP and global fixed-window ceilings through
// Redis so all API replicas share the same counters.
func RateLimit(client *redis.Client) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
			defer cancel()

			ipHash := sha256.Sum256([]byte(c.RealIP()))
			keys := []string{
				"bento:ratelimit:v1:ip:" + hex.EncodeToString(ipHash[:]),
				"bento:ratelimit:v1:global",
			}
			allowed, err := rateLimitScript.Run(ctx, client, keys,
				rateLimitWindow.Milliseconds(), requestsPerIPPerMinute, requestsGlobalPerMinute,
			).Int64()
			if err != nil {
				return echo.NewHTTPError(http.StatusServiceUnavailable, "request limit service unavailable")
			}
			if allowed == 0 {
				return echo.NewHTTPError(http.StatusTooManyRequests, "rate limit exceeded")
			}
			return next(c)
		}
	}
}
