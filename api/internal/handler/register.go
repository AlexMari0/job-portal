package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/AlexMari0/job-portal/api/internal/store"
)

// RegisterStore is the slice of the store Company registration needs.
type RegisterStore interface {
	CompanyEmailExists(ctx context.Context, email string) (bool, error)
	CreateCompany(ctx context.Context, name, email, passwordHash, image string) (store.Company, error)
}

var logoRule = fileRule{
	field:    "image",
	maxBytes: 1 << 20, // 1 MB
	types:    []string{"image/png", "image/jpeg", "image/webp"},
	message:  "must be a PNG, JPEG or WebP image up to 1 MB",
}

// Same rules as the Mongoose schema: an email-ish string, password ≥ 6.
var emailPattern = regexp.MustCompile(`^\S+@\S+\.\S+$`)

const minPasswordLength = 6

func emailTaken(w http.ResponseWriter) {
	writeError(w, http.StatusConflict, "email_taken", "A Company with this email is already registered")
}

// RegisterCompany handles POST /companies (multipart: name, email, password,
// image). Order: text fields, logo, email check, and only then the upload,
// so a failed registration never leaves a logo in Cloudinary.
func RegisterCompany(s RegisterStore, tokens TokenIssuer, up Uploader) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := parseMultipart(w, r, logoRule.maxBytes); err != nil {
			writeInvalidFile(w, logoRule) // body too large (or not multipart)
			return
		}
		name := strings.TrimSpace(r.FormValue("name"))
		email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
		password := r.FormValue("password")

		fields := map[string]string{}
		if name == "" {
			fields["name"] = "required"
		}
		if !emailPattern.MatchString(email) {
			fields["email"] = "must be a valid email address"
		}
		if len(password) < minPasswordLength {
			fields["password"] = "must be at least 6 characters"
		}
		if len(fields) > 0 {
			writeValidationError(w, fields)
			return
		}

		logo, err := readFile(w, r, logoRule)
		if err != nil {
			writeInvalidFile(w, logoRule)
			return
		}

		taken, err := s.CompanyEmailExists(r.Context(), email)
		if err != nil {
			writeInternalError(w, r, err)
			return
		}
		if taken {
			emailTaken(w)
			return
		}

		// Cost 10, like the Mongoose pre("save") hook.
		hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
		if err != nil {
			writeInternalError(w, r, err)
			return
		}
		image, err := up.Upload(r.Context(), bytes.NewReader(logo), "job-portal/logos")
		if errors.Is(err, ErrFileRejected) {
			writeInvalidFile(w, logoRule)
			return
		}
		if err != nil {
			writeInternalError(w, r, err)
			return
		}

		company, err := s.CreateCompany(r.Context(), name, email, string(hash), image)
		if errors.Is(err, store.ErrEmailTaken) {
			emailTaken(w) // lost a race with a simultaneous sign-up
			return
		}
		if err != nil {
			writeInternalError(w, r, err)
			return
		}
		writeCompanySession(w, r, tokens, company, http.StatusCreated)
	})
}
