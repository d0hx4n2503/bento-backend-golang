package middleware

import "testing"

func TestCORSAllowsOnlyConfiguredOrigins(t *testing.T) {
	t.Parallel()

	allowOrigin := allowConfiguredOrigin(map[string]struct{}{
		"https://app.example.com": {},
		"http://localhost:5173":   {},
	})
	tests := []struct {
		name       string
		origin     string
		wantOrigin string
		wantAllow  bool
	}{
		{name: "configured production origin", origin: "https://app.example.com", wantOrigin: "https://app.example.com", wantAllow: true},
		{name: "configured development origin", origin: "http://localhost:5173", wantOrigin: "http://localhost:5173", wantAllow: true},
		{name: "default https port", origin: "https://app.example.com:443", wantOrigin: "https://app.example.com", wantAllow: true},
		{name: "unlisted subdomain", origin: "https://evil.bentoguard.xyz"},
		{name: "unlisted cloudflare tunnel", origin: "https://random.trycloudflare.com"},
		{name: "unlisted localhost port", origin: "http://localhost:3001"},
		{name: "origin with path", origin: "https://app.example.com/attacker"},
		{name: "null origin", origin: "null"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			origin, allowed, err := allowOrigin(nil, test.origin)
			if err != nil {
				t.Fatalf("allowConfiguredOrigin() error = %v", err)
			}
			if origin != test.wantOrigin || allowed != test.wantAllow {
				t.Fatalf("allowConfiguredOrigin() = (%q, %t), want (%q, %t)", origin, allowed, test.wantOrigin, test.wantAllow)
			}
		})
	}
}

func TestNormalizeOriginRejectsNonOrigins(t *testing.T) {
	t.Parallel()

	for _, origin := range []string{
		"*",
		"https://*.example.com",
		"https://user:password@app.example.com",
		"https://app.example.com/path",
		"https://app.example.com?query=1",
		"https://app.example.com#fragment",
		"ftp://app.example.com",
		"null",
	} {
		t.Run(origin, func(t *testing.T) {
			t.Parallel()
			if _, err := normalizeOrigin(origin); err == nil {
				t.Fatalf("normalizeOrigin(%q) accepted invalid origin", origin)
			}
		})
	}
}

func TestNormalizeOriginPreservesIPv6HostWhenRemovingDefaultPort(t *testing.T) {
	t.Parallel()

	origin, err := normalizeOrigin("http://[::80]:80")
	if err != nil {
		t.Fatalf("normalizeOrigin() error = %v", err)
	}
	if origin != "http://[::80]" {
		t.Fatalf("normalizeOrigin() = %q, want %q", origin, "http://[::80]")
	}
}
