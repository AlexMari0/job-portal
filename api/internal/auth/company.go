package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type companyKey struct{}

// WithCompanyID returns a context carrying the signed-in Company's id.
func WithCompanyID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, companyKey{}, id)
}

// CompanyID returns the signed-in Company's id, if any.
func CompanyID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(companyKey{}).(string)
	return id, ok && id != ""
}

// CompanyTokens issues and verifies Company session tokens: HS256 JWTs with
// the Company id in "sub". The Next.js server keeps them in an httpOnly cookie
// and forwards them as "Authorization: Bearer <token>".
type CompanyTokens struct {
	secret []byte
	ttl    time.Duration
}

func NewCompanyTokens(secret string, ttl time.Duration) *CompanyTokens {
	return &CompanyTokens{secret: []byte(secret), ttl: ttl}
}

// Issue signs a token for the Company, valid for the configured lifetime.
func (t *CompanyTokens) Issue(companyID string) (token string, expiresAt time.Time, err error) {
	now := time.Now()
	expiresAt = now.Add(t.ttl).Truncate(time.Second) // JWT "exp" has second precision
	token, err = jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   companyID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}).SignedString(t.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign company token: %w", err)
	}
	return token, expiresAt, nil
}

var errNoSubject = errors.New("token has no subject")

// Verify returns the Company id of a valid, unexpired token signed with our
// secret. Only HS256 is accepted, so a token cannot pick its own algorithm.
func (t *CompanyTokens) Verify(token string) (string, error) {
	var claims jwt.RegisteredClaims
	_, err := jwt.ParseWithClaims(token, &claims,
		func(*jwt.Token) (any, error) { return t.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return "", err
	}
	if claims.Subject == "" {
		return "", errNoSubject
	}
	return claims.Subject, nil
}

// Middleware requires a valid Company token and puts the Company id in the
// request context; onFail answers everything else.
func (t *CompanyTokens) Middleware(onFail http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok {
				onFail.ServeHTTP(w, r)
				return
			}
			id, err := t.Verify(token)
			if err != nil {
				onFail.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithCompanyID(r.Context(), id)))
		})
	}
}
