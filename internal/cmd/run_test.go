package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunPassesImageParametersThroughUnchanged(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/jobs" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"NOT_FOUND","message":"not found"}}`))
			return
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode job body: %v", err)
		}

		input, ok := body["input"].(map[string]any)
		if !ok {
			t.Fatalf("expected input object, got %#v", body["input"])
		}
		if input["aspect_ratio"] != "9:16" {
			t.Fatalf("expected aspect_ratio to pass through, got %#v", input["aspect_ratio"])
		}
		if input["resolution"] != "1024x1820" {
			t.Fatalf("expected resolution to pass through, got %#v", input["resolution"])
		}
		if input["image_url"] != "https://example.com/reference.png" {
			t.Fatalf("expected image_url to pass through, got %#v", input["image_url"])
		}
		if input["reference_image_url"] != "https://example.com/reference-alias.png" {
			t.Fatalf("expected reference_image_url to pass through, got %#v", input["reference_image_url"])
		}
		if _, exists := input["size"]; exists {
			t.Fatalf("expected no injected size, got %#v", input["size"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"data":{"job_id":"job_passthrough","status":"pending"}}`))
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"run", "popiskill-image-img2img-basic-v1",
		"--input", `{"prompt":"keep subject","aspect_ratio":"9:16","resolution":"1024x1820","image_url":"https://example.com/reference.png","reference_image_url":"https://example.com/reference-alias.png"}`,
	})

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected run data object, got %#v", resp["data"])
	}
	if data["job_id"] != "job_passthrough" {
		t.Fatalf("unexpected job id: %#v", data["job_id"])
	}
}

func TestRunNormalizesAspectRatioWhenPresent(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/jobs" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"NOT_FOUND","message":"not found"}}`))
			return
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode job body: %v", err)
		}

		input, ok := body["input"].(map[string]any)
		if !ok {
			t.Fatalf("expected input object, got %#v", body["input"])
		}
		if input["aspect_ratio"] != "4:5" {
			t.Fatalf("expected normalized aspect_ratio, got %#v", input["aspect_ratio"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"data":{"job_id":"job_passthrough_ratio","status":"pending"}}`))
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"run", "popiskill-image-img2img-basic-v1",
		"--input", `{"prompt":"keep subject","aspect_ratio":"2048x2560"}`,
	})

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected run data object, got %#v", resp["data"])
	}
	if data["job_id"] != "job_passthrough_ratio" {
		t.Fatalf("unexpected job id: %#v", data["job_id"])
	}
}
