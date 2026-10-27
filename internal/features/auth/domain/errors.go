package domain

import "errors"

var (
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrInvalidCredentials = errors.New("invalid credentials")
)
