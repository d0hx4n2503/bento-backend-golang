package http

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/d0hx4n2503/bento-backend-golang/config"
	"github.com/d0hx4n2503/bento-backend-golang/internal/features/sign-in"
	"github.com/d0hx4n2503/bento-backend-golang/internal/features/sign-in/domain"
	"github.com/d0hx4n2503/bento-backend-golang/internal/transport/http/cookies"
	"github.com/labstack/echo/v5"
)

type handlers struct {
	signIn       signin.Usecase
	cfg          config.Config
	contextLimit time.Duration
}

type googleSignInRequest struct {
	Token string `json:"idToken"`
}

func NewHandler(signIn signin.Usecase, cfg config.Config) signin.Handlers {
	return &handlers{signIn: signIn, cfg: cfg, contextLimit: cfg.Server.ContextTimeout}
}

func (h *handlers) GoogleSignIn() echo.HandlerFunc {
	return func(c *echo.Context) error {
		var request googleSignInRequest
		if err := c.Bind(&request); err != nil || strings.TrimSpace(request.Token) == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "idToken is required")
		}
		ctx, cancel := context.WithTimeout(c.Request().Context(), h.contextLimit)
		defer cancel()

		accessToken, refreshToken, err := h.signIn.SignInWithGoogle(ctx, request.Token)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrInvalidIdentity):
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid Google credential")
			case errors.Is(err, domain.ErrIdentityConflict):
				return echo.NewHTTPError(http.StatusConflict, "Google identity is already linked to another account")
			default:
				return echo.NewHTTPError(http.StatusBadGateway, "could not complete Google sign-in")
			}
		}
		cookies.Set(c, h.cfg, accessToken, refreshToken)
		return c.JSON(http.StatusOK, map[string]bool{"success": true})
	}
}

var _ signin.Handlers = (*handlers)(nil)
