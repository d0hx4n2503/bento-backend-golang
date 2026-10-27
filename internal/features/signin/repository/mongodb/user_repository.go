package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/d0hx4n2503/bento-backend-golang/internal/features/signin"
	"github.com/d0hx4n2503/bento-backend-golang/internal/features/signin/domain"
	"github.com/d0hx4n2503/bento-backend-golang/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	mongoDriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const usersCollection = "users"

// userRepository persists provider identities in the existing users collection.
type userRepository struct {
	collection *mongoDriver.Collection
}

// NewUserRepository creates a repository backed by the existing users collection.
func NewUserRepository(database *mongoDriver.Database) (signin.MongoRepository, error) {
	if database == nil {
		return nil, fmt.Errorf("mongodb database is required")
	}
	return &userRepository{collection: database.Collection(usersCollection)}, nil
}

// EnsureIndexes creates uniqueness guarantees for email and provider subjects.
func (r *userRepository) EnsureIndexes(ctx context.Context) error {
	googleSubjectIndex := options.Index().SetUnique(true).SetPartialFilterExpression(
		bson.D{{Key: "provider_ids.google", Value: bson.D{{Key: "$type", Value: "string"}}}},
	)
	legacyGoogleIndex := options.Index().SetUnique(true).SetPartialFilterExpression(
		bson.D{{Key: "google_id", Value: bson.D{{Key: "$type", Value: "string"}}}},
	)
	models := []mongoDriver.IndexModel{
		{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "provider_ids.google", Value: 1}}, Options: googleSubjectIndex},
		{Keys: bson.D{{Key: "google_id", Value: 1}}, Options: legacyGoogleIndex},
	}
	if _, err := r.collection.Indexes().CreateMany(ctx, models); err != nil {
		return fmt.Errorf("ensure user indexes: %w", err)
	}
	return nil
}

// FindOrCreateUser links a verified provider identity to one Bento user.
func (r *userRepository) FindOrCreateUser(ctx context.Context, identity models.ProviderIdentity) (models.User, error) {
	identity, err := domain.NormalizeIdentity(identity)
	if err != nil {
		return models.User{}, err
	}
	if identity.Provider != models.ProviderGoogle {
		return models.User{}, domain.ErrInvalidIdentity
	}

	providerField := "provider_ids." + string(identity.Provider)
	identityFilter := bson.A{bson.D{{Key: providerField, Value: identity.Subject}}}
	if identity.Provider == models.ProviderGoogle {
		identityFilter = append(identityFilter, bson.D{{Key: "google_id", Value: identity.Subject}})
	}
	var existing models.User
	err = r.collection.FindOne(ctx, bson.D{{Key: "$or", Value: identityFilter}}).Decode(&existing)
	if err == nil {
		if identity.Provider == models.ProviderGoogle && existing.GoogleID != "" && existing.GoogleID != identity.Subject {
			return models.User{}, domain.ErrIdentityConflict
		}
		return r.updateIdentity(ctx, existing, identity, providerField)
	}
	if !errors.Is(err, mongoDriver.ErrNoDocuments) {
		return models.User{}, fmt.Errorf("find user by provider identity: %w", err)
	}

	// Email equality does not link accounts; existing accounts require an explicit
	// provider-link flow before a different provider can authenticate them.
	now := time.Now().UTC()
	created := models.User{
		ID:          bson.NewObjectID(),
		Email:       identity.Email,
		ProviderIDs: map[string]string{string(identity.Provider): identity.Subject},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if identity.Provider == models.ProviderGoogle {
		created.GoogleID = identity.Subject
	}
	if _, err := r.collection.InsertOne(ctx, created); err != nil {
		if mongoDriver.IsDuplicateKeyError(err) {
			return r.resolveDuplicate(ctx, identity, providerField)
		}
		return models.User{}, fmt.Errorf("insert provider-backed user: %w", err)
	}
	return created, nil
}

func (r *userRepository) updateIdentity(ctx context.Context, document models.User, identity models.ProviderIdentity, providerField string) (models.User, error) {
	now := time.Now().UTC()
	setFields := bson.D{
		{Key: providerField, Value: identity.Subject},
		{Key: "email", Value: identity.Email},
		{Key: "updated_at", Value: now},
	}
	if identity.Provider == models.ProviderGoogle {
		setFields = append(setFields, bson.E{Key: "google_id", Value: identity.Subject})
	}
	var updated models.User
	if err := r.collection.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: document.ID}},
		bson.D{{Key: "$set", Value: setFields}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updated); err != nil {
		if mongoDriver.IsDuplicateKeyError(err) {
			return models.User{}, domain.ErrIdentityConflict
		}
		return models.User{}, fmt.Errorf("update provider identity: %w", err)
	}
	return updated, nil
}

func (r *userRepository) resolveDuplicate(ctx context.Context, identity models.ProviderIdentity, providerField string) (models.User, error) {
	var existing models.User
	filters := bson.A{bson.D{{Key: providerField, Value: identity.Subject}}}
	if identity.Provider == models.ProviderGoogle {
		filters = append(filters, bson.D{{Key: "google_id", Value: identity.Subject}})
	}
	filter := bson.D{{Key: "$or", Value: filters}}
	if err := r.collection.FindOne(ctx, filter).Decode(&existing); err == nil && existing.Email == identity.Email {
		return existing, nil
	}
	return models.User{}, domain.ErrIdentityConflict
}

var _ signin.MongoRepository = (*userRepository)(nil)
