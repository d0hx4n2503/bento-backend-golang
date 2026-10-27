package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/d0hx4n2503/bento-backend-golang/config"
	"github.com/d0hx4n2503/bento-backend-golang/internal/features/auth"
	"github.com/d0hx4n2503/bento-backend-golang/internal/features/auth/domain"
	"github.com/d0hx4n2503/bento-backend-golang/internal/models"
	"github.com/d0hx4n2503/bento-backend-golang/internal/transport/http/cookies"
	"github.com/labstack/echo/v5"
)

type handlers struct {
	sessions auth.Usecase
	cfg      config.Config
}

type adminLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewHandler(sessions auth.Usecase, cfg config.Config) auth.Handlers {
	return &handlers{sessions: sessions, cfg: cfg}
}

func (h *handlers) AdminLogin() echo.HandlerFunc {
	return func(c *echo.Context) error {
		var request adminLoginRequest
		if err := c.Bind(&request); err != nil || strings.TrimSpace(request.Username) == "" || request.Password == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "username and password are required")
		}
		accessToken, refreshToken, err := h.sessions.AdminLogin(request.Username, request.Password)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
		}
		cookies.Set(c, h.cfg, accessToken, refreshToken)
		return c.JSON(http.StatusOK, map[string]bool{"success": true})
	}
}

func (h *handlers) Refresh() echo.HandlerFunc {
	return func(c *echo.Context) error {
		cookie, err := c.Cookie(cookies.RefreshTokenName)
		if err != nil || strings.TrimSpace(cookie.Value) == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "missing refresh token")
		}
		accessToken, refreshToken, err := h.sessions.Refresh(cookie.Value)
		if err != nil {
			if errors.Is(err, domain.ErrInvalidToken) {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired refresh token")
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "could not refresh session")
		}
		cookies.Set(c, h.cfg, accessToken, refreshToken)
		return c.JSON(http.StatusOK, map[string]bool{"success": true})
	}
}

func (h *handlers) Logout() echo.HandlerFunc {
	return func(c *echo.Context) error {
		cookies.Clear(c, h.cfg)
		return c.JSON(http.StatusOK, map[string]bool{"success": true})
	}
}

func (h *handlers) Check() echo.HandlerFunc {
	return func(c *echo.Context) error {
		principal, err := echo.ContextGet[models.Principal](c, "principal")
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "missing authenticated user")
		}
		return c.JSON(http.StatusOK, map[string]any{
			"success": true,
			"user":    map[string]string{"id": principal.Subject, "email": principal.Email, "role": principal.Role},
		})
	}
}

var _ auth.Handlers = (*handlers)(nil)
