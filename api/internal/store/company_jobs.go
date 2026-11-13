package store

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// ErrForbidden is returned when a Company acts on something it does not own.
var ErrForbidden = errors.New("forbidden")

// NewJob is a validated, sanitized Job to create.
type NewJob struct {
	Title, Description, Location, Category, Level string
	Salary                                        int64
	CompanyID                                     bson.ObjectID
}

// JobWithApplicationCount is a Job as its Company sees it in Manage Jobs.
type JobWithApplicationCount struct {
	Job              `bson:",inline"`
	ApplicationCount int `bson:"applicationCount"`
}

// mongooseNumber stores n the way Mongoose (js-bson) does: int32 when it
// fits, double otherwise.
func mongooseNumber(n int64) any {
	if n >= math.MinInt32 && n <= math.MaxInt32 {
		return int32(n)
	}
	return float64(n)
}

// CreateJob inserts a visible Job shaped like the documents Mongoose wrote.
func (s *Store) CreateJob(ctx context.Context, j NewJob, now time.Time) (Job, error) {
	job := Job{
		ID: bson.NewObjectID(), Title: j.Title, Description: j.Description,
		Location: j.Location, Category: j.Category, Level: j.Level, Salary: j.Salary,
		Date: now.UnixMilli(), Visible: true, CompanyID: j.CompanyID,
	}
	doc := bson.D{
		{Key: "_id", Value: job.ID}, {Key: "title", Value: job.Title}, {Key: "description", Value: job.Description},
		{Key: "location", Value: job.Location}, {Key: "category", Value: job.Category}, {Key: "level", Value: job.Level},
		{Key: "salary", Value: mongooseNumber(job.Salary)},
		{Key: "date", Value: float64(job.Date)}, // Date.now() was stored as a double
		{Key: "visible", Value: true}, {Key: "companyId", Value: job.CompanyID}, {Key: "__v", Value: int32(0)},
	}
	if _, err := s.DB.Collection(JobsCollection).InsertOne(ctx, doc); err != nil {
		return Job{}, fmt.Errorf("insert job: %w", err)
	}
	return job, nil
}

// ListCompanyJobs returns all of a Company's Jobs (Hidden ones included),
// newest first, each with its number of Applications.
func (s *Store) ListCompanyJobs(ctx context.Context, companyID bson.ObjectID) ([]JobWithApplicationCount, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "companyId", Value: companyID}}}},
		{{Key: "$sort", Value: bson.D{{Key: "date", Value: -1}}}},
		// Count inside the $lookup, so Application documents are never loaded.
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: JobApplicationsCollection}, {Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "jobId"},
			{Key: "pipeline", Value: mongo.Pipeline{{{Key: "$count", Value: "n"}}}},
			{Key: "as", Value: "applications"}, // [] or [{n: <count>}]
		}}},
		{{Key: "$addFields", Value: bson.D{{Key: "applicationCount", Value: bson.D{
			{Key: "$ifNull", Value: bson.A{bson.D{{Key: "$first", Value: "$applications.n"}}, 0}},
		}}}}},
		{{Key: "$project", Value: bson.D{{Key: "applications", Value: 0}}}},
	}
	cur, err := s.DB.Collection(JobsCollection).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregate company jobs: %w", err)
	}
	jobs := []JobWithApplicationCount{}
	if err := cur.All(ctx, &jobs); err != nil {
		return nil, fmt.Errorf("decode company jobs: %w", err)
	}
	return jobs, nil
}

// SetJobVisibility shows or hides one of the Company's Jobs. Returns
// ErrNotFound for unknown ids and ErrForbidden for other Companies' Jobs.
func (s *Store) SetJobVisibility(ctx context.Context, jobID string, companyID bson.ObjectID, visible bool) (JobWithApplicationCount, error) {
	oid, err := bson.ObjectIDFromHex(jobID)
	if err != nil {
		return JobWithApplicationCount{}, ErrNotFound
	}
	jobs := s.DB.Collection(JobsCollection)

	var job Job
	err = jobs.FindOne(ctx, bson.D{{Key: "_id", Value: oid}}).Decode(&job)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return JobWithApplicationCount{}, ErrNotFound
	}
	if err != nil {
		return JobWithApplicationCount{}, fmt.Errorf("find job: %w", err)
	}
	if job.CompanyID != companyID {
		return JobWithApplicationCount{}, ErrForbidden
	}

	if _, err := jobs.UpdateByID(ctx, oid, bson.D{{Key: "$set", Value: bson.D{{Key: "visible", Value: visible}}}}); err != nil {
		return JobWithApplicationCount{}, fmt.Errorf("set visibility: %w", err)
	}
	job.Visible = visible

	n, err := s.DB.Collection(JobApplicationsCollection).CountDocuments(ctx, bson.D{{Key: "jobId", Value: oid}})
	if err != nil {
		return JobWithApplicationCount{}, fmt.Errorf("count applications: %w", err)
	}
	return JobWithApplicationCount{Job: job, ApplicationCount: int(n)}, nil
}
