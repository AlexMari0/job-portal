package store

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
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

// UpsertCandidate creates or updates a Candidate from Clerk data.
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
