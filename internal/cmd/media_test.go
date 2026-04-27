package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMediaUploadCommand(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)
	t.Setenv("POPIART_KEY", "pk-demo")
	t.Setenv("POPIART_PROJECT", "proj_media_demo")

	sourcePath := filepath.Join(t.TempDir(), "poster.png")
	if err := os.WriteFile(sourcePath, []byte("png-body"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/media/upload" {
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
		if r.FormValue("project_id") != "proj_media_demo" {
			t.Fatalf("unexpected project_id: %q", r.FormValue("project_id"))
		}
		if r.FormValue("visibility") != "public" {
			t.Fatalf("unexpected visibility: %q", r.FormValue("visibility"))
		}
		if r.FormValue("metadata_json") != `{"origin":"poster"}` {
			t.Fatalf("unexpected metadata_json: %q", r.FormValue("metadata_json"))
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("read multipart file: %v", err)
		}
		defer file.Close()

		if header.Filename != "poster.png" {
			t.Fatalf("unexpected upload filename: %q", header.Filename)
		}
		if header.Header.Get("Content-Type") != "image/png" {
			t.Fatalf("unexpected upload part content type: %q", header.Header.Get("Content-Type"))
		}
		body, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("read upload body: %v", err)
		}
		if string(body) != "png-body" {
			t.Fatalf("unexpected upload body: %q", string(body))
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":{"id":"med_demo_1","project_id":"proj_media_demo","filename":"poster.png","content_type":"image/png","size_bytes":8,"created_at":"2026-04-08T04:00:00Z","url":"http://127.0.0.1:18080/v1/media/med_demo_1/content","visibility":"public","sha256":"demo-sha256"}}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"media", "upload", sourcePath,
		"--filename", "poster.png",
		"--metadata-json", `{"origin":"poster"}`,
		"--visibility", "public",
	})

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected upload data object, got %#v", resp["data"])
	}
	if data["media_id"] != "med_demo_1" {
		t.Fatalf("unexpected media_id: %#v", data["media_id"])
	}
	if data["project_id"] != "proj_media_demo" {
		t.Fatalf("unexpected project_id: %#v", data["project_id"])
	}
	if data["url"] != "https://server.popi.art/v1/media/med_demo_1/content" {
		t.Fatalf("unexpected url: %#v", data["url"])
	}
	if data["stable_url"] != "https://server.popi.art/v1/media/med_demo_1/content" {
		t.Fatalf("unexpected stable_url: %#v", data["stable_url"])
	}
	if data["public_url"] != "https://server.popi.art/v1/media/med_demo_1/content" {
		t.Fatalf("unexpected public_url: %#v", data["public_url"])
	}
	if data["original_url"] != "http://127.0.0.1:18080/v1/media/med_demo_1/content" {
		t.Fatalf("unexpected original_url: %#v", data["original_url"])
	}
	if data["visibility"] != "public" {
		t.Fatalf("unexpected visibility: %#v", data["visibility"])
	}
}

func TestMediaGetCommandReturnsStablePublicURL(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/media/med_demo_1" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":{"id":"med_demo_1","filename":"poster.png","content_type":"image/png","size_bytes":8,"created_at":"2026-04-08T04:00:00Z","url":"http://localhost:18080/v1/media/med_demo_1/content","visibility":"unlisted"}}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{"media", "get", "med_demo_1"})

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected media data object, got %#v", resp["data"])
	}
	if data["id"] != "med_demo_1" || data["media_id"] != "med_demo_1" {
		t.Fatalf("unexpected ids: %#v", data)
	}
	if data["url"] != "https://server.popi.art/v1/media/med_demo_1/content" {
		t.Fatalf("unexpected url: %#v", data["url"])
	}
	if data["stable_url"] != "https://server.popi.art/v1/media/med_demo_1/content" {
		t.Fatalf("unexpected stable_url: %#v", data["stable_url"])
	}
	if data["original_url"] != "http://localhost:18080/v1/media/med_demo_1/content" {
		t.Fatalf("unexpected original_url: %#v", data["original_url"])
	}
}
