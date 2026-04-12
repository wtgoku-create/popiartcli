package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/wtgoku-create/popiartcli/internal/output"
)

func TestImageGenerateCommandSubmitsOfficialRuntimeJob(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/jobs" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["skill_id"] != officialText2ImageSkillID {
			t.Fatalf("unexpected skill_id: %#v", body["skill_id"])
		}
		input := body["input"].(map[string]any)
		if input["prompt"] != "hero poster" {
			t.Fatalf("unexpected prompt: %#v", input["prompt"])
		}
		if input["aspect_ratio"] != "9:16" {
			t.Fatalf("unexpected aspect_ratio: %#v", input["aspect_ratio"])
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_image_generate_1","status":"pending","skill_id":"popiskill-image-text2image-basic-v1"}}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"image", "generate",
		"--prompt", "hero poster",
		"--aspect-ratio", "9:16",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "job_image_generate_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestImageGenerateModelOverrideUsesModelsInfer(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/models/infer" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["model_id"] != "gemini-3-pro-image-preview" {
			t.Fatalf("unexpected model_id: %#v", body["model_id"])
		}
		input := body["input"].(map[string]any)
		if input["prompt"] != "hero poster" {
			t.Fatalf("unexpected prompt: %#v", input["prompt"])
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_image_model_override_1","status":"pending"}}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"image", "generate",
		"--prompt", "hero poster",
		"--model", "gemini-3-pro-image-preview",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "job_image_model_override_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
	if data["execution_mode"] != "direct-model-override" {
		t.Fatalf("unexpected execution_mode: %#v", data["execution_mode"])
	}
}

func TestImageParentSugarUsesPositionalPrompt(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/jobs" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["skill_id"] != officialText2ImageSkillID {
			t.Fatalf("unexpected skill_id: %#v", body["skill_id"])
		}
		input := body["input"].(map[string]any)
		if input["prompt"] != "sunset over tokyo" {
			t.Fatalf("unexpected prompt: %#v", input["prompt"])
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_image_parent_1","status":"pending"}}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"image", "sunset over tokyo",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "job_image_parent_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestImageImg2ImgUploadsLocalImageBeforeSubmittingJob(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	sourcePath := filepath.Join(t.TempDir(), "source.png")
	if err := os.WriteFile(sourcePath, []byte("png-body"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	var uploadSeen bool
	var jobSeen bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/artifacts/upload":
			uploadSeen = true
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Fatalf("parse multipart form: %v", err)
			}
			if r.FormValue("role") != "source" {
				t.Fatalf("unexpected upload role: %q", r.FormValue("role"))
			}
			fmt.Fprint(w, `{"ok":true,"data":{"id":"art_img2img_source_1","filename":"source.png","content_type":"image/png","size_bytes":8,"created_at":"2026-04-11T00:00:00Z","url":"https://media.popi.test/source.png","visibility":"unlisted"}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/jobs":
			jobSeen = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode job body: %v", err)
			}
			if body["skill_id"] != officialImage2ImageSkillID {
				t.Fatalf("unexpected skill_id: %#v", body["skill_id"])
			}
			input := body["input"].(map[string]any)
			if input["source_artifact_id"] != "art_img2img_source_1" {
				t.Fatalf("unexpected source_artifact_id: %#v", input["source_artifact_id"])
			}
			if input["prompt"] != "watercolor restyle" {
				t.Fatalf("unexpected prompt: %#v", input["prompt"])
			}
			if input["strength"] != float64(0.6) {
				t.Fatalf("unexpected strength: %#v", input["strength"])
			}
			fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_img2img_1","status":"pending"}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"image", "img2img",
		"--image", sourcePath,
		"--prompt", "watercolor restyle",
		"--strength", "0.6",
	})

	if !uploadSeen || !jobSeen {
		t.Fatalf("expected upload and job submission, uploadSeen=%v jobSeen=%v", uploadSeen, jobSeen)
	}
	data := resp["data"].(map[string]any)
	if data["job_id"] != "job_img2img_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestImageTransformAliasUsesOfficialRuntimeJob(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/jobs" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["skill_id"] != officialImage2ImageSkillID {
			t.Fatalf("unexpected skill_id: %#v", body["skill_id"])
		}
		input := body["input"].(map[string]any)
		if input["image_url"] != "https://example.com/source.png" {
			t.Fatalf("unexpected image_url: %#v", input["image_url"])
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_transform_1","status":"pending"}}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"image", "transform",
		"--image", "https://example.com/source.png",
		"--prompt", "restyle it",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "job_transform_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestVideoGenerateUploadsLocalImageBeforeSubmittingJob(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	sourcePath := filepath.Join(t.TempDir(), "source.png")
	if err := os.WriteFile(sourcePath, []byte("png-body"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	var uploadSeen bool
	var jobSeen bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/artifacts/upload":
			uploadSeen = true
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Fatalf("parse multipart form: %v", err)
			}
			if r.FormValue("role") != "source" {
				t.Fatalf("unexpected upload role: %q", r.FormValue("role"))
			}
			file, _, err := r.FormFile("file")
			if err != nil {
				t.Fatalf("form file: %v", err)
			}
			defer file.Close()
			body, err := io.ReadAll(file)
			if err != nil {
				t.Fatalf("read upload body: %v", err)
			}
			if string(body) != "png-body" {
				t.Fatalf("unexpected upload body: %q", string(body))
			}
			fmt.Fprint(w, `{"ok":true,"data":{"id":"art_source_1","filename":"source.png","content_type":"image/png","size_bytes":8,"created_at":"2026-04-11T00:00:00Z","url":"https://media.popi.test/source.png","visibility":"unlisted"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/skills/"+officialImage2VideoSkillID:
			fmt.Fprint(w, `{"ok":true,"data":{"id":"`+officialImage2VideoSkillID+`","name":"Basic Image2Video","description":"usable runtime","tags":["official"],"version":"1.0.0","model_type":"video","input_schema":{"type":"object"},"output_schema":{"type":"object"}}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/jobs":
			jobSeen = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode job body: %v", err)
			}
			if body["skill_id"] != officialImage2VideoSkillID {
				t.Fatalf("unexpected skill_id: %#v", body["skill_id"])
			}
			input := body["input"].(map[string]any)
			if input["source_artifact_id"] != "art_source_1" {
				t.Fatalf("unexpected source_artifact_id: %#v", input["source_artifact_id"])
			}
			if input["prompt"] != "slow push-in" {
				t.Fatalf("unexpected prompt: %#v", input["prompt"])
			}
			fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_video_generate_1","status":"pending"}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "generate",
		"--image", sourcePath,
		"--prompt", "slow push-in",
	})

	if !uploadSeen || !jobSeen {
		t.Fatalf("expected upload and job submission, uploadSeen=%v jobSeen=%v", uploadSeen, jobSeen)
	}
	data := resp["data"].(map[string]any)
	if data["job_id"] != "job_video_generate_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestVideoParentSugarUsesFromFlag(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/skills/"+officialImage2VideoSkillID {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"`+officialImage2VideoSkillID+`","name":"Basic Image2Video","description":"usable runtime","tags":["official"],"version":"1.0.0","model_type":"video","input_schema":{"type":"object"},"output_schema":{"type":"object"}}}`)
			return
		}
		if r.Method != http.MethodPost || r.URL.Path != "/jobs" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		input := body["input"].(map[string]any)
		if input["image_url"] != "https://example.com/source.png" {
			t.Fatalf("unexpected image_url: %#v", input["image_url"])
		}
		if input["prompt"] != "gentle wind motion" {
			t.Fatalf("unexpected prompt: %#v", input["prompt"])
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_video_parent_1","status":"pending"}}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "--from", "https://example.com/source.png", "gentle wind motion",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "job_video_parent_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestVideoGeneratePromptOnlyReturnsCapabilityUnavailable(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	stdout, stderr, err := executeRootRaw(NewRootCmd("0.test"), []string{
		"video", "generate",
		"--prompt", "make a cinematic teaser",
	})
	if err == nil {
		t.Fatal("expected prompt-only video generate to fail until text2video is ready")
	}
	if stdout != "" || stderr != "" {
		t.Fatalf("expected no stdout/stderr from Execute error path, got stdout=%q stderr=%q", stdout, stderr)
	}
	cliErr, ok := err.(*output.CLIError)
	if !ok {
		t.Fatalf("expected CLIError, got %T", err)
	}
	if cliErr.Code != "CAPABILITY_UNAVAILABLE" {
		t.Fatalf("expected CAPABILITY_UNAVAILABLE, got %#v", cliErr.Code)
	}
}

func TestVideoGeneratePromptOnlyWithModelOverrideUsesModelsInfer(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/models/infer" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["model_id"] != "veo-3-fast" {
			t.Fatalf("unexpected model_id: %#v", body["model_id"])
		}
		input := body["input"].(map[string]any)
		if input["prompt"] != "make a cinematic teaser" {
			t.Fatalf("unexpected prompt: %#v", input["prompt"])
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_video_text2video_override_1","status":"pending"}}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "generate",
		"--prompt", "make a cinematic teaser",
		"--model", "veo-3-fast",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "job_video_text2video_override_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
	if data["execution_mode"] != "direct-model-override" {
		t.Fatalf("unexpected execution_mode: %#v", data["execution_mode"])
	}
}

func TestVideoParentSugarPromptOnlyReturnsCapabilityUnavailable(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	_, _, err := executeRootRaw(NewRootCmd("0.test"), []string{
		"video", "make a cinematic teaser",
	})
	if err == nil {
		t.Fatal("expected prompt-only video sugar to fail until text2video is ready")
	}
	cliErr, ok := err.(*output.CLIError)
	if !ok {
		t.Fatalf("expected CLIError, got %T", err)
	}
	if cliErr.Code != "CAPABILITY_UNAVAILABLE" {
		t.Fatalf("expected CAPABILITY_UNAVAILABLE, got %#v", cliErr.Code)
	}
}

func TestAudioTTSCommandReadsTextFileAndSubmitsJob(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	textPath := filepath.Join(t.TempDir(), "speech.txt")
	if err := os.WriteFile(textPath, []byte("hello from file"), 0o644); err != nil {
		t.Fatalf("write text file: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/models/infer" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["model_id"] != defaultMiniMaxSpeechModelID {
			t.Fatalf("unexpected model_id: %#v", body["model_id"])
		}
		input := body["input"].(map[string]any)
		if input["text"] != "hello from file" {
			t.Fatalf("unexpected text payload: %#v", input["text"])
		}
		if input["format"] != "mp3" {
			t.Fatalf("unexpected format payload: %#v", input["format"])
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_audio_tts_1","status":"pending"}}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"audio", "tts",
		"--text-file", textPath,
		"--format", "mp3",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "job_audio_tts_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
	if data["execution_mode"] != "direct-model-default" {
		t.Fatalf("unexpected execution_mode: %#v", data["execution_mode"])
	}
}

func TestVideoImg2VideoCommandSubmitsOfficialRuntimeJob(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/skills/"+officialImage2VideoSkillID {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"`+officialImage2VideoSkillID+`","name":"Basic Image2Video","description":"usable runtime","tags":["official"],"version":"1.0.0","model_type":"video","input_schema":{"type":"object"},"output_schema":{"type":"object"}}}`)
			return
		}
		if r.Method != http.MethodPost || r.URL.Path != "/jobs" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["skill_id"] != officialImage2VideoSkillID {
			t.Fatalf("unexpected skill_id: %#v", body["skill_id"])
		}
		input := body["input"].(map[string]any)
		if input["image_url"] != "https://example.com/source.png" {
			t.Fatalf("unexpected image_url: %#v", input["image_url"])
		}
		if input["prompt"] != "subtle camera move" {
			t.Fatalf("unexpected prompt: %#v", input["prompt"])
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_img2video_1","status":"pending"}}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "img2video",
		"--image", "https://example.com/source.png",
		"--prompt", "subtle camera move",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "job_img2video_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestVideoFromImageAliasSubmitsOfficialRuntimeJob(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/skills/"+officialImage2VideoSkillID {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"`+officialImage2VideoSkillID+`","name":"Basic Image2Video","description":"usable runtime","tags":["official"],"version":"1.0.0","model_type":"video","input_schema":{"type":"object"},"output_schema":{"type":"object"}}}`)
			return
		}
		if r.Method != http.MethodPost || r.URL.Path != "/jobs" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_from_image_1","status":"pending"}}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "from-image",
		"--image", "https://example.com/source.png",
		"--prompt", "slow push-in",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "job_from_image_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestVideoGenerateDryRunShowsUploadPreflight(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	sourcePath := filepath.Join(t.TempDir(), "source.png")
	if err := os.WriteFile(sourcePath, []byte("png-body"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "generate",
		"--image", sourcePath,
		"--prompt", "gentle motion",
		"--dry-run",
	})

	data := resp["data"].(map[string]any)
	preflight := data["preflight"].(map[string]any)
	if preflight["path"] != "/artifacts/upload" {
		t.Fatalf("unexpected preflight path: %#v", preflight["path"])
	}
	request := data["request"].(map[string]any)
	body := request["body"].(map[string]any)
	input := body["input"].(map[string]any)
	if input["source_artifact_id"] != "(from artifacts.upload)" {
		t.Fatalf("expected placeholder source_artifact_id in dry-run, got %#v", input["source_artifact_id"])
	}
}

func TestImageImg2ImgDryRunShowsUploadPreflight(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	sourcePath := filepath.Join(t.TempDir(), "source.png")
	if err := os.WriteFile(sourcePath, []byte("png-body"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"image", "img2img",
		"--image", sourcePath,
		"--prompt", "watercolor restyle",
		"--dry-run",
	})

	data := resp["data"].(map[string]any)
	preflight := data["preflight"].(map[string]any)
	if preflight["path"] != "/artifacts/upload" {
		t.Fatalf("unexpected preflight path: %#v", preflight["path"])
	}
	request := data["request"].(map[string]any)
	body := request["body"].(map[string]any)
	input := body["input"].(map[string]any)
	if input["source_artifact_id"] != "(from artifacts.upload)" {
		t.Fatalf("expected placeholder source_artifact_id in dry-run, got %#v", input["source_artifact_id"])
	}
	if input["prompt"] != "watercolor restyle" {
		t.Fatalf("unexpected prompt: %#v", input["prompt"])
	}
}

func TestSpeechSynthesizeAliasReadsTextFileAndSubmitsJob(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	textPath := filepath.Join(t.TempDir(), "speech.txt")
	if err := os.WriteFile(textPath, []byte("hello from speech alias"), 0o644); err != nil {
		t.Fatalf("write text file: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/models/infer" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["model_id"] != defaultMiniMaxSpeechModelID {
			t.Fatalf("unexpected model_id: %#v", body["model_id"])
		}
		input := body["input"].(map[string]any)
		if input["text"] != "hello from speech alias" {
			t.Fatalf("unexpected text payload: %#v", input["text"])
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_speech_alias_1","status":"pending"}}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"speech", "synthesize",
		"--text-file", textPath,
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "job_speech_alias_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
	if data["execution_mode"] != "direct-model-default" {
		t.Fatalf("unexpected execution_mode: %#v", data["execution_mode"])
	}
}

func TestSpeechSynthesizeModelOverrideUsesModelsInfer(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/models/infer" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["model_id"] != "speech-2.6" {
			t.Fatalf("unexpected model_id: %#v", body["model_id"])
		}
		input := body["input"].(map[string]any)
		if input["text"] != "hello from override" {
			t.Fatalf("unexpected text payload: %#v", input["text"])
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_speech_model_override_1","status":"pending"}}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"speech", "synthesize",
		"--text", "hello from override",
		"--model", "speech-2.6",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "job_speech_model_override_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
	if data["execution_mode"] != "direct-model-override" {
		t.Fatalf("unexpected execution_mode: %#v", data["execution_mode"])
	}
}

func TestMusicGenerateUsesDefaultMiniMaxModel(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/models/infer" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["model_id"] != defaultMiniMaxMusicModelID {
			t.Fatalf("unexpected model_id: %#v", body["model_id"])
		}
		input := body["input"].(map[string]any)
		if input["prompt"] != "Upbeat pop" {
			t.Fatalf("unexpected prompt: %#v", input["prompt"])
		}
		if input["lyrics"] != "La la la" {
			t.Fatalf("unexpected lyrics: %#v", input["lyrics"])
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_music_generate_1","status":"pending"}}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"music", "generate",
		"--prompt", "Upbeat pop",
		"--lyrics", "La la la",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "job_music_generate_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
	if data["execution_mode"] != "direct-model-default" {
		t.Fatalf("unexpected execution_mode: %#v", data["execution_mode"])
	}
}

func TestMusicRootSugarUsesPositionalPromptAndInstrumental(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/models/infer" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		input := body["input"].(map[string]any)
		if input["prompt"] != "Warm morning folk" {
			t.Fatalf("unexpected prompt: %#v", input["prompt"])
		}
		if input["instrumental"] != true {
			t.Fatalf("unexpected instrumental flag: %#v", input["instrumental"])
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_music_root_1","status":"pending"}}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"music", "Warm morning folk",
		"--instrumental",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "job_music_root_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestMusicGenerateLyricsOptimizerConflictsWithLyrics(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())

	_, _, err := executeRootRaw(NewRootCmd("0.test"), []string{
		"music", "generate",
		"--prompt", "Upbeat pop",
		"--lyrics", "La la la",
		"--lyrics-optimizer",
	})
	if err == nil {
		t.Fatal("expected conflicting lyrics flags to fail")
	}
	cliErr, ok := err.(*output.CLIError)
	if !ok {
		t.Fatalf("expected CLIError, got %T", err)
	}
	if cliErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %#v", cliErr.Code)
	}
}

func TestMusicGenerateDryRunLoadsLyricsFile(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())

	lyricsPath := filepath.Join(t.TempDir(), "lyrics.txt")
	if err := os.WriteFile(lyricsPath, []byte("line one\nline two"), 0o644); err != nil {
		t.Fatalf("write lyrics file: %v", err)
	}

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"music", "generate",
		"--prompt", "Upbeat pop",
		"--lyrics-file", lyricsPath,
		"--dry-run",
	})

	data := resp["data"].(map[string]any)
	if data["model_id"] != defaultMiniMaxMusicModelID {
		t.Fatalf("unexpected model_id: %#v", data["model_id"])
	}
	request := data["request"].(map[string]any)
	body := request["body"].(map[string]any)
	input := body["input"].(map[string]any)
	if input["lyrics"] != "line one\nline two" {
		t.Fatalf("unexpected lyrics payload: %#v", input["lyrics"])
	}
}
