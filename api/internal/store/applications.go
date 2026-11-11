package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Application statuses (the enum in the Mongoose schema).
const (
	StatusPending  = "Pending"
	StatusAccepted = "Accepted"
	StatusRejected = "Rejected"
)

// Application mirrors a document in the jobapplications collection.
type Application struct {
	ID        bson.ObjectID `bson:"_id"`
	UserID    string        `bson:"userId"` // Candidate's Clerk user id
	CompanyID bson.ObjectID `bson:"companyId"`
	JobID     bson.ObjectID `bson:"jobId"`
	Status    string        `bson:"status"`
	Date      int64         `bson:"date"` // epoch milliseconds
}

// ApplicationWithJob is an Application joined with its Job and Company.
type ApplicationWithJob struct {
	Application `bson:",inline"`
	Job         Job     `bson:"job"`
	Company     Company `bson:"company"`
}

// HasApplied reports whether the Candidate already applied to the Job.
func (s *Store) HasApplied(ctx context.Context, candidateID string, jobID bson.ObjectID) (bool, error) {
	err := s.DB.Collection(JobApplicationsCollection).
		FindOne(ctx, bson.D{{Key: "userId", Value: candidateID}, {Key: "jobId", Value: jobID}}).
		Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check application: %w", err)
	}
	return true, nil
}

// CreateApplication stores a new Pending Application for the Job, shaped
// exactly like the documents Mongoose wrote.
func (s *Store) CreateApplication(ctx context.Context, candidateID string, job Job, now time.Time) (Application, error) {
	app := Application{
		ID:        bson.NewObjectID(),
		UserID:    candidateID,
		CompanyID: job.CompanyID,
		JobID:     job.ID,
		Status:    StatusPending,
		Date:      now.UnixMilli(),
	}
	doc := bson.D{
		{Key: "_id", Value: app.ID},
		{Key: "userId", Value: app.UserID},
		{Key: "companyId", Value: app.CompanyID},
		{Key: "jobId", Value: app.JobID},
		{Key: "status", Value: app.Status},
		{Key: "date", Value: float64(app.Date)}, // Mongoose stored Date.now() as a BSON double
		{Key: "__v", Value: int32(0)},
	}
	if _, err := s.DB.Collection(JobApplicationsCollection).InsertOne(ctx, doc); err != nil {
		return Application{}, fmt.Errorf("insert application: %w", err)
	}
	return app, nil
}

// ListCandidateApplications returns the Candidate's Applications, newest
// first, each with its Job and Company. Applications to Hidden Jobs are
// included; ones whose Job or Company no longer exists are left out.
func (s *Store) ListCandidateApplications(ctx context.Context, candidateID string) ([]ApplicationWithJob, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "userId", Value: candidateID}}}},
		{{Key: "$sort", Value: bson.D{{Key: "date", Value: -1}}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: JobsCollection}, {Key: "localField", Value: "jobId"},
			{Key: "foreignField", Value: "_id"}, {Key: "as", Value: "job"},
		}}},
		{{Key: "$unwind", Value: "$job"}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: CompaniesCollection}, {Key: "localField", Value: "companyId"},
			{Key: "foreignField", Value: "_id"}, {Key: "as", Value: "company"},
		}}},
		{{Key: "$unwind", Value: "$company"}},
		{{Key: "$project", Value: bson.D{{Key: "company.password", Value: 0}}}},
	}

	cur, err := s.DB.Collection(JobApplicationsCollection).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregate applications: %w", err)
	}
	apps := []ApplicationWithJob{}
	if err := cur.All(ctx, &apps); err != nil {
		return nil, fmt.Errorf("decode applications: %w", err)
	}
	return apps, nil
}
