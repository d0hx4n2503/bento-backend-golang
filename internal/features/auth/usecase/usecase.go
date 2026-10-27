package usecase

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/d0hx4n2503/bento-backend-golang/config"
	"github.com/d0hx4n2503/bento-backend-golang/internal/features/auth"
	"github.com/d0hx4n2503/bento-backend-golang/internal/features/auth/domain"
	"github.com/d0hx4n2503/bento-backend-golang/internal/models"
)

const jwtAlgorithm = "RS256"

type tokenHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

type tokenClaims struct {
	models.Principal
	IssuedAt  int64 `json:"iat"`
	ExpiresAt int64 `json:"exp"`
}

// Service signs, refreshes, and verifies RS256 sessions.
type service struct {
	accessPrivate  *rsa.PrivateKey
	accessPublic   *rsa.PublicKey
	refreshPrivate *rsa.PrivateKey
	refreshPublic  *rsa.PublicKey
	accessTTL      time.Duration
	refreshTTL     time.Duration
	adminUsername  string
	adminPassword  string
	clock          func() time.Time
}

func NewService(cfg config.AuthConfig) (auth.Usecase, error) {
	accessPrivate, err := readPrivateKey(cfg.AccessTokenPrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load access token signing key: %w", err)
	}
	accessPublic, err := readPublicKey(cfg.AccessTokenPublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load access token verification key: %w", err)
	}
	refreshPrivate, err := readPrivateKey(cfg.RefreshTokenPrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load refresh token signing key: %w", err)
	}
	refreshPublic, err := readPublicKey(cfg.RefreshTokenPublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load refresh token verification key: %w", err)
	}
	if !sameKey(&accessPrivate.PublicKey, accessPublic) {
		return nil, fmt.Errorf("access token public key does not match private key")
	}
	if !sameKey(&refreshPrivate.PublicKey, refreshPublic) {
		return nil, fmt.Errorf("refresh token public key does not match private key")
	}
	if accessPrivate.N.BitLen() < 2048 || refreshPrivate.N.BitLen() < 2048 {
		return nil, fmt.Errorf("RS256 signing keys must be at least 2048 bits")
	}
	if cfg.AccessTokenTTL <= 0 || cfg.RefreshTokenTTL <= cfg.AccessTokenTTL {
		return nil, fmt.Errorf("token lifetimes must be positive and refresh lifetime must exceed access lifetime")
	}
	return &service{
		accessPrivate: accessPrivate, accessPublic: accessPublic,
		refreshPrivate: refreshPrivate, refreshPublic: refreshPublic,
		accessTTL: cfg.AccessTokenTTL, refreshTTL: cfg.RefreshTokenTTL,
		adminUsername: cfg.AdminUsername, adminPassword: cfg.AdminPassword,
		clock: time.Now,
	}, nil
}

// Issue creates access and refresh JWTs with Bento's RS256 token format.
func (s *service) Issue(subject, email, role string) (string, string, error) {
	principal := models.Principal{Subject: strings.TrimSpace(subject), Email: strings.TrimSpace(email), Role: strings.TrimSpace(role)}
	if principal.Subject == "" || (principal.Role != "admin" && principal.Email == "") {
		return "", "", fmt.Errorf("token subject and user email are required")
	}
	now := s.clock().UTC()
	accessToken, err := signToken(s.accessPrivate, tokenClaims{
		Principal: principal, IssuedAt: now.Unix(), ExpiresAt: now.Add(s.accessTTL).Unix(),
	})
	if err != nil {
		return "", "", fmt.Errorf("sign access token: %w", err)
	}
	refreshToken, err := signToken(s.refreshPrivate, tokenClaims{
		Principal: principal, IssuedAt: now.Unix(), ExpiresAt: now.Add(s.refreshTTL).Unix(),
	})
	if err != nil {
		return "", "", fmt.Errorf("sign refresh token: %w", err)
	}
	return accessToken, refreshToken, nil
}

// Refresh verifies a refresh token and rotates the token pair.
func (s *service) Refresh(token string) (string, string, error) {
	claims, err := verifyToken(s.refreshPublic, token, s.clock())
	if err != nil {
		return "", "", err
	}
	return s.Issue(claims.Subject, claims.Email, claims.Role)
}

// Authenticate verifies an access token and returns its user principal.
func (s *service) Authenticate(token string) (models.Principal, error) {
	claims, err := verifyToken(s.accessPublic, token, s.clock())
	if err != nil {
		return models.Principal{}, err
	}
	if claims.Subject == "" || (claims.Role != "admin" && claims.Email == "") {
		return models.Principal{}, domain.ErrInvalidToken
	}
	return claims.Principal, nil
}

// AdminLogin issues a session when both admin credentials are configured and match.
func (s *service) AdminLogin(username, password string) (string, string, error) {
	usernameValid := constantTimeEqual(username, s.adminUsername)
	passwordValid := constantTimeEqual(password, s.adminPassword)
	if s.adminUsername == "" || s.adminPassword == "" || !usernameValid || !passwordValid {
		return "", "", domain.ErrInvalidCredentials
	}
	return s.Issue("admin-dashboard", "", "admin")
}

func signToken(key *rsa.PrivateKey, claims tokenClaims) (string, error) {
	header, err := json.Marshal(tokenHeader{Algorithm: jwtAlgorithm, Type: "JWT"})
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	hash := sha256.Sum256([]byte(encoded))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash[:])
	if err != nil {
		return "", err
	}
	return encoded + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func verifyToken(key *rsa.PublicKey, token string, now time.Time) (tokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return tokenClaims{}, domain.ErrInvalidToken
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return tokenClaims{}, domain.ErrInvalidToken
	}
	var header tokenHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil || header.Algorithm != jwtAlgorithm || header.Type != "JWT" {
		return tokenClaims{}, domain.ErrInvalidToken
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return tokenClaims{}, domain.ErrInvalidToken
	}
	hash := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, hash[:], signature); err != nil {
		return tokenClaims{}, domain.ErrInvalidToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return tokenClaims{}, domain.ErrInvalidToken
	}
	var claims tokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil || claims.ExpiresAt <= now.Unix() || claims.Subject == "" || claims.IssuedAt > now.Add(30*time.Second).Unix() || claims.ExpiresAt <= claims.IssuedAt {
		return tokenClaims{}, domain.ErrInvalidToken
	}
	return claims, nil
}

func readPrivateKey(path string) (*rsa.PrivateKey, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(contents)
	if block == nil {
		return nil, fmt.Errorf("file %q does not contain a PEM private key", path)
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		if err := key.Validate(); err != nil {
			return nil, fmt.Errorf("validate RSA private key: %w", err)
		}
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse PEM private key: %w", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("file %q does not contain an RSA private key", path)
	}
	if err := key.Validate(); err != nil {
		return nil, fmt.Errorf("validate RSA private key: %w", err)
	}
	return key, nil
}

func readPublicKey(path string) (*rsa.PublicKey, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(contents)
	if block == nil {
		return nil, fmt.Errorf("file %q does not contain a PEM public key", path)
	}
	if key, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse PEM public key: %w", err)
	}
	key, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("file %q does not contain an RSA public key", path)
	}
	return key, nil
}

func sameKey(left, right *rsa.PublicKey) bool {
	return left != nil && right != nil && left.N.Cmp(right.N) == 0 && left.E == right.E
}

func constantTimeEqual(left, right string) bool {
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

var _ auth.Usecase = (*service)(nil)
