package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
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
