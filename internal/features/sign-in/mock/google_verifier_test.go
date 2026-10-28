package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	googleProvider "github.com/d0hx4n2503/bento-backend-golang/internal/features/sign-in/providers/google"
	"github.com/d0hx4n2503/bento-backend-golang/internal/models"
)

const (
	testGoogleClientID       = "bento-web.apps.googleusercontent.com"
	maxGoogleAccessTokenSize = 8 << 10
	googleEmailScope         = "https://www.googleapis.com/auth/userinfo.email"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type tokenInfo struct {
	Audience      string `json:"aud"`
	Authorized    string `json:"azp"`
	Subject       string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	ExpiresAt     string `json:"exp"`
	Scope         string `json:"scope"`
}

func TestVerifierValidatesGoogleAccessToken(t *testing.T) {
	t.Parallel()

	client := testClient(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodGet || request.URL.Scheme != "https" || request.URL.Host != "oauth2.googleapis.com" || request.URL.Path != "/tokeninfo" {
			return nil, fmt.Errorf("unexpected Google tokeninfo endpoint")
		}
		if got := request.URL.Query().Get("access_token"); got != "google-access-token" {
			return nil, fmt.Errorf("Google access token was not sent for verification")
		}
		return jsonResponse(t, http.StatusOK, validTokenInfo()), nil
	})

	provider, err := googleProvider.NewVerifier(testGoogleClientID, client)
	if err != nil {
		t.Fatalf("NewVerifier() error = %v", err)
	}
	identity, err := provider.Verify(context.Background(), "google-access-token")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	want := models.ProviderIdentity{Provider: models.ProviderGoogle, Subject: "google-subject", Email: "user@example.com"}
	if identity != want {
		t.Fatalf("Verify() identity = %+v, want %+v", identity, want)
	}
}

func TestVerifierRejectsInvalidTokenInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		change func(*tokenInfo)
	}{
		{name: "wrong audience", change: func(info *tokenInfo) { info.Audience = "another-client" }},
		{name: "wrong authorized party", change: func(info *tokenInfo) { info.Authorized = "another-client" }},
		{name: "expired", change: func(info *tokenInfo) {
			info.ExpiresAt = strconv.FormatInt(time.Now().Add(-time.Minute).Unix(), 10)
		}},
		{name: "unverified email", change: func(info *tokenInfo) { info.EmailVerified = "false" }},
		{name: "missing subject", change: func(info *tokenInfo) { info.Subject = " " }},
		{name: "missing openid scope", change: func(info *tokenInfo) { info.Scope = googleEmailScope }},
		{name: "missing email scope", change: func(info *tokenInfo) { info.Scope = "openid profile" }},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			info := validTokenInfo()
			test.change(&info)
			provider, err := googleProvider.NewVerifier(testGoogleClientID, testClient(func(_ *http.Request) (*http.Response, error) {
				return jsonResponse(t, http.StatusOK, info), nil
			}))
			if err != nil {
				t.Fatalf("NewVerifier() error = %v", err)
			}
			if _, err := provider.Verify(context.Background(), "google-access-token"); err == nil {
				t.Fatal("Verify() accepted invalid Google tokeninfo claims")
			}
		})
	}
}

func TestVerifierRequiresOAuthClientID(t *testing.T) {
	t.Parallel()

	if _, err := googleProvider.NewVerifier(" ", nil); err == nil {
		t.Fatal("NewVerifier() accepted an empty client ID")
	}
}

func TestVerifierRejectsOversizedAccessTokenBeforeRequest(t *testing.T) {
	t.Parallel()

	called := false
	provider, err := googleProvider.NewVerifier(testGoogleClientID, testClient(func(_ *http.Request) (*http.Response, error) {
		called = true
		return jsonResponse(t, http.StatusOK, validTokenInfo()), nil
	}))
	if err != nil {
		t.Fatalf("NewVerifier() error = %v", err)
	}
	if _, err := provider.Verify(context.Background(), strings.Repeat("a", maxGoogleAccessTokenSize+1)); err == nil {
		t.Fatal("Verify() accepted an oversized token")
	}
	if called {
		t.Fatal("Verify() sent an oversized token to Google")
	}
}

func TestVerifierDoesNotExposeTokenInTransportError(t *testing.T) {
	t.Parallel()

	provider, err := googleProvider.NewVerifier(testGoogleClientID, testClient(func(request *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("transport error for %s", request.URL.String())
	}))
	if err != nil {
		t.Fatalf("NewVerifier() error = %v", err)
	}
	_, err = provider.Verify(context.Background(), "google-access-token")
	if err == nil {
		t.Fatal("Verify() error = nil, want transport error")
	}
	if strings.Contains(err.Error(), "google-access-token") {
		t.Fatal("Verify() exposed the access token in its error")
	}
}

func TestVerifierDoesNotFollowRedirects(t *testing.T) {
	t.Parallel()

	requests := 0
	provider, err := googleProvider.NewVerifier(testGoogleClientID, testClient(func(request *http.Request) (*http.Response, error) {
		requests++
		return &http.Response{
			StatusCode: http.StatusFound,
			Header:     http.Header{"Location": []string{"https://example.invalid/tokeninfo"}},
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    request,
		}, nil
	}))
	if err != nil {
		t.Fatalf("NewVerifier() error = %v", err)
	}
	if _, err := provider.Verify(context.Background(), "google-access-token"); err == nil {
		t.Fatal("Verify() accepted a redirect response")
	}
	if requests != 1 {
		t.Fatalf("Google tokeninfo requests = %d, want 1", requests)
	}
}

func testClient(roundTrip roundTripFunc) *http.Client {
	return &http.Client{Transport: roundTrip}
}

func validTokenInfo() tokenInfo {
	return tokenInfo{
		Audience:      testGoogleClientID,
		Authorized:    testGoogleClientID,
		Subject:       "google-subject",
		Email:         "user@example.com",
		EmailVerified: "true",
		ExpiresAt:     strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10),
		Scope:         "openid profile " + googleEmailScope,
	}
}

func jsonResponse(t *testing.T, status int, body any) *http.Response {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal test response: %v", err)
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(string(encoded))),
		Header:     make(http.Header),
	}
}
