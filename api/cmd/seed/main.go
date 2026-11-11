// Command seed fills a LOCAL MongoDB with sample data whose document shapes
// match what the Express/Mongoose app writes in production.
//
// It drops the four app collections first, so it refuses to run against
// anything other than localhost.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"golang.org/x/crypto/bcrypt"

	"github.com/AlexMari0/job-portal/api/internal/config"
	"github.com/AlexMari0/job-portal/api/internal/store"
)

// Same password for every sample Company, so you can log in locally.
const samplePassword = "password123"

func main() {
	if err := run(); err != nil {
		slog.Error("seed failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	if err := requireLocal(cfg.MongoURI); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s, err := store.Connect(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		return err
	}
	defer s.Close(context.Background())

	for _, name := range []string{
		store.UsersCollection, store.CompaniesCollection,
		store.JobsCollection, store.JobApplicationsCollection,
	} {
		if err := s.DB.Collection(name).Drop(ctx); err != nil {
			return fmt.Errorf("drop %s: %w", name, err)
		}
	}

	// Mongoose's pre("save") hook hashes with bcrypt cost 10; do the same.
	hash, err := bcrypt.GenerateFromPassword([]byte(samplePassword), 10)
	if err != nil {
		return err
	}

	now := time.Now()
	// `date` is epoch ms. Mongoose stores Date.now() as a BSON Double (it does
	// not fit in int32), so use float64 to match production documents.
	daysAgo := func(d int) float64 { return float64(now.AddDate(0, 0, -d).UnixMilli()) }

	acme, globex := bson.NewObjectID(), bson.NewObjectID()
	companies := []any{
		bson.D{
			{Key: "_id", Value: acme}, {Key: "name", Value: "Acme Corp"},
			{Key: "email", Value: "hr@acme.test"}, {Key: "image", Value: "https://placehold.co/96x96.png?text=Acme"},
			{Key: "password", Value: string(hash)}, {Key: "__v", Value: 0},
		},
		bson.D{
			{Key: "_id", Value: globex}, {Key: "name", Value: "Globex"},
			{Key: "email", Value: "jobs@globex.test"}, {Key: "image", Value: "https://placehold.co/96x96.png?text=Globex"},
			{Key: "password", Value: string(hash)}, {Key: "__v", Value: 0},
		},
	}

	// Candidate _id is the Clerk user id (a string, not an ObjectId).
	users := []any{
		bson.D{
			{Key: "_id", Value: "user_seed_alice"}, {Key: "name", Value: "Alice Santoso"},
			{Key: "email", Value: "alice@example.test"}, {Key: "image", Value: "https://placehold.co/96x96.png?text=A"},
			{Key: "resume", Value: "https://example.test/resume-alice.pdf"}, {Key: "__v", Value: 0},
		},
		bson.D{
			{Key: "_id", Value: "user_seed_budi"}, {Key: "name", Value: "Budi Pratama"},
			{Key: "email", Value: "budi@example.test"}, {Key: "image", Value: "https://placehold.co/96x96.png?text=B"},
			{Key: "resume", Value: ""}, {Key: "__v", Value: 0},
		},
	}

	type jobSeed struct {
		title, location, category, level string
		salary                           int
		company                          bson.ObjectID
		daysAgo                          int
		visible                          bool
	}
	jobSeeds := []jobSeed{
		{"Backend Engineer (Go)", "Bangalore", "Programming", "Intermediate level", 90000, acme, 1, true},
		{"Frontend Engineer (React)", "Mumbai", "Programming", "Beginner level", 60000, acme, 3, true},
		{"Data Analyst", "Hyderabad", "Data Science", "Beginner level", 55000, acme, 5, true},
		{"Product Designer", "California", "Designing", "Senior level", 120000, globex, 2, true},
		{"Network Engineer", "Washington", "Networking", "Intermediate level", 80000, globex, 8, true},
		{"Security Analyst", "New York", "Cybersecurity", "Senior level", 130000, globex, 10, true},
		{"Marketing Lead (hidden)", "Chennai", "Marketing", "Senior level", 70000, globex, 12, false},
	}
	jobIDs := make([]bson.ObjectID, len(jobSeeds))
	jobs := make([]any, len(jobSeeds))
	for i, j := range jobSeeds {
		jobIDs[i] = bson.NewObjectID()
		jobs[i] = bson.D{
			{Key: "_id", Value: jobIDs[i]}, {Key: "title", Value: j.title},
			// description is HTML produced by the Quill editor.
			{Key: "description", Value: "<p>We are hiring a <strong>" + j.title + "</strong>.</p><ul><li>Collaborate with the team</li><li>Ship quality work</li></ul>"},
			{Key: "location", Value: j.location}, {Key: "category", Value: j.category},
			{Key: "level", Value: j.level}, {Key: "salary", Value: j.salary},
			{Key: "date", Value: daysAgo(j.daysAgo)}, {Key: "visible", Value: j.visible},
			{Key: "companyId", Value: j.company}, {Key: "__v", Value: 0},
		}
	}

	application := func(user string, job int, company bson.ObjectID, status string, d int) any {
		return bson.D{
			{Key: "_id", Value: bson.NewObjectID()}, {Key: "userId", Value: user},
			{Key: "companyId", Value: company}, {Key: "jobId", Value: jobIDs[job]},
			{Key: "status", Value: status}, {Key: "date", Value: daysAgo(d)}, {Key: "__v", Value: 0},
		}
	}
	applications := []any{
		application("user_seed_alice", 0, acme, "Pending", 0),
		application("user_seed_alice", 3, globex, "Accepted", 1),
		application("user_seed_alice", 4, globex, "Rejected", 6),
	}

	inserts := []struct {
		collection string
		docs       []any
	}{
		{store.CompaniesCollection, companies},
		{store.UsersCollection, users},
		{store.JobsCollection, jobs},
		{store.JobApplicationsCollection, applications},
	}
	for _, in := range inserts {
		if _, err := s.DB.Collection(in.collection).InsertMany(ctx, in.docs); err != nil {
			return fmt.Errorf("insert %s: %w", in.collection, err)
		}
		slog.Info("seeded", "collection", in.collection, "count", len(in.docs))
	}

	// The indexes Mongoose creates from the schemas (unique emails, foreign
	// keys), so local behaviour matches production, e.g. duplicate sign-ups.
	unique := options.Index().SetUnique(true)
	indexes := map[string][]mongo.IndexModel{
		store.UsersCollection:           {{Keys: bson.D{{Key: "email", Value: 1}}, Options: unique}},
		store.CompaniesCollection:       {{Keys: bson.D{{Key: "email", Value: 1}}, Options: unique}},
		store.JobsCollection:            {{Keys: bson.D{{Key: "companyId", Value: 1}}}},
		store.JobApplicationsCollection: {{Keys: bson.D{{Key: "companyId", Value: 1}}}, {Keys: bson.D{{Key: "jobId", Value: 1}}}},
	}
	for collection, models := range indexes {
		if _, err := s.DB.Collection(collection).Indexes().CreateMany(ctx, models); err != nil {
			return fmt.Errorf("create indexes on %s: %w", collection, err)
		}
	}

	slog.Info("done", "company_password", samplePassword)
	return nil
}

// requireLocal refuses any MongoDB host other than localhost, because seed
// drops collections.
func requireLocal(uri string) error {
	u, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("parse MONGODB_URI: %w", err)
	}
	switch u.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		return nil
	}
	return fmt.Errorf("refusing to seed non-local MongoDB host %q", u.Hostname())
}
