package mock

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/d0hx4n2503/bento-backend-golang/config"
	authFeature "github.com/d0hx4n2503/bento-backend-golang/internal/features/auth"
	authDomain "github.com/d0hx4n2503/bento-backend-golang/internal/features/auth/domain"
	authUsecase "github.com/d0hx4n2503/bento-backend-golang/internal/features/auth/usecase"
	"github.com/d0hx4n2503/bento-backend-golang/internal/models"
)

func TestSessionIssueAuthenticateAndRefresh(t *testing.T) {
	t.Parallel()

	sessions := newAuthUsecase(t)
	want := models.Principal{Subject: "google:subject-123", Email: "user@example.com", Role: "user"}
	accessToken, refreshToken, err := sessions.Issue(want.Subject, want.Email, want.Role)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	principal, err := sessions.Authenticate(accessToken)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if principal != want {
		t.Fatalf("Authenticate() principal = %+v, want %+v", principal, want)
	}

	refreshedAccessToken, _, err := sessions.Refresh(refreshToken)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	principal, err = sessions.Authenticate(refreshedAccessToken)
	if err != nil {
		t.Fatalf("Authenticate() refreshed token error = %v", err)
	}
	if principal != want {
		t.Fatalf("Authenticate() refreshed principal = %+v, want %+v", principal, want)
	}

	if _, err := sessions.Authenticate(refreshToken); !errors.Is(err, authDomain.ErrInvalidToken) {
		t.Fatalf("Authenticate() refresh token error = %v, want %v", err, authDomain.ErrInvalidToken)
	}
}

func TestAdminLoginValidatesConfiguredCredentials(t *testing.T) {
	t.Parallel()

	sessions := newAuthUsecase(t)
	if _, _, err := sessions.AdminLogin("wrong", "admin-password"); !errors.Is(err, authDomain.ErrInvalidCredentials) {
		t.Fatalf("AdminLogin() error = %v, want %v", err, authDomain.ErrInvalidCredentials)
	}

	accessToken, _, err := sessions.AdminLogin("admin", "admin-password")
	if err != nil {
		t.Fatalf("AdminLogin() error = %v", err)
	}
	principal, err := sessions.Authenticate(accessToken)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	want := models.Principal{Subject: "admin-dashboard", Role: "admin"}
	if principal != want {
		t.Fatalf("Authenticate() principal = %+v, want %+v", principal, want)
	}
}

func TestAuthenticateRejectsMalformedToken(t *testing.T) {
	t.Parallel()

	if _, err := newAuthUsecase(t).Authenticate("not-a-jwt"); !errors.Is(err, authDomain.ErrInvalidToken) {
		t.Fatalf("Authenticate() error = %v, want %v", err, authDomain.ErrInvalidToken)
	}
}

func newAuthUsecase(t *testing.T) authFeature.Usecase {
	t.Helper()

	accessPrivatePath, accessPublicPath := writeRSAKeyPair(t, "access")
	refreshPrivatePath, refreshPublicPath := writeRSAKeyPair(t, "refresh")
	sessions, err := authUsecase.NewService(config.AuthConfig{
		AccessTokenPrivateKeyPath:  accessPrivatePath,
		AccessTokenPublicKeyPath:   accessPublicPath,
		RefreshTokenPrivateKeyPath: refreshPrivatePath,
		RefreshTokenPublicKeyPath:  refreshPublicPath,
		AccessTokenTTL:             15 * time.Minute,
		RefreshTokenTTL:            24 * time.Hour,
		AdminUsername:              "admin",
		AdminPassword:              "admin-password",
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return sessions
}

func writeRSAKeyPair(t *testing.T, name string) (string, string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate %s RSA key: %v", name, err)
	}

	privateKeyPath := filepath.Join(t.TempDir(), name+"-private.pem")
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	if err := os.WriteFile(privateKeyPath, privateKeyPEM, 0o600); err != nil {
		t.Fatalf("write %s private key: %v", name, err)
	}

	publicKeyPath := filepath.Join(t.TempDir(), name+"-public.pem")
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&key.PublicKey),
	})
	if err := os.WriteFile(publicKeyPath, publicKeyPEM, 0o600); err != nil {
		t.Fatalf("write %s public key: %v", name, err)
	}

	return privateKeyPath, publicKeyPath
}
