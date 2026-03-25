package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetJSONUnwrapsEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_123","status":"pending"}}`)
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	var dst struct {
		JobID  string `json:"job_id"`
		Status string `json:"status"`
	}
	if err := client.GetJSON(context.Background(), "/jobs/job_123", nil, &dst); err != nil {
		t.Fatalf("GetJSON returned error: %v", err)
	}
	if dst.JobID != "job_123" {
		t.Fatalf("expected job_123, got %q", dst.JobID)
	}
	if dst.Status != "pending" {
		t.Fatalf("expected pending, got %q", dst.Status)
	}
}

func TestGetJSONDecodesBarePayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"skill_abc","name":"Skill ABC"}`)
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	var dst struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := client.GetJSON(context.Background(), "/skills/skill_abc", nil, &dst); err != nil {
		t.Fatalf("GetJSON returned error: %v", err)
	}
	if dst.ID != "skill_abc" {
		t.Fatalf("expected skill_abc, got %q", dst.ID)
	}
}

func TestGetJSONReturnsEnvelopeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":false,"error":{"code":"UNAUTHENTICATED","message":"token expired"}}`)
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	var dst struct{}
	err := client.GetJSON(context.Background(), "/auth/me", nil, &dst)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "token expired" {
		t.Fatalf("expected token expired, got %q", err.Error())
	}
}
