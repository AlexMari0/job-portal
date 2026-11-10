// Package store owns all MongoDB access (the Go counterpart of the
// Mongoose models).
package store

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// Collection names, matching what Mongoose created from the model names.
// The document shapes in these collections must not change during the
// migration (see the wayfinder map).
const (
	UsersCollection           = "users"
	CompaniesCollection       = "companies"
	JobsCollection            = "jobs"
	JobApplicationsCollection = "jobapplications"
)

type Store struct {
	client *mongo.Client
	DB     *mongo.Database
}

// Connect opens a connection pool to MongoDB and verifies it with a ping.
func Connect(ctx context.Context, uri, dbName string) (*Store, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("connect mongo: %w", err)
	}

	s := &Store{client: client, DB: client.Database(dbName)}
	if err := s.Ping(ctx); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("ping mongo: %w", err)
	}
	return s, nil
}

// Ping satisfies handler.Pinger.
func (s *Store) Ping(ctx context.Context) error {
	return s.client.Ping(ctx, readpref.Primary())
}

func (s *Store) Close(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}
