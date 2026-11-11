package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// CompanyCredentials is a Company with its bcrypt password hash, loaded
// only for login.
type CompanyCredentials struct {
	Company      `bson:",inline"`
	PasswordHash string `bson:"password"`
}

// GetCompanyCredentials finds a Company by email (case-insensitive, as
// Mongoose stored emails lowercased) including its password hash.
func (s *Store) GetCompanyCredentials(ctx context.Context, email string) (CompanyCredentials, error) {
	var c CompanyCredentials
	err := s.DB.Collection(CompaniesCollection).
		FindOne(ctx, bson.D{{Key: "email", Value: strings.ToLower(strings.TrimSpace(email))}}).
		Decode(&c)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return CompanyCredentials{}, ErrNotFound
	}
	if err != nil {
		return CompanyCredentials{}, fmt.Errorf("find company by email: %w", err)
	}
	return c, nil
}

// GetCompany returns a Company (without its password) by id.
func (s *Store) GetCompany(ctx context.Context, id string) (Company, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return Company{}, ErrNotFound
	}
	var c Company
	err = s.DB.Collection(CompaniesCollection).FindOne(ctx, bson.D{{Key: "_id", Value: oid}}).Decode(&c)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return Company{}, ErrNotFound
	}
	if err != nil {
		return Company{}, fmt.Errorf("get company %s: %w", id, err)
	}
	return c, nil
}

// ErrEmailTaken is returned when another Company already uses the email.
var ErrEmailTaken = errors.New("email already registered")

// CompanyEmailExists reports whether a Company uses the email (any case).
func (s *Store) CompanyEmailExists(ctx context.Context, email string) (bool, error) {
	n, err := s.DB.Collection(CompaniesCollection).CountDocuments(ctx,
		bson.D{{Key: "email", Value: strings.ToLower(strings.TrimSpace(email))}}, options.Count().SetLimit(1))
	if err != nil {
		return false, fmt.Errorf("check company email: %w", err)
	}
	return n > 0, nil
}

// CreateCompany inserts a Company shaped like the Mongoose documents (email
// lowercased, bcrypt hash in "password"). A concurrent sign-up with the
// same email hits the unique index and returns ErrEmailTaken.
func (s *Store) CreateCompany(ctx context.Context, name, email, passwordHash, image string) (Company, error) {
	c := Company{ID: bson.NewObjectID(), Name: name, Email: strings.ToLower(strings.TrimSpace(email)), Image: image}
	doc := bson.D{
		{Key: "_id", Value: c.ID}, {Key: "name", Value: c.Name}, {Key: "email", Value: c.Email},
		{Key: "image", Value: c.Image}, {Key: "password", Value: passwordHash}, {Key: "__v", Value: int32(0)},
	}
	_, err := s.DB.Collection(CompaniesCollection).InsertOne(ctx, doc)
	if mongo.IsDuplicateKeyError(err) {
		return Company{}, ErrEmailTaken
	}
	if err != nil {
		return Company{}, fmt.Errorf("insert company: %w", err)
	}
	return c, nil
}
