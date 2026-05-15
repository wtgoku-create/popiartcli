package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSkillsListIncludesBuiltInOfficialImage2VideoSkill(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/skills" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"NOT_FOUND","message":"not found"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"data":{"items":[],"total":0,"limit":50,"offset":0}}`))
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{"skills", "list"})

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %#v", resp["data"])
	}
	items, ok := data["items"].([]any)
	if !ok {
		t.Fatalf("expected items array, got %#v", data["items"])
	}
	var found bool
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if entry["id"] == officialImage2VideoSkillID {
			if entry["source"] != "official-runtime" {
				t.Fatalf("expected official-runtime source, got %#v", entry["source"])
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected built-in official skill in list, got %#v", items)
	}
}

func TestSkillsGetFallsBackToBuiltInOfficialImage2VideoSkill(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"NOT_FOUND","message":"not found"}}`))
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{"skills", "get", officialImage2VideoSkillID})

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %#v", resp["data"])
	}
	if data["id"] != officialImage2VideoSkillID {
		t.Fatalf("unexpected skill id: %#v", data["id"])
	}
	if data["source"] != "official-runtime" {
		t.Fatalf("expected official-runtime source, got %#v", data["source"])
	}
	description, _ := data["description"].(string)
	if !strings.Contains(description, "Built-in PopiArt image2video") {
		t.Fatalf("expected built-in description, got %q", description)
	}
	inputSchema, ok := data["input_schema"].(map[string]any)
	if !ok || len(inputSchema) == 0 {
		t.Fatalf("expected non-empty input_schema, got %#v", data["input_schema"])
	}
}

func TestSkillsSchemaOverlaysOfficialImage2VideoPlaceholder(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/skills/"+officialImage2VideoSkillID+"/schema" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"NOT_FOUND","message":"not found"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"data":{"input_schema":{},"output_schema":{}}}`))
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{"skills", "schema", officialImage2VideoSkillID})

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %#v", resp["data"])
	}
	inputSchema, ok := data["input_schema"].(map[string]any)
	if !ok {
		t.Fatalf("expected input_schema object, got %#v", data["input_schema"])
	}
	properties, ok := inputSchema["properties"].(map[string]any)
	if !ok || properties["source_artifact_id"] == nil {
		t.Fatalf("expected built-in image2video schema, got %#v", inputSchema)
	}
}

func TestRunOfficialImage2VideoUsesFallbackModelForUnsupportedPrimaryDuration(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/skills/"+officialImage2VideoSkillID:
			_, _ = w.Write([]byte(`{"ok":true,"data":{"id":"` + officialImage2VideoSkillID + `","name":"` + officialImage2VideoSkillID + `","description":"Reserved image2video test skill. The runtime is not connected yet.","tags":["remote"],"version":"1.0.0","model_type":"video","input_schema":{},"output_schema":{}}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/models/infer":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode infer body: %v", err)
			}
			if body["model_id"] != officialImage2VideoFallbackModelID {
				t.Fatalf("expected fallback model for unsupported duration, got %#v", body["model_id"])
			}
			input, ok := body["input"].(map[string]any)
			if !ok {
				t.Fatalf("expected infer input object, got %#v", body["input"])
			}
			if input["image_url"] != "https://example.com/source.png" {
				t.Fatalf("expected image_url alias mapping, got %#v", input["image_url"])
			}
			if input["duration_s"] != float64(6) {
				t.Fatalf("expected duration_s to mirror seconds, got %#v", input["duration_s"])
			}
			_, _ = w.Write([]byte(`{"ok":true,"data":{"job_id":"job_image2video_fallback","status":"pending"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"NOT_FOUND","message":"not found"}}`))
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"run", officialImage2VideoSkillID,
		"--input", `{"reference_image_url":"https://example.com/source.png","seconds":6,"prompt":"make it cinematic"}`,
	})

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %#v", resp["data"])
	}
	if data["job_id"] != "job_image2video_fallback" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
	if data["model_id"] != officialImage2VideoFallbackModelID {
		t.Fatalf("expected fallback model in response, got %#v", data["model_id"])
	}
	if data["requested_skill_id"] != officialImage2VideoSkillID {
		t.Fatalf("expected requested skill id in response, got %#v", data["requested_skill_id"])
	}
	if data["execution_mode"] != "direct-model-fallback" {
		t.Fatalf("expected direct fallback mode, got %#v", data["execution_mode"])
	}
}

func TestRunOfficialImage2VideoFallbackNormalizesStartEndFrameAliases(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/skills/"+officialImage2VideoSkillID:
			_, _ = w.Write([]byte(`{"ok":true,"data":{"id":"` + officialImage2VideoSkillID + `","name":"` + officialImage2VideoSkillID + `","description":"Reserved image2video test skill. The runtime is not connected yet.","tags":["remote"],"version":"1.0.0","model_type":"video","input_schema":{},"output_schema":{}}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/models/infer":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode infer body: %v", err)
			}
			input, ok := body["input"].(map[string]any)
			if !ok {
				t.Fatalf("expected infer input object, got %#v", body["input"])
			}
			images, ok := input["images"].([]any)
			if !ok || len(images) != 2 || images[0] != "https://example.com/first.jpg" || images[1] != "https://example.com/last.jpg" {
				t.Fatalf("unexpected images: %#v", input["images"])
			}
			metadata := input["metadata"].(map[string]any)
			if metadata["action"] != "firstTailGenerate" {
				t.Fatalf("unexpected metadata action: %#v", metadata)
			}
			if input["end_frame_image_url"] != "https://example.com/last.jpg" {
				t.Fatalf("unexpected end frame alias: %#v", input["end_frame_image_url"])
			}
			_, _ = w.Write([]byte(`{"ok":true,"data":{"job_id":"job_image2video_start_end_fallback","status":"pending"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"NOT_FOUND","message":"not found"}}`))
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"run", officialImage2VideoSkillID,
		"--input", `{"image_url":"https://example.com/first.jpg","end_frame_image_url":"https://example.com/last.jpg","prompt":"transition naturally"}`,
	})

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %#v", resp["data"])
	}
	if data["job_id"] != "job_image2video_start_end_fallback" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}
