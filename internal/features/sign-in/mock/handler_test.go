package mock

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/d0hx4n2503/bento-backend-golang/config"
	"github.com/d0hx4n2503/bento-backend-golang/internal/features/sign-in"
	signInHTTP "github.com/d0hx4n2503/bento-backend-golang/internal/features/sign-in/delivery/http"
	"github.com/d0hx4n2503/bento-backend-golang/internal/features/sign-in/domain"
	"github.com/d0hx4n2503/bento-backend-golang/internal/transport/http/cookies"
	"github.com/labstack/echo/v5"
)

var errSignInHandlerTest = errors.New("sign-in usecase failed")

func TestGoogleSignInHandlerValidatesRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{name: "missing credential", body: `{}`},
		{name: "malformed JSON", body: `{"idToken":`},
		{name: "blank credential", body: `{"idToken":"  "}`},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			usecase := &signInHandlerUsecaseStub{}
			handler := signInHTTP.NewHandler(usecase, config.Config{})
			ctx, _ := newSignInHandlerContext(test.body)
			assertSignInHTTPErrorStatus(t, handler.GoogleSignIn()(ctx), http.StatusBadRequest)
			if usecase.calls != 0 {
				t.Fatalf("SignInWithGoogle() calls = %d, want 0 for invalid request", usecase.calls)
			}
		})
	}
}

func TestGoogleSignInHandlerCreatesSecureSession(t *testing.T) {
	t.Parallel()

	usecase := &signInHandlerUsecaseStub{accessToken: "access", refreshToken: "refresh"}
	cfg := config.Config{
		Server: config.ServerConfig{ContextTimeout: 3 * time.Second},
		Auth:   config.AuthConfig{AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 24 * time.Hour},
		Cookie: config.CookieConfig{Secure: true},
	}
	handler := signInHTTP.NewHandler(usecase, cfg)
	ctx, response := newSignInHandlerContext(`{"idToken":"google-credential"}`)

	if err := handler.GoogleSignIn()(ctx); err != nil {
		t.Fatalf("GoogleSignIn() error = %v", err)
	}
	if response.Code != http.StatusOK {
		t.Fatalf("response status = %d, want %d", response.Code, http.StatusOK)
	}
	if usecase.calls != 1 || usecase.token != "google-credential" {
		t.Fatalf("SignInWithGoogle() calls/token = %d/%q, want 1/google-credential", usecase.calls, usecase.token)
	}
	cookiesByName := make(map[string]*http.Cookie)
	for _, cookie := range response.Result().Cookies() {
		cookiesByName[cookie.Name] = cookie
	}
	for name, wantValue := range map[string]string{
		cookies.AccessTokenName:  "access",
		cookies.RefreshTokenName: "refresh",
	} {
		cookie := cookiesByName[name]
		if cookie == nil {
			t.Errorf("response is missing %q cookie", name)
			continue
		}
		if cookie.Value != wantValue || !cookie.HttpOnly || !cookie.Secure {
			t.Errorf("cookie %q = value %q, HttpOnly %t, Secure %t; want %q, true, true", name, cookie.Value, cookie.HttpOnly, cookie.Secure, wantValue)
		}
	}
}

func TestGoogleSignInHandlerMapsUsecaseErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		usecaseErr error
		wantStatus int
	}{
		{name: "invalid Google credential", usecaseErr: domain.ErrInvalidIdentity, wantStatus: http.StatusUnauthorized},
		{name: "identity conflict", usecaseErr: domain.ErrIdentityConflict, wantStatus: http.StatusConflict},
		{name: "unexpected dependency failure", usecaseErr: errSignInHandlerTest, wantStatus: http.StatusBadGateway},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			usecase := &signInHandlerUsecaseStub{err: test.usecaseErr}
			handler := signInHTTP.NewHandler(usecase, config.Config{})
			ctx, _ := newSignInHandlerContext(`{"idToken":"google-credential"}`)
			assertSignInHTTPErrorStatus(t, handler.GoogleSignIn()(ctx), test.wantStatus)
		})
	}
}

func newSignInHandlerContext(body string) (*echo.Context, *httptest.ResponseRecorder) {
	request := httptest.NewRequest(http.MethodPost, "/google", strings.NewReader(body))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	response := httptest.NewRecorder()
	return echo.New().NewContext(request, response), response
}

func assertSignInHTTPErrorStatus(t *testing.T, err error, want int) {
	t.Helper()
	var httpError *echo.HTTPError
	if !errors.As(err, &httpError) {
		t.Fatalf("handler error = %v, want HTTP %d", err, want)
	}
	if httpError.Code != want {
		t.Fatalf("handler HTTP status = %d, want %d", httpError.Code, want)
	}
}

type signInHandlerUsecaseStub struct {
	accessToken  string
	refreshToken string
	token        string
	err          error
	calls        int
}

func (s *signInHandlerUsecaseStub) SignInWithGoogle(_ context.Context, token string) (string, string, error) {
	s.calls++
	s.token = token
	return s.accessToken, s.refreshToken, s.err
}

var _ signin.Usecase = (*signInHandlerUsecaseStub)(nil)
