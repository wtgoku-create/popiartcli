package popiart

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/wtgoku-create/popiartcli/internal/api"
)

func TestUploadMediaRetriesServerErrors(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "poster.png")
	if err := os.WriteFile(sourcePath, []byte("png-body"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if r.Method != http.MethodPost || r.URL.Path != "/api_client/media/upload" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if attempts == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"message":"boom"}`)
			return
		}

		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("parse multipart form: %v", err)
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("read multipart file: %v", err)
		}
		defer file.Close()
		body, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("read file body: %v", err)
		}
		if string(body) != "png-body" {
			t.Fatalf("unexpected upload body: %q", string(body))
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":{"id":"med_demo_1","url":"http://127.0.0.1:18080/v1/media/med_demo_1/content"}}`)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "pk-demo")
	media, err := UploadMedia(context.Background(), client, sourcePath, UploadOptions{
		Filename:    "poster.png",
		ContentType: "image/png",
		MaxRetries:  3,
	})
	if err != nil {
		t.Fatalf("UploadMedia returned error: %v", err)
	}
	if media.ID != "med_demo_1" {
		t.Fatalf("unexpected media payload: %#v", media)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestStableMediaURL(t *testing.T) {
	got := StableMediaURL("http://127.0.0.1:18080/v1/media/med_demo/content")
	want := "https://server.popi.art/v1/media/med_demo/content"
	if got != want {
		t.Fatalf("StableMediaURL() = %q, want %q", got, want)
	}
}

func TestUploadMediaSupportsWebEnvelopeAndAliasURL(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "poster.png")
	if err := os.WriteFile(sourcePath, []byte("png-body"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api_client/media/upload" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":{"id":3001,"project_id":"proj_demo","name":"poster.png","content_type":"image/png","createTime":"2026-06-10T17:26:26+08:00","fileUrl":"https://static.popi.art/media/2026/0610/poster.png","visibility":"unlisted"},"message":"ok","status":"0000"}`)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "pk-demo")
	media, err := UploadMedia(context.Background(), client, sourcePath, UploadOptions{
		Filename:    "poster.png",
		ContentType: "image/png",
		MaxRetries:  1,
	})
	if err != nil {
		t.Fatalf("UploadMedia returned error: %v", err)
	}
	if media.ID != "3001" || media.URL != "https://static.popi.art/media/2026/0610/poster.png" {
		t.Fatalf("unexpected normalized media payload: %#v", media)
	}
	if media.Filename != "poster.png" || media.CreatedAt != "2026-06-10T17:26:26+08:00" {
		t.Fatalf("unexpected aliased media fields: %#v", media)
	}
}
