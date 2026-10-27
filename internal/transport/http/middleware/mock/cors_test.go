package mock

import (
	"net/http"
	"net/http/httptest"
	"testing"

	httpMiddleware "github.com/d0hx4n2503/bento-backend-golang/internal/transport/http/middleware"
	"github.com/labstack/echo/v5"
)

func TestCORSAllowsOnlyConfiguredOrigins(t *testing.T) {
	t.Parallel()

	allowedOrigins := []string{"https://app.example.com", "http://localhost:5173"}
	tests := []struct {
		name       string
		origin     string
		wantOrigin string
	}{
		{name: "configured production origin", origin: "https://app.example.com", wantOrigin: "https://app.example.com"},
		{name: "configured development origin", origin: "http://localhost:5173", wantOrigin: "http://localhost:5173"},
		{name: "default https port", origin: "https://app.example.com:443", wantOrigin: "https://app.example.com"},
		{name: "unlisted subdomain", origin: "https://evil.example.com"},
		{name: "unlisted cloudflare tunnel", origin: "https://random.trycloudflare.com"},
		{name: "unlisted localhost port", origin: "http://localhost:3001"},
		{name: "origin with path", origin: "https://app.example.com/attacker"},
		{name: "null origin", origin: "null"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := responseAllowOrigin(t, allowedOrigins, test.origin)
			if got != test.wantOrigin {
				t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, test.wantOrigin)
			}
		})
	}
}

func TestCORSDoesNotAllowWildcardConfiguredOrigins(t *testing.T) {
	t.Parallel()

	got := responseAllowOrigin(t, []string{"*", "https://*.example.com"}, "https://app.example.com")
	if got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want no CORS permission", got)
	}
}

func responseAllowOrigin(t *testing.T, allowedOrigins []string, origin string) string {
	t.Helper()

	server := echo.New()
	server.Use(httpMiddleware.CORS(allowedOrigins))
	server.GET("/", func(c *echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Origin", origin)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	return response.Header().Get("Access-Control-Allow-Origin")
}
