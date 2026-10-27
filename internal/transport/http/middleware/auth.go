package middleware

import (
	"net/http"
	"strings"

	"github.com/d0hx4n2503/bento-backend-golang/internal/features/auth"
	"github.com/d0hx4n2503/bento-backend-golang/internal/transport/http/cookies"
	"github.com/labstack/echo/v5"
)

// RequireAuthentication accepts a Bearer token or the access-token cookie.
func RequireAuthentication(sessions auth.Usecase) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			token := bearerToken(c.Request().Header.Get(echo.HeaderAuthorization))
			if token == "" {
				cookie, err := c.Cookie(cookies.AccessTokenName)
				if err == nil {
					token = cookie.Value
				}
			}
			if token == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing access token")
			}
			principal, err := sessions.Authenticate(token)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired access token")
			}
			c.Set("principal", principal)
			return next(c)
		}
	}
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}
