package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestUploadFileSendsMultipartAndUnwrapsEnvelope(t *testing.T) {
	uploadPath := filepath.Join(t.TempDir(), "source.txt")
	if err := os.WriteFile(uploadPath, []byte("upload-body"), 0o644); err != nil {
		t.Fatalf("write upload file: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/artifacts/upload" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer pk-demo" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "multipart/form-data; boundary=") {
			t.Fatalf("expected multipart content type, got %q", got)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("parse multipart form: %v", err)
		}
		if r.FormValue("role") != "source" {
			t.Fatalf("expected role=source, got %q", r.FormValue("role"))
		}
		if r.FormValue("metadata_json") != `{"origin":"agent-chat"}` {
			t.Fatalf("unexpected metadata_json: %q", r.FormValue("metadata_json"))
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("read form file: %v", err)
		}
		defer file.Close()

		if header.Filename != "agent-upload.txt" {
			t.Fatalf("expected filename agent-upload.txt, got %q", header.Filename)
		}
		if header.Header.Get("Content-Type") != "text/plain" {
			t.Fatalf("expected part content type text/plain, got %q", header.Header.Get("Content-Type"))
		}
		body, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("read form file body: %v", err)
		}
		if string(body) != "upload-body" {
			t.Fatalf("unexpected upload body: %q", string(body))
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":{"id":"art_upload_123","filename":"agent-upload.txt","content_type":"text/plain","size_bytes":11}}`)
	}))
	defer server.Close()

	client := NewClient(server.URL, "pk-demo")
	var dst struct {
		ID          string `json:"id"`
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
		SizeBytes   int64  `json:"size_bytes"`
	}
	err := client.UploadFile(context.Background(), "/artifacts/upload", uploadPath, UploadFileOptions{
		Filename:    "agent-upload.txt",
		ContentType: "text/plain",
		Fields: map[string]string{
			"role":          "source",
			"metadata_json": `{"origin":"agent-chat"}`,
		},
	}, &dst)
	if err != nil {
		t.Fatalf("UploadFile returned error: %v", err)
	}
	if dst.ID != "art_upload_123" {
		t.Fatalf("expected art_upload_123, got %q", dst.ID)
	}
	if dst.Filename != "agent-upload.txt" {
		t.Fatalf("expected uploaded filename, got %q", dst.Filename)
	}
	if dst.SizeBytes != 11 {
		t.Fatalf("expected size 11, got %d", dst.SizeBytes)
	}
}
