package domain

import "errors"

var (
	ErrInvalidIdentity  = errors.New("invalid provider identity")
	ErrIdentityConflict = errors.New("provider identity is already linked to another account")
)
