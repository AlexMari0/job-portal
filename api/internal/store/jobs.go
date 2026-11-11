package store

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// ErrNotFound is returned when a requested document does not exist
// (or is not visible to the caller).
var ErrNotFound = errors.New("not found")

// Company mirrors a document in the companies collection.
// Password is never loaded by the job queries below.
type Company struct {
	ID    bson.ObjectID `bson:"_id"`
	Name  string        `bson:"name"`
	Email string        `bson:"email"`
	Image string        `bson:"image"`
}

// Job mirrors a document in the jobs collection.
type Job struct {
	ID          bson.ObjectID `bson:"_id"`
	Title       string        `bson:"title"`
	Description string        `bson:"description"`
	Location    string        `bson:"location"`
	Category    string        `bson:"category"`
	Level       string        `bson:"level"`
	// Mongoose writes numbers as int32 or double; both decode into int64
	// as long as they hold whole numbers.
	Salary    int64         `bson:"salary"`
	Date      int64         `bson:"date"` // epoch milliseconds
	Visible   bool          `bson:"visible"`
	CompanyID bson.ObjectID `bson:"companyId"`
}

// JobWithCompany is a Job joined with its Company (what Mongoose's
// populate("companyId") used to produce).
type JobWithCompany struct {
	Job     `bson:",inline"`
	Company Company `bson:"company"`
}

// withCompany joins each job with its company, dropping jobs whose company
// no longer exists, and never selecting the company password.
var withCompany = mongo.Pipeline{
	{{Key: "$lookup", Value: bson.D{
		{Key: "from", Value: CompaniesCollection},
		{Key: "localField", Value: "companyId"},
		{Key: "foreignField", Value: "_id"},
		{Key: "as", Value: "company"},
	}}},
	{{Key: "$unwind", Value: "$company"}}, // removes jobs with no matching company
	{{Key: "$project", Value: bson.D{{Key: "company.password", Value: 0}}}},
}

// ListVisibleJobs returns every visible Job with its Company, newest first.
func (s *Store) ListVisibleJobs(ctx context.Context) ([]JobWithCompany, error) {
	pipeline := append(mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "visible", Value: true}}}},
		{{Key: "$sort", Value: bson.D{{Key: "date", Value: -1}}}},
	}, withCompany...)

	return s.aggregateJobs(ctx, pipeline)
}

// GetVisibleJob returns one visible Job with its Company, or ErrNotFound
// for unknown, malformed, hidden, or company-less ids.
func (s *Store) GetVisibleJob(ctx context.Context, id string) (JobWithCompany, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return JobWithCompany{}, ErrNotFound
	}

	pipeline := append(mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "_id", Value: oid}, {Key: "visible", Value: true}}}},
	}, withCompany...)

	jobs, err := s.aggregateJobs(ctx, pipeline)
	if err != nil {
		return JobWithCompany{}, err
	}
	if len(jobs) == 0 {
		return JobWithCompany{}, ErrNotFound
	}
	return jobs[0], nil
}

func (s *Store) aggregateJobs(ctx context.Context, pipeline mongo.Pipeline) ([]JobWithCompany, error) {
	cur, err := s.DB.Collection(JobsCollection).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregate jobs: %w", err)
	}
	jobs := []JobWithCompany{} // non-nil so an empty result encodes as [] not null
	if err := cur.All(ctx, &jobs); err != nil {
		return nil, fmt.Errorf("decode jobs: %w", err)
	}
	return jobs, nil
}
