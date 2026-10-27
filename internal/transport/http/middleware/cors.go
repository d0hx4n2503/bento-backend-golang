package middleware

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v5"
	echoMiddleware "github.com/labstack/echo/v5/middleware"
)

var errInvalidOrigin = errors.New("invalid CORS origin")

// CORS allows credentialed browser requests only from explicitly configured origins.
func CORS(origins []string) echo.MiddlewareFunc {
	allowedOrigins := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		normalized, err := normalizeOrigin(origin)
		if err == nil {
			allowedOrigins[normalized] = struct{}{}
		}
	}

	return echoMiddleware.CORSWithConfig(echoMiddleware.CORSConfig{
		UnsafeAllowOriginFunc: allowConfiguredOrigin(allowedOrigins),
		AllowMethods: []string{
			http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch,
			http.MethodPost, http.MethodDelete, http.MethodOptions,
		},
		AllowHeaders: []string{
			echo.HeaderContentType,
			echo.HeaderAuthorization,
			echo.HeaderAccept,
			echo.HeaderXRequestedWith,
			echo.HeaderOrigin,
			"X-CSRF-Token",
		},
		AllowCredentials: true,
		MaxAge:           600,
	})
}

func allowConfiguredOrigin(allowedOrigins map[string]struct{}) func(*echo.Context, string) (string, bool, error) {
	return func(_ *echo.Context, origin string) (string, bool, error) {
		normalized, err := normalizeOrigin(origin)
		if err != nil {
			return "", false, nil
		}
		if _, allowed := allowedOrigins[normalized]; !allowed {
			return "", false, nil
		}
		return normalized, true, nil
	}
}

func normalizeOrigin(origin string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || strings.Contains(parsed.Host, "*") || strings.HasSuffix(parsed.Host, ":") || parsed.User != nil || parsed.Opaque != "" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return "", errInvalidOrigin
	}

	hostname := strings.ToLower(parsed.Hostname())
	port := parsed.Port()
	if (parsed.Scheme == "http" && port == "80") || (parsed.Scheme == "https" && port == "443") {
		port = ""
	}
	host := hostname
	if strings.Contains(hostname, ":") {
		host = "[" + hostname + "]"
	}
	if port != "" {
		host = net.JoinHostPort(hostname, port)
	}
	return parsed.Scheme + "://" + host, nil
}
