// Package upload sends files to Cloudinary (the Go counterpart of the
// multer + cloudinary.uploader setup in Express).
package upload

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"

	"github.com/AlexMari0/job-portal/api/internal/handler"
)

// Cloudinary satisfies handler.Uploader.
type Cloudinary struct {
	cld *cloudinary.Cloudinary
}

func NewCloudinary(cloudName, apiKey, apiSecret string) (*Cloudinary, error) {
	cld, err := cloudinary.NewFromParams(cloudName, apiKey, apiSecret)
	if err != nil {
		return nil, fmt.Errorf("configure cloudinary: %w", err)
	}
	return &Cloudinary{cld: cld}, nil
}

// Upload streams file to Cloudinary under folder and returns its HTTPS URL.
// resource_type "auto" treats PDFs and images the way the Express app did.
func (c *Cloudinary) Upload(ctx context.Context, file io.Reader, folder string) (string, error) {
	res, err := c.cld.Upload.Upload(ctx, file, uploader.UploadParams{
		Folder:       folder,
		ResourceType: "auto",
	})
	if err != nil {
		return "", fmt.Errorf("cloudinary upload: %w", err)
	}
	// Cloudinary reports API errors in the body rather than as a Go error.
	if res.Error.Message != "" {
		return "", classify(res.Error.Message)
	}
	return res.SecureURL, nil
}

// classify turns a Cloudinary error message into an error. "Invalid PDF
// file" / "Invalid image file" mean the upload itself is unreadable, which
// is the user's problem; anything else (bad credentials, outages) is ours.
func classify(msg string) error {
	if strings.HasPrefix(msg, "Invalid ") && strings.HasSuffix(msg, " file") {
		return fmt.Errorf("cloudinary upload: %s: %w", msg, handler.ErrFileRejected)
	}
	return errors.New("cloudinary upload: " + msg)
}
