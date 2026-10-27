package mock

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/d0hx4n2503/bento-backend-golang/config"
	authHTTP "github.com/d0hx4n2503/bento-backend-golang/internal/features/auth/delivery/http"
	authDomain "github.com/d0hx4n2503/bento-backend-golang/internal/features/auth/domain"
	"github.com/d0hx4n2503/bento-backend-golang/internal/models"
	"github.com/d0hx4n2503/bento-backend-golang/internal/transport/http/cookies"
	"github.com/labstack/echo/v5"
)

var errAuthHandlerTest = errors.New("auth usecase failed")

func TestAdminLoginHandlerValidatesInputAndCredentials(t *testing.T) {
	t.Parallel()

	t.Run("requires a username and password", func(t *testing.T) {
		handler := authHTTP.NewHandler(&authHandlerUsecaseStub{}, config.Config{})
		ctx, _ := newAuthHandlerContext(http.MethodPost, "{}")
		assertHandlerErrorStatus(t, handler.AdminLogin()(ctx), http.StatusBadRequest)
	})

	t.Run("rejects invalid credentials", func(t *testing.T) {
		usecase := &authHandlerUsecaseStub{adminErr: authDomain.ErrInvalidCredentials}
		handler := authHTTP.NewHandler(usecase, config.Config{})
		ctx, _ := newAuthHandlerContext(http.MethodPost, `{"username":"admin","password":"wrong"}`)
		assertHandlerErrorStatus(t, handler.AdminLogin()(ctx), http.StatusUnauthorized)
		if usecase.adminCalls != 1 {
			t.Fatalf("AdminLogin() calls = %d, want 1", usecase.adminCalls)
		}
	})
}

func TestAdminLoginHandlerSetsHttpOnlySessionCookies(t *testing.T) {
	t.Parallel()

	usecase := &authHandlerUsecaseStub{accessToken: "access", sessionRefreshToken: "refresh"}
	cfg := config.Config{
		Auth: config.AuthConfig{AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 24 * time.Hour},
	}
	handler := authHTTP.NewHandler(usecase, cfg)
	ctx, response := newAuthHandlerContext(http.MethodPost, `{"username":"admin","password":"secret"}`)

	if err := handler.AdminLogin()(ctx); err != nil {
		t.Fatalf("AdminLogin() error = %v", err)
	}
	if response.Code != http.StatusOK {
		t.Fatalf("response status = %d, want %d", response.Code, http.StatusOK)
	}
	if usecase.adminUsername != "admin" || usecase.adminPassword != "secret" {
		t.Fatalf("AdminLogin() credentials = %q/%q", usecase.adminUsername, usecase.adminPassword)
	}
	assertSessionCookies(t, response, "access", "refresh")
}

func TestRefreshHandlerValidatesCookieAndMapsErrors(t *testing.T) {
	t.Parallel()

	t.Run("requires refresh cookie", func(t *testing.T) {
		handler := authHTTP.NewHandler(&authHandlerUsecaseStub{}, config.Config{})
		ctx, _ := newAuthHandlerContext(http.MethodPost, "")
		assertHandlerErrorStatus(t, handler.Refresh()(ctx), http.StatusUnauthorized)
	})

	t.Run("rejects invalid refresh token", func(t *testing.T) {
		usecase := &authHandlerUsecaseStub{refreshErr: authDomain.ErrInvalidToken}
		handler := authHTTP.NewHandler(usecase, config.Config{})
		ctx, _ := newAuthHandlerContext(http.MethodPost, "", &http.Cookie{Name: cookies.RefreshTokenName, Value: "invalid"})
		assertHandlerErrorStatus(t, handler.Refresh()(ctx), http.StatusUnauthorized)
		if usecase.refreshInput != "invalid" {
			t.Fatalf("Refresh() token = %q, want invalid", usecase.refreshInput)
		}
	})

	t.Run("maps unexpected failure to internal server error", func(t *testing.T) {
		usecase := &authHandlerUsecaseStub{refreshErr: errAuthHandlerTest}
		handler := authHTTP.NewHandler(usecase, config.Config{})
		ctx, _ := newAuthHandlerContext(http.MethodPost, "", &http.Cookie{Name: cookies.RefreshTokenName, Value: "refresh"})
		assertHandlerErrorStatus(t, handler.Refresh()(ctx), http.StatusInternalServerError)
	})
}

func TestRefreshHandlerReplacesSessionCookies(t *testing.T) {
	t.Parallel()

	usecase := &authHandlerUsecaseStub{accessToken: "new-access", sessionRefreshToken: "new-refresh"}
	cfg := config.Config{Auth: config.AuthConfig{AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 24 * time.Hour}}
	handler := authHTTP.NewHandler(usecase, cfg)
	ctx, response := newAuthHandlerContext(http.MethodPost, "", &http.Cookie{Name: cookies.RefreshTokenName, Value: "old-refresh"})

	if err := handler.Refresh()(ctx); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if response.Code != http.StatusOK {
		t.Fatalf("response status = %d, want %d", response.Code, http.StatusOK)
	}
	if usecase.refreshInput != "old-refresh" {
		t.Fatalf("Refresh() token = %q, want old-refresh", usecase.refreshInput)
	}
	assertSessionCookies(t, response, "new-access", "new-refresh")
}

