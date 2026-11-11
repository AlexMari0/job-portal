// Package testdb gives integration tests a throwaway MongoDB database.
package testdb

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AlexMari0/job-portal/api/internal/store"
)

// New connects to the local MongoDB (TEST_MONGODB_URI, default localhost),
// creates a uniquely named database, and drops it when the test ends.
// Start MongoDB first with `make db`.
func New(t *testing.T) *store.Store {
	t.Helper()

	uri := os.Getenv("TEST_MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	name := fmt.Sprintf("job-portal-test-%s", bson.NewObjectID().Hex())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s, err := store.Connect(ctx, uri, name)
	if err != nil {
		t.Fatalf("connect test MongoDB at %s (run `make db`): %v", uri, err)
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.DB.Drop(ctx)
		_ = s.Close(ctx)
	})
	return s
}

// Insert writes raw documents into a collection, failing the test on error.
// Use bson.D with production BSON types (e.g. date as float64).
func Insert(t *testing.T, s *store.Store, collection string, docs ...any) {
	t.Helper()
	if _, err := s.DB.Collection(collection).InsertMany(context.Background(), docs); err != nil {
		t.Fatalf("insert into %s: %v", collection, err)
	}
}
