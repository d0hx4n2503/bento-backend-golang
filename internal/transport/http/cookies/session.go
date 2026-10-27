package cookies

import (
	"net/http"
	"strings"

	"github.com/d0hx4n2503/bento-backend-golang/config"
	"github.com/d0hx4n2503/bento-backend-golang/pkg/csrf"
	"github.com/labstack/echo/v5"
)

const (
	AccessTokenName  = "access_token"
	RefreshTokenName = "refresh_token"
	CSRFTokenName    = "csrf_token"
	RefreshTokenPath = "/api/v1/auth/refresh"
)

// Set writes the HttpOnly session cookies and, when enabled, a readable CSRF cookie.
func Set(c *echo.Context, cfg config.Config, accessToken, refreshToken string) {
	secure := Secure(c, cfg)
	sameSite := http.SameSiteLaxMode
	if secure {
		sameSite = http.SameSiteNoneMode
	}
	set(c, AccessTokenName, accessToken, "/", int(cfg.Auth.AccessTokenTTL.Seconds()), secure, true, sameSite)
	set(c, RefreshTokenName, refreshToken, RefreshTokenPath, int(cfg.Auth.RefreshTokenTTL.Seconds()), secure, true, sameSite)
	if cfg.Server.CSRF {
		accessTokenCSRF, accessErr := csrf.MakeToken(accessToken, []byte(cfg.Server.CSRFSecretKey))
		refreshTokenCSRF, refreshErr := csrf.MakeToken(refreshToken, []byte(cfg.Server.CSRFSecretKey))
		if accessErr == nil && refreshErr == nil {
			token := accessTokenCSRF + "." + refreshTokenCSRF
			set(c, CSRFTokenName, token, "/", int(cfg.Auth.RefreshTokenTTL.Seconds()), secure, false, sameSite)
		}
	}
}

// Clear removes session cookies using the same scope and security attributes.
func Clear(c *echo.Context, cfg config.Config) {
	secure := Secure(c, cfg)
	sameSite := http.SameSiteLaxMode
	if secure {
		sameSite = http.SameSiteNoneMode
	}
	set(c, AccessTokenName, "", "/", -1, secure, true, sameSite)
	set(c, RefreshTokenName, "", RefreshTokenPath, -1, secure, true, sameSite)
	set(c, CSRFTokenName, "", "/", -1, secure, false, sameSite)
}

// Secure reports whether cookies must be restricted to HTTPS.
func Secure(c *echo.Context, cfg config.Config) bool {
	return cfg.Cookie.Secure || strings.EqualFold(c.Request().Header.Get("X-Forwarded-Proto"), "https") || c.IsTLS() || strings.EqualFold(cfg.Server.Mode, "production")
}

func set(c *echo.Context, name, value, path string, maxAge int, secure, httpOnly bool, sameSite http.SameSite) {
	c.SetCookie(&http.Cookie{
		Name: name, Value: value, Path: path, MaxAge: maxAge,
		Secure: secure, HttpOnly: httpOnly, SameSite: sameSite,
	})
}