func TestLogoutHandlerExpiresSessionCookies(t *testing.T) {
	t.Parallel()

	handler := authHTTP.NewHandler(&authHandlerUsecaseStub{}, config.Config{})
	ctx, response := newAuthHandlerContext(http.MethodPost, "")
	if err := handler.Logout()(ctx); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if response.Code != http.StatusOK {
		t.Fatalf("response status = %d, want %d", response.Code, http.StatusOK)
	}
	deleted := make(map[string]bool)
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == cookies.AccessTokenName || cookie.Name == cookies.RefreshTokenName || cookie.Name == cookies.CSRFTokenName {
			if cookie.MaxAge >= 0 {
				t.Errorf("cookie %q max-age = %d, want expired cookie", cookie.Name, cookie.MaxAge)
			}
			deleted[cookie.Name] = true
		}
	}
	for _, name := range []string{cookies.AccessTokenName, cookies.RefreshTokenName, cookies.CSRFTokenName} {
		if !deleted[name] {
			t.Errorf("Logout() did not expire %q cookie", name)
		}
	}
}

func TestCheckHandlerRequiresAndReturnsPrincipal(t *testing.T) {
	t.Parallel()

	handler := authHTTP.NewHandler(&authHandlerUsecaseStub{}, config.Config{})
	t.Run("requires principal", func(t *testing.T) {
		ctx, _ := newAuthHandlerContext(http.MethodGet, "")
		assertHandlerErrorStatus(t, handler.Check()(ctx), http.StatusUnauthorized)
	})

	t.Run("returns authenticated principal", func(t *testing.T) {
		ctx, response := newAuthHandlerContext(http.MethodGet, "")
		ctx.Set("principal", models.Principal{Subject: "user-123", Email: "user@example.com", Role: "user"})
		if err := handler.Check()(ctx); err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"id":"user-123"`) {
			t.Fatalf("Check() response = %d %s", response.Code, response.Body.String())
		}
	})
}

func newAuthHandlerContext(method, body string, requestCookies ...*http.Cookie) (*echo.Context, *httptest.ResponseRecorder) {
	request := httptest.NewRequest(method, "/", strings.NewReader(body))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	for _, cookie := range requestCookies {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	return echo.New().NewContext(request, response), response
}

func assertHandlerErrorStatus(t *testing.T, err error, want int) {
	t.Helper()
	var httpError *echo.HTTPError
	if !errors.As(err, &httpError) {
		t.Fatalf("handler error = %v, want HTTP %d", err, want)
	}
	if httpError.Code != want {
		t.Fatalf("handler HTTP status = %d, want %d", httpError.Code, want)
	}
}

func assertSessionCookies(t *testing.T, response *httptest.ResponseRecorder, accessToken, refreshToken string) {
	t.Helper()
	cookiesByName := make(map[string]*http.Cookie)
	for _, cookie := range response.Result().Cookies() {
		cookiesByName[cookie.Name] = cookie
	}
	for name, wantValue := range map[string]string{
		cookies.AccessTokenName:  accessToken,
		cookies.RefreshTokenName: refreshToken,
	} {
		cookie := cookiesByName[name]
		if cookie == nil {
			t.Errorf("response is missing %q cookie", name)
			continue
		}
		if cookie.Value != wantValue || !cookie.HttpOnly {
			t.Errorf("cookie %q = value %q, HttpOnly %t; want value %q and HttpOnly", name, cookie.Value, cookie.HttpOnly, wantValue)
		}
	}
}

type authHandlerUsecaseStub struct {
	accessToken         string
	sessionRefreshToken string
	adminErr            error
	refreshErr          error
	adminCalls          int
	adminUsername       string
	adminPassword       string
	refreshInput        string
}

func (s *authHandlerUsecaseStub) Issue(string, string, string) (string, string, error) {
	return s.accessToken, s.sessionRefreshToken, nil
}

func (s *authHandlerUsecaseStub) Refresh(token string) (string, string, error) {
	s.refreshInput = token
	return s.accessToken, s.sessionRefreshToken, s.refreshErr
}

func (s *authHandlerUsecaseStub) Authenticate(string) (models.Principal, error) {
	return models.Principal{}, nil
}

func (s *authHandlerUsecaseStub) AdminLogin(username, password string) (string, string, error) {
	s.adminCalls++
	s.adminUsername = username
	s.adminPassword = password
	return s.accessToken, s.sessionRefreshToken, s.adminErr
}
