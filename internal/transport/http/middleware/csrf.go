package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/d0hx4n2503/bento-backend-golang/internal/transport/http/cookies"
	"github.com/d0hx4n2503/bento-backend-golang/pkg/csrf"
	"github.com/labstack/echo/v5"
)

// CSRF verifies cookie-authenticated state-changing requests with a session-bound token.
func CSRF(secret []byte) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			switch c.Request().Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
				return next(c)
			}

			accessToken := cookieValue(c, cookies.AccessTokenName)
			refreshToken := cookieValue(c, cookies.RefreshTokenName)
			if accessToken == "" && refreshToken == "" {
				return next(c)
			}
			csrfCookie := cookieValue(c, cookies.CSRFTokenName)
			csrfHeader := strings.TrimSpace(c.Request().Header.Get(csrf.CSRFHeader))
			if csrfCookie == "" || csrfHeader == "" || subtle.ConstantTimeCompare([]byte(csrfCookie), []byte(csrfHeader)) != 1 {
				return echo.NewHTTPError(http.StatusForbidden, "invalid CSRF token")
			}
			tokens := strings.SplitN(csrfHeader, ".", 2)
			valid := len(tokens) == 2 && accessToken != "" && csrf.ValidateToken(tokens[0], accessToken, secret)
			valid = valid || (len(tokens) == 2 && refreshToken != "" && csrf.ValidateToken(tokens[1], refreshToken, secret))
			if !valid {
				return echo.NewHTTPError(http.StatusForbidden, "invalid CSRF token")
			}
			return next(c)
		}
	}
}

func cookieValue(c *echo.Context, name string) string {
	cookie, err := c.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}
