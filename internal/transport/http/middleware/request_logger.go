package middleware

import (
	"errors"
	"net/http"
	"time"

	"github.com/d0hx4n2503/bento-backend-golang/internal/infrastructure/observability/logger"
	"github.com/labstack/echo/v5"
)

// RequestLogger logs one structured summary per completed HTTP request.
func RequestLogger(log logger.Logger) echo.MiddlewareFunc {
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
			log.Infof("http request method=%s route=%s status=%d duration=%s request_id=%s", c.Request().Method, route, status, time.Since(started), c.Response().Header().Get(echo.HeaderXRequestID))
			return err
		}
	}
}

func statusFromError(err error) int {
	var httpError *echo.HTTPError
	if errors.As(err, &httpError) {
		return httpError.Code
	}
	return http.StatusInternalServerError
}
