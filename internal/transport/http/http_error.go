package http

import (
	"errors"
	"net/http"

	"github.com/d0hx4n2503/bento-backend-golang/internal/infrastructure/observability/logger"
	"github.com/labstack/echo/v5"
)

// NewErrorHandler maps application and framework errors to a consistent public response.
func NewErrorHandler(log logger.Logger) echo.HTTPErrorHandler {
	return func(c *echo.Context, err error) {
		if response, _ := echo.UnwrapResponse(c.Response()); response != nil && response.Committed {
			return
		}

		status := http.StatusInternalServerError
		message := http.StatusText(status)
		var httpError *echo.HTTPError
		if errors.As(err, &httpError) {
			status = httpError.Code
			if status >= http.StatusInternalServerError {
				log.Errorw("HTTP request failed",
					"status", status,
					"request_id", c.Response().Header().Get(echo.HeaderXRequestID),
					"error", err,
				)
			}
			if status < http.StatusInternalServerError {
				if httpError.Message != "" {
					message = httpError.Message
				} else {
					message = http.StatusText(status)
				}
			}
		} else {
			log.Errorw("Unhandled HTTP error",
				"request_id", c.Response().Header().Get(echo.HeaderXRequestID),
				"error", err,
			)
		}
		_ = c.JSON(status, map[string]any{
			"success": false,
			"error": map[string]string{
				"message":   message,
				"requestId": c.Response().Header().Get(echo.HeaderXRequestID),
			},
		})
	}
}
