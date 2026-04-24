package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestImageGenerateNormalizesAspectRatioFlag(t *testing.T) {
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
		input := body["input"].(map[string]any)
		if input["aspect_ratio"] != "4:5" {
			t.Fatalf("unexpected aspect_ratio: %#v", input["aspect_ratio"])
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_image_generate_ratio_1","status":"pending"}}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"image", "generate",
		"--prompt", "hero poster",
		"--aspect-ratio", "4x5",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "job_image_generate_ratio_1" {
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

func TestImageDescribeReturnsDescriptionPrompt(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/models/infer":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["model_id"] != "gemini-2.5-flash" {
				t.Fatalf("unexpected model_id: %#v", body["model_id"])
			}
			input := body["input"].(map[string]any)
			if input["image_url"] != "https://example.com/source.png" {
				t.Fatalf("unexpected image_url: %#v", input["image_url"])
			}
			if !strings.Contains(input["prompt"].(string), "补充要求：请写成适合文生图反推的 prompt") {
				t.Fatalf("unexpected describe prompt: %#v", input["prompt"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_image_describe_1","status":"pending"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/jobs/job_image_describe_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_image_describe_1","status":"done","output_text":"一位年轻女性站在海边木栈道上，逆光，长发被海风吹起，电影感中景。"}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"image", "describe",
		"--image", "https://example.com/source.png",
		"--model", "gemini-2.5-flash",
		"--prompt", "请写成适合文生图反推的 prompt",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "job_image_describe_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
	if data["description_prompt"] != "一位年轻女性站在海边木栈道上，逆光，长发被海风吹起，电影感中景。" {
		t.Fatalf("unexpected description_prompt: %#v", data["description_prompt"])
	}
}

func TestImageDescribeHydratesArtifactURLWhenAvailable(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/artifacts/art_source_vision_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"art_source_vision_1","url":"https://media.popi.test/source-vision.png"}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/models/infer":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			input := body["input"].(map[string]any)
			if input["image_url"] != "https://media.popi.test/source-vision.png" {
				t.Fatalf("expected hydrated artifact url, got %#v", input["image_url"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_image_describe_artifact_1","status":"pending"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/jobs/job_image_describe_artifact_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_image_describe_artifact_1","status":"done","text":"一张白底产品图，主体居中，柔和棚拍光。"}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"image", "describe",
		"--source-artifact-id", "art_source_vision_1",
		"--model", "gemini-2.5-flash",
	})

	data := resp["data"].(map[string]any)
	if data["description_prompt"] != "一张白底产品图，主体居中，柔和棚拍光。" {
		t.Fatalf("unexpected description_prompt: %#v", data["description_prompt"])
	}
}

func TestImageDescribeDryRunShowsModelsInferRequest(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"image", "describe",
		"--image", "https://example.com/source.png",
		"--model", "gemini-2.5-flash",
		"--dry-run",
	})

	data := resp["data"].(map[string]any)
	if data["action"] != "image.describe" {
		t.Fatalf("unexpected action: %#v", data["action"])
	}
	if data["model_id"] != "gemini-2.5-flash" {
		t.Fatalf("unexpected model_id: %#v", data["model_id"])
	}
	request := data["request"].(map[string]any)
	if request["path"] != "/models/infer" {
		t.Fatalf("unexpected request path: %#v", request["path"])
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
		if input["image"] != "https://example.com/source.png" {
			t.Fatalf("unexpected image: %#v", input["image"])
		}
		if input["image_url"] != "https://example.com/source.png" {
			t.Fatalf("unexpected image_url: %#v", input["image_url"])
		}
		if input["reference_image_url"] != "https://example.com/source.png" {
			t.Fatalf("unexpected reference_image_url: %#v", input["reference_image_url"])
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

func TestImageImg2ImgUploadsRemoteSourceAndReferenceImagesForFusion(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	var uploadRoles []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/source.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("source-body"))
		case r.Method == http.MethodGet && r.URL.Path == "/subject.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("subject-body"))
		case r.Method == http.MethodGet && r.URL.Path == "/style.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("style-body"))
		case r.Method == http.MethodPost && r.URL.Path == "/artifacts/upload":
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Fatalf("parse multipart form: %v", err)
			}
			role := r.FormValue("role")
			uploadRoles = append(uploadRoles, role)
			switch len(uploadRoles) {
			case 1:
				fmt.Fprint(w, `{"ok":true,"data":{"id":"art_source_uploaded","filename":"source.png","content_type":"image/png","size_bytes":11,"created_at":"2026-04-16T00:00:00Z","visibility":"unlisted"}}`)
			case 2:
				fmt.Fprint(w, `{"ok":true,"data":{"id":"art_ref_subject","filename":"subject.png","content_type":"image/png","size_bytes":12,"created_at":"2026-04-16T00:00:00Z","visibility":"unlisted"}}`)
			case 3:
				fmt.Fprint(w, `{"ok":true,"data":{"id":"art_ref_style","filename":"style.png","content_type":"image/png","size_bytes":10,"created_at":"2026-04-16T00:00:00Z","visibility":"unlisted"}}`)
			default:
				t.Fatalf("unexpected extra upload role=%q", role)
			}
		case r.Method == http.MethodPost && r.URL.Path == "/jobs":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["skill_id"] != officialImage2ImageSkillID {
				t.Fatalf("unexpected skill_id: %#v", body["skill_id"])
			}
			input := body["input"].(map[string]any)
			if input["source_artifact_id"] != "art_source_uploaded" {
				t.Fatalf("unexpected source_artifact_id: %#v", input["source_artifact_id"])
			}
			refs, ok := input["reference_artifact_ids"].([]any)
			if !ok || len(refs) != 2 || refs[0] != "art_ref_subject" || refs[1] != "art_ref_style" {
				t.Fatalf("unexpected reference_artifact_ids: %#v", input["reference_artifact_ids"])
			}
			identityRefs, ok := input["identity_reference_artifact_ids"].([]any)
			if !ok || len(identityRefs) != 1 || identityRefs[0] != "art_ref_subject" {
				t.Fatalf("unexpected identity_reference_artifact_ids: %#v", input["identity_reference_artifact_ids"])
			}
			styleRefs, ok := input["style_reference_artifact_ids"].([]any)
			if !ok || len(styleRefs) != 1 || styleRefs[0] != "art_ref_style" {
				t.Fatalf("unexpected style_reference_artifact_ids: %#v", input["style_reference_artifact_ids"])
			}
			if input["negative_prompt"] != "extra people" {
				t.Fatalf("unexpected negative_prompt: %#v", input["negative_prompt"])
			}
			if input["preserve_composition"] != true {
				t.Fatalf("unexpected preserve_composition: %#v", input["preserve_composition"])
			}
			if _, exists := input["image"]; exists {
				t.Fatalf("did not expect direct image field when fusion uses artifacts: %#v", input["image"])
			}
			fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_transform_fusion","status":"pending"}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"image", "img2img",
		"--image", server.URL + "/source.png",
		"--identity-reference-image", server.URL + "/subject.png",
		"--style-reference-image", server.URL + "/style.png",
		"--prompt", "fuse the subject into the main scene",
		"--negative-prompt", "extra people",
		"--preserve-composition",
	})

	if len(uploadRoles) != 3 {
		t.Fatalf("expected three uploads, got %#v", uploadRoles)
	}
	if uploadRoles[0] != "source" || uploadRoles[1] != "reference" || uploadRoles[2] != "reference" {
		t.Fatalf("unexpected upload roles: %#v", uploadRoles)
	}
	data := resp["data"].(map[string]any)
	if data["job_id"] != "job_transform_fusion" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestImageTransformModelOverrideCanonicalizesImageForModelsInfer(t *testing.T) {
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
		if body["model_id"] != "seedream-4-5-251128" {
			t.Fatalf("unexpected model_id: %#v", body["model_id"])
		}
		input := body["input"].(map[string]any)
		if input["source_artifact_id"] != "art_source_1" {
			t.Fatalf("unexpected source_artifact_id: %#v", input["source_artifact_id"])
		}
		refs, ok := input["reference_artifact_ids"].([]any)
		if !ok || len(refs) != 2 || refs[0] != "art_ref_1" || refs[1] != "art_ref_2" {
			t.Fatalf("unexpected reference_artifact_ids: %#v", input["reference_artifact_ids"])
		}
		if _, exists := input["image"]; exists {
			t.Fatalf("did not expect direct image field in direct infer payload: %#v", input["image"])
		}
		if _, exists := input["image_url"]; exists {
			t.Fatalf("did not expect image_url alias in direct infer payload: %#v", input["image_url"])
		}
		if _, exists := input["reference_image_url"]; exists {
			t.Fatalf("did not expect reference_image_url alias in direct infer payload: %#v", input["reference_image_url"])
		}
		identityRefs, ok := input["identity_reference_artifact_ids"].([]any)
		if !ok || len(identityRefs) != 1 || identityRefs[0] != "art_ref_1" {
			t.Fatalf("unexpected identity_reference_artifact_ids: %#v", input["identity_reference_artifact_ids"])
		}
		styleRefs, ok := input["style_reference_artifact_ids"].([]any)
		if !ok || len(styleRefs) != 1 || styleRefs[0] != "art_ref_2" {
			t.Fatalf("unexpected style_reference_artifact_ids: %#v", input["style_reference_artifact_ids"])
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_transform_override_1","status":"pending"}}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"image", "transform",
		"--source-artifact-id", "art_source_1",
		"--identity-reference-artifact-id", "art_ref_1",
		"--style-reference-artifact-id", "art_ref_2",
		"--prompt", "restyle it",
		"--model", "seedream-4-5-251128",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "job_transform_override_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
	if data["execution_mode"] != "direct-model-override" {
		t.Fatalf("unexpected execution_mode: %#v", data["execution_mode"])
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

func TestVideoGenerateWithPromptEnhancerModelUsesTwoStageModelsInfer(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	modelInferCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/models/infer":
			modelInferCalls++
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			input := body["input"].(map[string]any)
			switch modelInferCalls {
			case 1:
				if body["model_id"] != "gemini-2.5-flash" {
					t.Fatalf("unexpected prompt enhancer model: %#v", body["model_id"])
				}
				if input["image_url"] != "https://example.com/source.png" {
					t.Fatalf("unexpected enhancer image_url: %#v", input["image_url"])
				}
				if !strings.Contains(input["prompt"].(string), "用户原始意图：让人物轻轻转头，镜头慢慢推进") {
					t.Fatalf("unexpected enhancer prompt: %#v", input["prompt"])
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_prompt_enhance_1","status":"pending"}}`)
			case 2:
				if body["model_id"] != "viduq2-pro-fast" {
					t.Fatalf("unexpected video model: %#v", body["model_id"])
				}
				if input["prompt"] != "保留人物姿态，头发轻微摆动，人物轻轻转头，镜头缓慢推进，背景有自然风动。" {
					t.Fatalf("unexpected enhanced prompt in video request: %#v", input["prompt"])
				}
				if input["image_url"] != "https://example.com/source.png" {
					t.Fatalf("unexpected video image_url: %#v", input["image_url"])
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_video_prompt_enhanced_1","status":"pending"}}`)
			default:
				t.Fatalf("unexpected extra models infer call: %d", modelInferCalls)
			}
		case r.Method == http.MethodGet && r.URL.Path == "/jobs/job_prompt_enhance_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_prompt_enhance_1","status":"done","output":{"text":"保留人物姿态，头发轻微摆动，人物轻轻转头，镜头缓慢推进，背景有自然风动。"}}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "generate",
		"--image", "https://example.com/source.png",
		"--prompt", "让人物轻轻转头，镜头慢慢推进",
		"--prompt-enhancer-model", "gemini-2.5-flash",
		"--model", "viduq2-pro-fast",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "job_video_prompt_enhanced_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
	if data["resolved_prompt"] != "保留人物姿态，头发轻微摆动，人物轻轻转头，镜头缓慢推进，背景有自然风动。" {
		t.Fatalf("unexpected resolved_prompt: %#v", data["resolved_prompt"])
	}
	enhancement := data["prompt_enhancement"].(map[string]any)
	if enhancement["job_id"] != "job_prompt_enhance_1" {
		t.Fatalf("unexpected prompt enhancement job_id: %#v", enhancement["job_id"])
	}
	if enhancement["enhanced_prompt"] != data["resolved_prompt"] {
		t.Fatalf("expected prompt enhancement payload to surface enhanced prompt, got %#v", enhancement["enhanced_prompt"])
	}
}

func TestVideoGenerateWithPromptEnhancerUsesArtifactJSONFallback(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	modelInferCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/models/infer":
			modelInferCalls++
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			switch modelInferCalls {
			case 1:
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_prompt_artifact_1","status":"pending"}}`)
			case 2:
				input := body["input"].(map[string]any)
				if input["prompt"] != "人物站定，眼神看向镜头，衣摆和发丝有轻微风动，镜头缓慢推近。" {
					t.Fatalf("unexpected artifact-derived prompt: %#v", input["prompt"])
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_video_from_artifact_prompt_1","status":"pending"}}`)
			default:
				t.Fatalf("unexpected extra models infer call: %d", modelInferCalls)
			}
		case r.Method == http.MethodGet && r.URL.Path == "/jobs/job_prompt_artifact_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_prompt_artifact_1","status":"done","artifact_ids":["art_prompt_json_1"]}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/artifacts/art_prompt_json_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"art_prompt_json_1","filename":"result.json","content_type":"application/json"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/artifacts/art_prompt_json_1/content":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"resolved_prompt":"人物站定，眼神看向镜头，衣摆和发丝有轻微风动，镜头缓慢推近。"}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "generate",
		"--image", "https://example.com/source.png",
		"--prompt", "让人物更自然一点",
		"--prompt-enhancer-model", "kimi-2.5-vision",
		"--model", "viduq2-pro-fast",
	})

	data := resp["data"].(map[string]any)
	if data["resolved_prompt"] != "人物站定，眼神看向镜头，衣摆和发丝有轻微风动，镜头缓慢推近。" {
		t.Fatalf("unexpected resolved_prompt: %#v", data["resolved_prompt"])
	}
}

func TestVideoGeneratePromptEnhancerDryRunShowsTwoStageRequests(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "generate",
		"--image", "https://example.com/source.png",
		"--prompt", "make it cinematic",
		"--prompt-enhancer-model", "gemini-2.5-flash",
		"--model", "viduq2-pro-fast",
		"--dry-run",
	})

	data := resp["data"].(map[string]any)
	if data["execution_mode"] != "prompt-enhanced-image2video" {
		t.Fatalf("unexpected execution_mode: %#v", data["execution_mode"])
	}
	enhancement := data["prompt_enhancement"].(map[string]any)
	if enhancement["model_id"] != "gemini-2.5-flash" {
		t.Fatalf("unexpected prompt enhancement model_id: %#v", enhancement["model_id"])
	}
	videoGeneration := data["video_generation"].(map[string]any)
	if videoGeneration["model_id"] != "viduq2-pro-fast" {
		t.Fatalf("unexpected video_generation model_id: %#v", videoGeneration["model_id"])
	}
	request := videoGeneration["request"].(map[string]any)
	input := request["body"].(map[string]any)["input"].(map[string]any)
	if input["prompt"] != "(generated by prompt enhancer)" {
		t.Fatalf("unexpected placeholder prompt in dry-run: %#v", input["prompt"])
	}
}

func TestVideoGeneratePromptEnhancerHydratesArtifactURLWhenAvailable(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	modelInferCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/artifacts/art_source_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"art_source_1","url":"https://media.popi.test/source.png"}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/models/infer":
			modelInferCalls++
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			input := body["input"].(map[string]any)
			switch modelInferCalls {
			case 1:
				if input["image_url"] != "https://media.popi.test/source.png" {
					t.Fatalf("expected hydrated artifact url, got %#v", input["image_url"])
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_prompt_hydrate_1","status":"pending"}}`)
			case 2:
				if input["prompt"] != "人物稳定站立，镜头轻微前推。" {
					t.Fatalf("unexpected hydrated follow-up prompt: %#v", input["prompt"])
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_video_hydrate_1","status":"pending"}}`)
			default:
				t.Fatalf("unexpected extra models infer call: %d", modelInferCalls)
			}
		case r.Method == http.MethodGet && r.URL.Path == "/jobs/job_prompt_hydrate_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_prompt_hydrate_1","status":"done","output_text":"人物稳定站立，镜头轻微前推。"}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "generate",
		"--source-artifact-id", "art_source_1",
		"--prompt", "轻一点",
		"--prompt-enhancer-model", "gemini-2.5-flash",
		"--model", "viduq2-pro-fast",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "job_video_hydrate_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
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

func TestVideoActionTransferUsesJimengDreamActorPayload(t *testing.T) {
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
		if body["model_id"] != defaultJimengActionTransferModelID {
			t.Fatalf("unexpected model_id: %#v", body["model_id"])
		}
		if body["model_type"] != "video" {
			t.Fatalf("unexpected model_type: %#v", body["model_type"])
		}
		input := body["input"].(map[string]any)
		images := input["images"].([]any)
		if len(images) != 1 || images[0] != "https://example.com/face.jpg" {
			t.Fatalf("unexpected images payload: %#v", input["images"])
		}
		videos := input["videos"].([]any)
		if len(videos) != 1 || videos[0] != "https://example.com/action.mp4" {
			t.Fatalf("unexpected videos payload: %#v", input["videos"])
		}
		metadata := input["metadata"].(map[string]any)
		if metadata["action"] != "actionGenerate" {
			t.Fatalf("unexpected action metadata: %#v", metadata)
		}
		if metadata["cut_result_first_second_switch"] != true {
			t.Fatalf("unexpected cut switch: %#v", metadata)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_jimeng_action_transfer_1","status":"pending"}}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "action-transfer",
		"--image", "https://example.com/face.jpg",
		"--video", "https://example.com/action.mp4",
		"--cut-result-first-second-switch",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "job_jimeng_action_transfer_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
	if data["execution_mode"] != "direct-model-default" {
		t.Fatalf("unexpected execution_mode: %#v", data["execution_mode"])
	}
}

func TestVideoActionTransferStripsJimengImageDataURLPrefix(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())

	imageBase64 := base64.StdEncoding.EncodeToString([]byte("fake-jpeg-body-for-jimeng"))
	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "action-transfer",
		"--image", "data:image/jpeg;base64," + imageBase64,
		"--video", "https://example.com/action.mp4",
		"--dry-run",
	})

	data := resp["data"].(map[string]any)
	request := data["request"].(map[string]any)
	body := request["body"].(map[string]any)
	input := body["input"].(map[string]any)
	images := input["images"].([]any)
	if len(images) != 1 || images[0] != imageBase64 {
		t.Fatalf("expected pure base64 image payload, got %#v", input["images"])
	}
	if strings.HasPrefix(images[0].(string), "data:image/") {
		t.Fatalf("did not expect data URL prefix in Jimeng payload: %#v", images[0])
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
		if input["is_instrumental"] != true {
			t.Fatalf("unexpected instrumental flag: %#v", input["is_instrumental"])
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

func TestMusicGenerateUsesGatewayFieldNames(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"music", "generate",
		"--prompt", "Warm vlog bed",
		"--instrumental",
		"--output-format", "url",
		"--format", "mp3",
		"--sample-rate-hz", "44100",
		"--bitrate", "256000",
		"--dry-run",
	})

	data := resp["data"].(map[string]any)
	request := data["request"].(map[string]any)
	body := request["body"].(map[string]any)
	if body["model_type"] != "music" {
		t.Fatalf("expected music model_type, got %#v", body["model_type"])
	}
	input := body["input"].(map[string]any)
	if input["is_instrumental"] != true {
		t.Fatalf("expected gateway is_instrumental field, got %#v", input)
	}
	if input["output_format"] != "url" {
		t.Fatalf("expected output_format=url, got %#v", input["output_format"])
	}
	if _, ok := input["instrumental"]; ok {
		t.Fatalf("did not expect legacy instrumental field: %#v", input)
	}
	if _, ok := input["format"]; ok {
		t.Fatalf("did not expect top-level format field: %#v", input)
	}
	audioSetting := input["audio_setting"].(map[string]any)
	if audioSetting["format"] != "mp3" || audioSetting["sample_rate"] != float64(44100) || audioSetting["bitrate"] != float64(256000) {
		t.Fatalf("unexpected audio_setting: %#v", audioSetting)
	}
}

func TestMusicCoverRequiresExactlyOneAudioSource(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())

	_, _, err := executeRootRaw(NewRootCmd("0.test"), []string{
		"music", "generate",
		"--model", "music-cover",
		"--prompt", "female pop cover",
		"--dry-run",
	})
	if err == nil {
		t.Fatal("expected missing cover audio source to fail")
	}

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"music", "generate",
		"--model", "music-cover",
		"--prompt", "female pop cover",
		"--audio-url", "https://example.com/demo.mp3",
		"--output-format", "url",
		"--dry-run",
	})
	data := resp["data"].(map[string]any)
	request := data["request"].(map[string]any)
	body := request["body"].(map[string]any)
	input := body["input"].(map[string]any)
	if input["audio_url"] != "https://example.com/demo.mp3" || input["output_format"] != "url" {
		t.Fatalf("unexpected cover payload: %#v", input)
	}
}
