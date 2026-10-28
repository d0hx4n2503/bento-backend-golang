package google

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/d0hx4n2503/bento-backend-golang/internal/features/sign-in"
	"github.com/d0hx4n2503/bento-backend-golang/internal/features/sign-in/domain"
	"github.com/d0hx4n2503/bento-backend-golang/internal/models"
)

const (
	tokenInfoEndpoint    = "https://oauth2.googleapis.com/tokeninfo"
	maxAccessTokenLength = 8 << 10
	maxTokenInfoBodySize = 64 << 10
	googleRequestTimeout = 5 * time.Second
	googleEmailScope     = "https://www.googleapis.com/auth/userinfo.email"
)

type tokenInfoResponse struct {
	Audience      string `json:"aud"`
	Authorized    string `json:"azp"`
	Subject       string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	ExpiresAt     string `json:"exp"`
	Scope         string `json:"scope"`
}

// Verifier validates Google access tokens and resolves their user identity.
type verifier struct {
	client           *http.Client
	expectedClientID string
}

// NewVerifier creates a Google token verifier bound to the configured OAuth client.
func NewVerifier(clientID string, client *http.Client) (signin.Provider, error) {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return nil, fmt.Errorf("Google OAuth client ID is required")
	}
	if client == nil {
		client = &http.Client{Timeout: googleRequestTimeout}
	}

	clientCopy := *client
	clientCopy.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}

	return &verifier{client: &clientCopy, expectedClientID: clientID}, nil
}

func (v *verifier) Verify(ctx context.Context, accessToken string) (models.ProviderIdentity, error) {
	if ctx == nil {
		return models.ProviderIdentity{}, errors.New("Google verification context is required")
	}
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" || len(accessToken) > maxAccessTokenLength {
		return models.ProviderIdentity{}, domain.ErrInvalidIdentity
	}

	requestContext, cancel := context.WithTimeout(ctx, googleRequestTimeout)
	defer cancel()

	endpoint, err := url.Parse(tokenInfoEndpoint)
	if err != nil {
		return models.ProviderIdentity{}, errors.New("create Google tokeninfo request")
	}
	query := endpoint.Query()
	query.Set("access_token", accessToken)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(requestContext, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return models.ProviderIdentity{}, errors.New("create Google tokeninfo request")
	}
	response, err := v.client.Do(request)
	if err != nil {
		// The HTTP client error can contain the full request URL, including the token.
		return models.ProviderIdentity{}, errors.New("Google tokeninfo request failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return models.ProviderIdentity{}, domain.ErrInvalidIdentity
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxTokenInfoBodySize+1))
	if err != nil {
		return models.ProviderIdentity{}, errors.New("read Google tokeninfo response")
	}
	if len(body) > maxTokenInfoBodySize {
		return models.ProviderIdentity{}, errors.New("Google tokeninfo response exceeds size limit")
	}

	var info tokenInfoResponse
	if err := json.Unmarshal(body, &info); err != nil {
		return models.ProviderIdentity{}, errors.New("decode Google tokeninfo response")
	}
	if !v.isValid(info) {
		return models.ProviderIdentity{}, domain.ErrInvalidIdentity
	}

	return models.ProviderIdentity{
		Provider: models.ProviderGoogle,
		Subject:  info.Subject,
		Email:    info.Email,
	}, nil
}

func (v *verifier) isValid(info tokenInfoResponse) bool {
	if info.Audience != v.expectedClientID || info.Authorized != v.expectedClientID {
		return false
	}
	if strings.TrimSpace(info.Subject) == "" || strings.TrimSpace(info.Email) == "" || info.EmailVerified != "true" {
		return false
	}

	expiresAt, err := strconv.ParseInt(info.ExpiresAt, 10, 64)
	if err != nil || expiresAt <= time.Now().Unix() {
		return false
	}

	return hasScope(info.Scope, "openid") &&
		(hasScope(info.Scope, "email") || hasScope(info.Scope, googleEmailScope))
}

func hasScope(scopes, expected string) bool {
	for _, scope := range strings.Fields(scopes) {
		if scope == expected {
			return true
		}
	}
	return false
}

var _ signin.Provider = (*verifier)(nil)
