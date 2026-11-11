package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"

	"github.com/AlexMari0/job-portal/api/internal/handler"
	"github.com/AlexMari0/job-portal/api/internal/store"
)

// recordingTransport keeps events in memory instead of sending them.
type recordingTransport struct {
	mu     sync.Mutex
	events []*sentry.Event
}

func (t *recordingTransport) SendEvent(e *sentry.Event) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, e)
}
func (t *recordingTransport) Configure(sentry.ClientOptions)        {}
func (t *recordingTransport) Flush(time.Duration) bool              { return true }
func (t *recordingTransport) FlushWithContext(context.Context) bool { return true }
func (t *recordingTransport) Close()                                {}
func (t *recordingTransport) Events() []*sentry.Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]*sentry.Event(nil), t.events...)
}

func initSentry(t *testing.T) *recordingTransport {
	t.Helper()
	tr := &recordingTransport{}
	if err := sentry.Init(sentry.ClientOptions{Dsn: "https://public@example.com/1", Transport: tr}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sentry.Init(sentry.ClientOptions{}) }) // back to a disabled client
	return tr
}

type failingJobs struct{}

func (failingJobs) ListVisibleJobs(context.Context) ([]store.JobWithCompany, error) {
	return nil, errors.New("mongo: connection reset")
}
func (failingJobs) GetVisibleJob(context.Context, string) (store.JobWithCompany, error) {
	return store.JobWithCompany{}, store.ErrNotFound
}

func TestObserveReportsServerErrors(t *testing.T) {
	tr := initSentry(t)
	h := handler.Observe(handler.ListJobs(failingJobs{}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/jobs", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	events := tr.Events()
	if len(events) != 1 || len(events[0].Exception) == 0 || events[0].Exception[0].Value != "mongo: connection reset" {
		t.Fatalf("events = %+v", events)
	}
	if req := events[0].Request; req == nil || req.Method != http.MethodGet || req.URL == "" {
		t.Errorf("event has no request info: %+v", events[0].Request)
	}
}

func TestObserveDoesNotReportClientErrors(t *testing.T) {
	tr := initSentry(t)
	h := handler.Observe(handler.GetJob(failingJobs{}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/jobs/x", nil)
	req.SetPathValue("id", "x")
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound || len(tr.Events()) != 0 {
		t.Errorf("status %d, events %d (404s are not bugs)", rec.Code, len(tr.Events()))
	}
}

func TestObserveRecoversPanics(t *testing.T) {
	tr := initSentry(t)
	h := handler.Observe(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/me", nil))

	if rec.Code != http.StatusInternalServerError || decode[errorJSON](t, rec).Error.Code != "internal_error" {
		t.Errorf("got %d %s", rec.Code, rec.Body)
	}
	if events := tr.Events(); len(events) != 1 || events[0].Message != "boom" {
		t.Errorf("events = %+v", events)
	}
}

func TestObserveWithoutSentryConfigured(t *testing.T) {
	// No DSN (local development): errors are still answered, nothing breaks.
	_ = sentry.Init(sentry.ClientOptions{})
	rec := httptest.NewRecorder()
	handler.Observe(handler.ListJobs(failingJobs{})).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/jobs", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d", rec.Code)
	}
}
