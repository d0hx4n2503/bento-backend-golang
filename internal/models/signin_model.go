package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// User is the persisted Bento user account shared across feature packages.
type User struct {
	ID          bson.ObjectID     `json:"id" bson:"_id,omitempty"`
	Email       string            `json:"email" bson:"email"`
	GoogleID    string            `json:"-" bson:"google_id,omitempty"`
	ProviderIDs map[string]string `json:"-" bson:"provider_ids,omitempty"`
	CreatedAt   time.Time         `json:"createdAt" bson:"created_at"`
	UpdatedAt   time.Time         `json:"updatedAt" bson:"updated_at"`
}

// IdentityProvider identifies an external authentication provider.
type IdentityProvider string

const ProviderGoogle IdentityProvider = "google"

// ProviderIdentity is a verified identity returned by an authentication provider.
type ProviderIdentity struct {
	Provider IdentityProvider
	Subject  string
	Email    string
}
