package upload

import (
	"errors"
	"testing"

	"github.com/AlexMari0/job-portal/api/internal/handler"
)

func TestClassifyCloudinaryError(t *testing.T) {
	for msg, rejected := range map[string]bool{
		"Invalid PDF file":     true, // corrupt upload: the user's fault
		"Invalid image file":   true,
		"Invalid Signature 1a": false, // our credentials: a server error
		"Invalid api_key abc":  false,
		"Resource not found":   false,
	} {
		err := classify(msg)
		if got := errors.Is(err, handler.ErrFileRejected); got != rejected {
			t.Errorf("%q: rejected = %v, want %v", msg, got, rejected)
		}
	}
}
