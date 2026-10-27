package middleware

import (
	"time"

	"github.com/d0hx4n2503/bento-backend-golang/internal/infrastructure/observability/metrics"
	"github.com/labstack/echo/v5"
)

// Metrics records HTTP counts and durations using normalized route templates.
func Metrics(recorder metrics.Metrics) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			started := time.Now()
			writer := trackStatus(c.Response())
			c.SetResponse(writer)
			err := next(c)
			status := writer.status
			if err != nil {
				status = statusFromError(err)
			}
			route := c.RouteInfo().Path
			if route == "" {
				route = "unmatched"
			}
			recorder.IncHits(status, c.Request().Method, route)
			recorder.ObserveResponseTime(status, c.Request().Method, route, time.Since(started).Seconds())
			return err
		}
	}
}
