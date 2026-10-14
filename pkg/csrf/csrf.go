package csrf

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

const (
	// CSRFHeader is the request header used to carry a CSRF token.
	CSRFHeader      = "X-CSRF-Token"
	minSecretLength = 32
)

var errInvalidCSRFInput = errors.New("session ID and a 32-byte CSRF secret are required")

// MakeToken creates a session-bound CSRF token. secret must be a dedicated,
// randomly generated server-side secret and must not be committed to source.
func MakeToken(sessionID string, secret []byte) (string, error) {
	mac, err := tokenMAC(sessionID, secret)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(mac), nil
}

// ValidateToken verifies a token without a timing-sensitive string comparison.
func ValidateToken(token, sessionID string, secret []byte) bool {
	if token == "" {
		return false
	}

	providedMAC, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return false
	}
	expectedMAC, err := tokenMAC(sessionID, secret)
	if err != nil {
		return false
	}
	return hmac.Equal(providedMAC, expectedMAC)
}

func tokenMAC(sessionID string, secret []byte) ([]byte, error) {
	if sessionID == "" || len(secret) < minSecretLength {
		return nil, errInvalidCSRFInput
	}

	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(sessionID)) // hash.Hash writes never return an error.
	return mac.Sum(nil), nil
}
