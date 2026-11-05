// Package mongodb implements the application's DAO contracts with MongoDB. It
// owns the client lifecycle, stores, indexes, and readiness checks.
package mongodb

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

const (
	uriEnv      = "MONGODB_URI"
	databaseEnv = "MONGODB_DATABASE"

	serverSelectionTimeout = 5 * time.Second
	connectTimeout         = 5 * time.Second
	maxPoolSize            = 10
)

// Config contains the MongoDB connection settings.
type Config struct {
	URI      string
	Database string
}

// ConfigFromEnv reads the MongoDB settings from MONGODB_URI and
// MONGODB_DATABASE. Both variables are required.
func ConfigFromEnv() (Config, error) {
	cfg := Config{
		URI:      os.Getenv(uriEnv),
		Database: os.Getenv(databaseEnv),
	}
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	if c.URI == "" {
		return fmt.Errorf("%s is not set", uriEnv)
	}
	if c.Database == "" {
		return fmt.Errorf("%s is not set", databaseEnv)
	}
	return nil
}

// Client wraps a MongoDB client and its configured database.
type Client struct {
	client   *mongo.Client
	database string
}

// New creates a client for cfg. It validates the configuration but does not
// contact the deployment; use Ping to check connectivity.
func New(cfg Config) (*Client, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	opts := options.Client().
		ApplyURI(cfg.URI).
		SetServerSelectionTimeout(serverSelectionTimeout).
		SetConnectTimeout(connectTimeout).
		SetMaxPoolSize(maxPoolSize)

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("create mongodb client: %w", err)
	}

	return &Client{client: client, database: cfg.Database}, nil
}

// OpenFromEnv creates a client from environment configuration.
func OpenFromEnv() (*Client, error) {
	cfg, err := ConfigFromEnv()
	if err != nil {
		return nil, err
	}
	return New(cfg)
}

// Database returns a handle to the configured database.
func (c *Client) collection(name string) *mongo.Collection {
	return c.client.Database(c.database).Collection(name)
}

// Ping verifies that a primary is reachable. The caller controls the deadline.
func (c *Client) Ping(ctx context.Context) error {
	if err := c.client.Ping(ctx, readpref.Primary()); err != nil {
		return fmt.Errorf("ping mongodb: %w", err)
	}
	return nil
}

// Close disconnects the client and releases its connections.
func (c *Client) Close(ctx context.Context) error {
	if err := c.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("disconnect mongodb: %w", err)
	}
	return nil
}
