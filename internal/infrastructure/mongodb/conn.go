// Package mongodb owns the shared MongoDB client used by repository adapters.
package mongodb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/d0hx4n2503/bento-backend-golang/config"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// NewClient connects to MongoDB and confirms that the server is reachable.
func NewClient(ctx context.Context, cfg config.MongoDBConfig) (*mongo.Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("mongodb context is required")
	}
	uri := strings.TrimSpace(cfg.URI)
	if uri == "" {
		return nil, fmt.Errorf("mongodb uri is required")
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("connect to mongodb: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("ping mongodb: %w", err)
	}
	return client, nil
}
