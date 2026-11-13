package store

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// CandidateProfile is the part of a Candidate that Clerk owns and syncs.
// The resume is owned by the job portal and is never touched by sync.
type CandidateProfile struct {
	ID    string // Clerk user id, stored as the users document _id (a string)
	Name  string
	Email string
	Image string
}

// UpsertCandidate creates or updates a Candidate from Clerk data. Returns
// ErrEmailTaken when another Candidate document already has the email.
// Safe to repeat: a retried webhook never duplicates the document or
// clears an uploaded resume.
func (s *Store) UpsertCandidate(ctx context.Context, p CandidateProfile) error {
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "name", Value: p.Name},
			{Key: "email", Value: p.Email},
			{Key: "image", Value: p.Image},
		}},
		// Only on first insert, matching what Mongoose wrote for new users.
		{Key: "$setOnInsert", Value: bson.D{
			{Key: "resume", Value: ""},
			{Key: "__v", Value: int32(0)},
		}},
	}
	_, err := s.DB.Collection(UsersCollection).UpdateByID(ctx, p.ID, update, options.UpdateOne().SetUpsert(true))
	if mongo.IsDuplicateKeyError(err) {
		return ErrEmailTaken // users.email is unique
	}
	if err != nil {
		return fmt.Errorf("upsert candidate %s: %w", p.ID, err)
	}
	return nil
}

// DeleteCandidate removes a Candidate. Deleting one that is already gone
// is not an error. Their Applications stay, as before the migration.
func (s *Store) DeleteCandidate(ctx context.Context, id string) error {
	if _, err := s.DB.Collection(UsersCollection).DeleteOne(ctx, bson.D{{Key: "_id", Value: id}}); err != nil {
		return fmt.Errorf("delete candidate %s: %w", id, err)
	}
	return nil
}

// Candidate mirrors a document in the users collection.
type Candidate struct {
	ID     string `bson:"_id"` // Clerk user id
	Name   string `bson:"name"`
	Email  string `bson:"email"`
	Image  string `bson:"image"`
	Resume string `bson:"resume"` // Cloudinary URL, or "" before the first upload
}

// GetCandidate returns the Candidate with the given Clerk user id, or
// ErrNotFound when the Clerk webhook has not created it (yet).
func (s *Store) GetCandidate(ctx context.Context, id string) (Candidate, error) {
	var c Candidate
	err := s.DB.Collection(UsersCollection).FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&c)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return Candidate{}, ErrNotFound
	}
	if err != nil {
		return Candidate{}, fmt.Errorf("get candidate %s: %w", id, err)
	}
	return c, nil
}

// SetCandidateResume replaces the Candidate's resume URL.
func (s *Store) SetCandidateResume(ctx context.Context, id, url string) error {
	res, err := s.DB.Collection(UsersCollection).UpdateByID(ctx, id,
		bson.D{{Key: "$set", Value: bson.D{{Key: "resume", Value: url}}}})
	if err != nil {
		return fmt.Errorf("set resume %s: %w", id, err)
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}
