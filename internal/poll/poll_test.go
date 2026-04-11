package poll

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/wtgoku-create/popiartcli/internal/api"
	"github.com/wtgoku-create/popiartcli/internal/output"
	"github.com/wtgoku-create/popiartcli/internal/types"
)

func TestWaitForJobReturnsCompletedJob(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/jobs/job_done" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeEnvelope(t, w, map[string]any{
			"job_id": "job_done",
			"status": "done",
		})
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "")
	job, err := WaitForJob(context.Background(), client, "job_done", time.Millisecond, 1)
	if err != nil {
		t.Fatalf("WaitForJob(done) error = %v", err)
	}
	if job == nil || job.JobID != "job_done" || job.Status != "done" {
		t.Fatalf("WaitForJob(done) = %#v, want completed job", job)
	}
}

func TestWaitForJobReturnsFailureDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(t, w, map[string]any{
			"job_id": "job_failed",
			"status": "failed",
			"error": map[string]any{
				"code":    "INSUFFICIENT_CREDITS",
				"message": "insufficient credits",
			},
		})
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "")
	err := withCapturedStderr(t, func() error {
		_, err := WaitForJob(context.Background(), client, "job_failed", time.Millisecond, 1)
		return err
	})
	if err == nil {
		t.Fatal("WaitForJob(failed) error = nil, want JOB_FAILED")
	}

	cliErr, ok := err.(*output.CLIError)
	if !ok {
		t.Fatalf("WaitForJob(failed) error type = %T, want *output.CLIError", err)
	}
	if cliErr.Code != "JOB_FAILED" {
		t.Fatalf("WaitForJob(failed) code = %q, want %q", cliErr.Code, "JOB_FAILED")
	}
	if cliErr.Message != "insufficient credits" {
		t.Fatalf("WaitForJob(failed) message = %q, want %q", cliErr.Message, "insufficient credits")
	}
	if cliErr.Details["job_id"] != "job_failed" {
		t.Fatalf("WaitForJob(failed) job_id detail = %#v, want %q", cliErr.Details["job_id"], "job_failed")
	}
}

func TestWaitForJobReturnsTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(t, w, map[string]any{
			"job_id": "job_pending",
			"status": "running",
		})
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "")
	err := withCapturedStderr(t, func() error {
		_, err := WaitForJob(context.Background(), client, "job_pending", time.Millisecond, 2)
		return err
	})
	if err == nil {
		t.Fatal("WaitForJob(timeout) error = nil, want POLL_TIMEOUT")
	}

	cliErr, ok := err.(*output.CLIError)
	if !ok {
		t.Fatalf("WaitForJob(timeout) error type = %T, want *output.CLIError", err)
	}
	if cliErr.Code != "POLL_TIMEOUT" {
		t.Fatalf("WaitForJob(timeout) code = %q, want %q", cliErr.Code, "POLL_TIMEOUT")
	}
	if cliErr.Details["job_id"] != "job_pending" {
		t.Fatalf("WaitForJob(timeout) job_id detail = %#v, want %q", cliErr.Details["job_id"], "job_pending")
	}
}

func TestMessageFromJobFallsBackToDefault(t *testing.T) {
	if got := messageFromJob(nil); got != "Job failed" {
		t.Fatalf("messageFromJob(nil) = %q, want %q", got, "Job failed")
	}

	jobWithoutMessage := &types.Job{
		Error: &types.JobError{},
	}
	if got := messageFromJob(jobWithoutMessage); got != "Job failed" {
		t.Fatalf("messageFromJob(empty error) = %q, want %q", got, "Job failed")
	}

	jobWithMessage := &types.Job{
		Error: &types.JobError{Message: "boom"},
	}
	if got := messageFromJob(jobWithMessage); got != "boom" {
		t.Fatalf("messageFromJob(with message) = %q, want %q", got, "boom")
	}
}

func writeEnvelope(t *testing.T, w http.ResponseWriter, data map[string]any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	payload, err := json.Marshal(map[string]any{
		"ok":   true,
		"data": data,
	})
	if err != nil {
		t.Fatalf("json.Marshal(envelope) error = %v", err)
	}
	if _, err := w.Write(payload); err != nil {
		t.Fatalf("Write(envelope) error = %v", err)
	}
}

func withCapturedStderr(t *testing.T, fn func() error) error {
	t.Helper()

	oldStderr := os.Stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stderr = writer
	defer func() {
		os.Stderr = oldStderr
	}()

	runErr := fn()
	_ = writer.Close()

	var sink [256]byte
	for {
		if _, readErr := reader.Read(sink[:]); readErr != nil {
			break
		}
	}
	_ = reader.Close()

	return runErr
}
