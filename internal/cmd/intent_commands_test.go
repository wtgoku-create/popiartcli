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

func decodeMetadataJSONForTest(t *testing.T, raw any) map[string]any {
	t.Helper()
	text, ok := raw.(string)
	if !ok {
		t.Fatalf("expected metadata string, got %#v", raw)
	}
	var metadata map[string]any
	if err := json.Unmarshal([]byte(text), &metadata); err != nil {
		t.Fatalf("decode metadata json: %v", err)
	}
	return metadata
}

func objectFieldForTest(t *testing.T, raw any, name string) map[string]any {
	t.Helper()
	value, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("expected %s object, got %#v", name, raw)
	}
	return value
}

func assertTaskModelIDOnlyForTest(t *testing.T, body map[string]any, want float64) {
	t.Helper()
	for _, key := range []string{"model", "aiModelCode", "aiModelCodeAlias", "aiModelname"} {
		if _, ok := body[key]; ok {
			t.Fatalf("%s should not be sent: %#v", key, body[key])
		}
	}
	if body["aiModelId"] != want {
		t.Fatalf("unexpected aiModelId: got=%#v want=%#v", body["aiModelId"], want)
	}
}

func assertSpeechTaskModelForTest(t *testing.T, body map[string]any, wantID float64, wantModel string) {
	t.Helper()
	for _, key := range []string{"aiModelCode", "aiModelCodeAlias", "aiModelname"} {
		if _, ok := body[key]; ok {
			t.Fatalf("%s should not be sent: %#v", key, body[key])
		}
	}
	if body["aiModelId"] != wantID {
		t.Fatalf("unexpected aiModelId: got=%#v want=%#v", body["aiModelId"], wantID)
	}
	if body["model"] != wantModel || body["aiPlatform"] != "GATEWAY" || body["origin"] != "web" {
		t.Fatalf("unexpected speech model fields: %#v", body)
	}
}

func TestImageGenerateCommandSubmitsOfficialRuntimeJob(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":101,"code":"seedream-4-5-251128","ratio":["9:16"],"resolution":["2K"],"categories":[{"taskSubType":103}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			assertTaskModelIDOnlyForTest(t, body, 101)
			if body["chatPrompt"] != "hero poster" {
				t.Fatalf("unexpected chatPrompt: %#v", body["chatPrompt"])
			}
			if body["aspectRatio"] != "9:16" || body["ratio"] != "9:16" {
				t.Fatalf("unexpected ratio payload: %#v", body)
			}
			if body["resolution"] != "2K" {
				t.Fatalf("unexpected resolution payload: %#v", body["resolution"])
			}
			if body["width"] != float64(2880) || body["height"] != float64(5120) {
				t.Fatalf("unexpected dimensions payload: %#v", body)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_image_generate_1","status":0,"type":1,"subType":103}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"image", "generate",
		"--prompt", "hero poster",
		"--aspect-ratio", "9:16",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_image_generate_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestImageGenerateDefaultsAspectRatioTo16By9(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":101,"code":"Nano-banana-pro","aiModelCodeAlias":"gemini-3-pro-image-preview","resolution":["2K"],"categories":[{"taskSubType":103}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["aspectRatio"] != "16:9" || body["ratio"] != "16:9" {
				t.Fatalf("expected default 16:9 ratio payload, got %#v", body)
			}
			if body["width"] != float64(5120) || body["height"] != float64(2880) {
				t.Fatalf("unexpected default dimensions payload: %#v", body)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_image_default_ratio_1","status":0,"type":1,"subType":103}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"image", "generate",
		"--prompt", "猫头鹰",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_image_default_ratio_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestImageGenerateAllowsArrayUploadImageLimitFromModelList(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":101,"code":"Nano-banana-pro","aiModelCodeAlias":"gemini-3-pro-image-preview","uploadImageLimit":[0,4],"resolution":["2K"],"categories":[{"taskSubType":103}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			assertTaskModelIDOnlyForTest(t, body, 101)
			if body["chatPrompt"] != "一只小狗" {
				t.Fatalf("unexpected chatPrompt: %#v", body["chatPrompt"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_image_array_limit_1","status":0,"type":1,"subType":103}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/task/detail":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_image_array_limit_1","status":2,"type":1,"subType":103}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/task/downloadUrls":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"downloadUrls":["https://cdn.example.com/dog.png"]}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"image", "generate",
		"--prompt", "一只小狗",
		"--wait",
		"--interval", "1",
	})

	data := resp["data"].(map[string]any)
	if data["status"] != float64(2) {
		t.Fatalf("unexpected status: %#v", data["status"])
	}
	downloadURLs := data["download_urls"].([]any)
	if len(downloadURLs) != 1 || downloadURLs[0] != "https://cdn.example.com/dog.png" {
		t.Fatalf("unexpected download_urls: %#v", data["download_urls"])
	}
}

func TestImageGenerateDownloadSavesResultWithoutReturningURLs(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":101,"code":"seedream-4-5-251128","ratio":["16:9"],"resolution":["2K"],"categories":[{"taskSubType":103}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_image_download_1","status":0,"type":1,"subType":103}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/task/detail":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_image_download_1","status":2,"type":1,"subType":103}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/task/downloadUrls":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"downloadUrls":["`+serverURLForHost(r.Host)+`/download/result.png"]}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/download/result.png":
			_, _ = w.Write([]byte("image-body"))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	dir := filepath.Join(t.TempDir(), "downloads")
	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"image", "generate",
		"--prompt", "一只小狗",
		"--download",
		"--dir", dir,
		"--interval", "1",
	})

	data := resp["data"].(map[string]any)
	if _, ok := data["download_urls"]; ok {
		t.Fatalf("download_urls should be omitted when --download is set: %#v", data["download_urls"])
	}
	if data["artifacts_downloaded"] != float64(1) {
		t.Fatalf("unexpected artifacts_downloaded: %#v", data["artifacts_downloaded"])
	}
	files := data["files"].([]any)
	if len(files) != 1 {
		t.Fatalf("unexpected files: %#v", data["files"])
	}
	file := files[0].(map[string]any)
	if _, ok := file["url"]; ok {
		t.Fatalf("file url should be omitted when --download is set: %#v", file)
	}
	if file["saved_to"] != filepath.Join(dir, "result.png") {
		t.Fatalf("unexpected saved_to: %#v", file["saved_to"])
	}
	body, err := os.ReadFile(filepath.Join(dir, "result.png"))
	if err != nil {
		t.Fatalf("read downloaded result: %v", err)
	}
	if string(body) != "image-body" {
		t.Fatalf("unexpected downloaded body: %q", string(body))
	}
}

func TestImageGenerateDirWithoutDownloadKeepsURLOutput(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":101,"code":"seedream-4-5-251128","ratio":["16:9"],"resolution":["2K"],"categories":[{"taskSubType":103}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_image_dir_only_1","status":0,"type":1,"subType":103}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/task/detail":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_image_dir_only_1","status":2,"type":1,"subType":103}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/task/downloadUrls":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"downloadUrls":["`+serverURLForHost(r.Host)+`/download/result.png"]}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	dir := filepath.Join(t.TempDir(), "dir-only")
	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"image", "generate",
		"--prompt", "一只小狗",
		"--wait",
		"--dir", dir,
		"--interval", "1",
	})

	data := resp["data"].(map[string]any)
	if _, ok := data["artifacts_downloaded"]; ok {
		t.Fatalf("artifacts_downloaded should be omitted without --download: %#v", data["artifacts_downloaded"])
	}
	if _, ok := data["files"]; ok {
		t.Fatalf("files should be omitted without --download: %#v", data["files"])
	}
	downloadURLs := data["download_urls"].([]any)
	if len(downloadURLs) != 1 || downloadURLs[0] != serverURLForHost(server.Listener.Addr().String())+"/download/result.png" {
		t.Fatalf("unexpected download_urls: %#v", data["download_urls"])
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("--dir should not create output directory without --download, err=%v", err)
	}
}

func TestVideoGenerateDownloadCompactsLocalUploadOutput(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	sourcePath := filepath.Join(t.TempDir(), "source.jpeg")
	if err := os.WriteFile(sourcePath, []byte("jpeg-body"), 0o600); err != nil {
		t.Fatalf("write source image: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":202,"code":"viduq2-pro","isSupportImages":true,"categories":[{"taskSubType":202}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/media/upload":
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Fatalf("parse upload form: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"media_source_1","filename":"source.jpeg","content_type":"image/jpeg","size_bytes":9,"created_at":"2026-08-11T00:00:00Z","url":"https://media.popi.test/source.jpeg","visibility":"unlisted"}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			images := body["images"].([]any)
			if len(images) != 1 || images[0] != "https://media.popi.test/source.jpeg" {
				t.Fatalf("unexpected images: %#v", body["images"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_video_download_1","status":0,"type":2,"subType":202}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/task/detail":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_video_download_1","status":2,"type":2,"subType":202}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/task/downloadUrls":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"downloadUrls":["`+serverURLForHost(r.Host)+`/download/result.mp4"]}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/download/result.mp4":
			_, _ = w.Write([]byte("video-body"))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	dir := filepath.Join(t.TempDir(), "videos")
	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "generate",
		"--download",
		"--dir", dir,
		"--image", sourcePath,
		"--prompt", "小狗在跑步",
		"--interval", "1",
	})

	data := resp["data"].(map[string]any)
	for _, key := range []string{"download_urls", "source_sources", "source_uploaded_media"} {
		if _, ok := data[key]; ok {
			t.Fatalf("%s should be omitted when --download is set: %#v", key, data[key])
		}
	}
	files := data["files"].([]any)
	if len(files) != 1 {
		t.Fatalf("unexpected files: %#v", data["files"])
	}
	if files[0].(map[string]any)["saved_to"] != filepath.Join(dir, "result.mp4") {
		t.Fatalf("unexpected files: %#v", data["files"])
	}
}

func TestImageGenerateNormalizesAspectRatioFlag(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":101,"code":"seedream-4-5-251128","ratio":["4:5"],"resolution":["1K"],"categories":[{"taskSubType":103}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["aspectRatio"] != "4:5" || body["ratio"] != "4:5" {
				t.Fatalf("unexpected aspect ratio payload: %#v", body)
			}
			if body["resolution"] != "1K" {
				t.Fatalf("unexpected resolution payload: %#v", body["resolution"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_image_generate_ratio_1","status":0,"type":1,"subType":103}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"image", "generate",
		"--prompt", "hero poster",
		"--aspect-ratio", "4x5",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_image_generate_ratio_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestImageGenerateModelOverrideUsesModelsInfer(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":101,"code":"Nano-banana-pro","aiModelCodeAlias":"gemini-3-pro-image-preview","resolution":["2K","4K"],"categories":[{"taskSubType":103}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			assertTaskModelIDOnlyForTest(t, body, 101)
			if body["chatPrompt"] != "hero poster" {
				t.Fatalf("unexpected prompt: %#v", body["chatPrompt"])
			}
			if body["resolution"] != "2K" {
				t.Fatalf("unexpected default resolution: %#v", body["resolution"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_image_model_override_1","status":0,"type":1,"subType":103}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"image", "generate",
		"--prompt", "hero poster",
		"--model", "101",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_image_model_override_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
	if data["execution_mode"] != "task-model-override" {
		t.Fatalf("unexpected execution_mode: %#v", data["execution_mode"])
	}
}

func TestImageDescribeReturnsDescriptionPrompt(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":501,"code":"doubao-seed-2-0-lite-260428","name":"Doubao seed 2.0 lite","isSupportImages":true,"categories":[{"taskSubType":501}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			assertTaskModelIDOnlyForTest(t, body, 501)
			images := body["images"].([]any)
			if len(images) != 1 || images[0] != "https://example.com/source.png" {
				t.Fatalf("unexpected images payload: %#v", body["images"])
			}
			if !strings.Contains(body["chatPrompt"].(string), "补充要求：请写成适合文生图反推的 prompt") {
				t.Fatalf("unexpected describe prompt: %#v", body["chatPrompt"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_image_describe_1","status":0,"type":5,"subType":501}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/task/detail":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_image_describe_1","status":2,"type":5,"subType":501,"output_text":"一位年轻女性站在海边木栈道上，逆光，长发被海风吹起，电影感中景。"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/task/downloadUrls":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"downloadUrls":[]}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"image", "describe",
		"--image", "https://example.com/source.png",
		"--prompt", "请写成适合文生图反推的 prompt",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_image_describe_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
	if data["description_prompt"] != "一位年轻女性站在海边木栈道上，逆光，长发被海风吹起，电影感中景。" {
		t.Fatalf("unexpected description_prompt: %#v", data["description_prompt"])
	}
}

func TestImageGenerateAutofillsTaskFieldsFromModelList(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":1,"code":"Nano-banana-pro","name":"Popi Banana Pro","aiModelCodeAlias":["gemini-3-pro-image-preview"],"ratio":["21:9","16:9"],"resolution":["2K","4K"],"categories":[{"taskSubType":103}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			assertTaskModelIDOnlyForTest(t, body, 1)
			if body["aspectRatio"] != "21:9" || body["ratio"] != "21:9" {
				t.Fatalf("unexpected ratio payload: %#v", body)
			}
			if body["resolution"] != "2K" {
				t.Fatalf("unexpected resolution payload: %#v", body["resolution"])
			}
			if body["width"] != float64(6720) || body["height"] != float64(2880) {
				t.Fatalf("unexpected dimensions payload: %#v", body)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_image_autofill_1","status":0,"type":1,"subType":103}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"image", "generate",
		"--prompt", "猫头鹰海报",
		"--model", "1",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_image_autofill_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestImageDescribeHydratesArtifactURLWhenAvailable(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":501,"code":"doubao-seed-2-0-lite-260428","name":"Doubao seed 2.0 lite","isSupportImages":true,"categories":[{"taskSubType":501}]}]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/media/detail":
			if r.URL.Query().Get("id") != "art_source_vision_1" {
				t.Fatalf("unexpected media detail id: %q", r.URL.Query().Get("id"))
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"art_source_vision_1","url":"https://media.popi.test/source-vision.png"}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			images := body["images"].([]any)
			if len(images) != 1 || images[0] != "https://media.popi.test/source-vision.png" {
				t.Fatalf("expected hydrated artifact url, got %#v", body["images"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_image_describe_artifact_1","status":0,"type":5,"subType":501}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/task/detail":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_image_describe_artifact_1","status":2,"type":5,"subType":501,"text":"一张白底产品图，主体居中，柔和棚拍光。"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/task/downloadUrls":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"downloadUrls":[]}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"image", "describe",
		"--source-artifact-id", "art_source_vision_1",
		"--model", "501",
	})

	data := resp["data"].(map[string]any)
	if data["description_prompt"] != "一张白底产品图，主体居中，柔和棚拍光。" {
		t.Fatalf("unexpected description_prompt: %#v", data["description_prompt"])
	}
}

func TestImageDescribeDryRunShowsModelsInferRequest(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api_client/anime/ai/model/list" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("origin"); got != "web" {
			t.Fatalf("expected image describe model list origin=web, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":[{"id":501,"code":"doubao-seed-2-0-lite-260428","name":"Doubao seed 2.0 lite","isSupportImages":true,"categories":[{"taskSubType":501}]}]}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"image", "describe",
		"--image", "https://example.com/source.png",
		"--model", "501",
		"--dry-run",
	})

	data := resp["data"].(map[string]any)
	if data["action"] != "image.describe" {
		t.Fatalf("unexpected action: %#v", data["action"])
	}
	if data["model_id"] != float64(501) {
		t.Fatalf("unexpected model_id: %#v", data["model_id"])
	}
	request := data["request"].(map[string]any)
	if request["path"] != "/api_client/anime/task/create" {
		t.Fatalf("unexpected request path: %#v", request["path"])
	}
}

func TestImageDescribeReturnsModelNotFoundWhenNoDescribeModel(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api_client/anime/ai/model/list" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":[]}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	_, _, err := executeRootRaw(NewRootCmd("0.test"), []string{
		"image", "describe",
		"--image", "https://example.com/source.png",
	})
	if err == nil {
		t.Fatal("expected MODEL_NOT_FOUND error")
	}
	cliErr, ok := err.(*output.CLIError)
	if !ok {
		t.Fatalf("expected CLIError, got %#v", err)
	}
	if cliErr.Code != "MODEL_NOT_FOUND" {
		t.Fatalf("unexpected error code: %#v", cliErr.Code)
	}
}

func TestImageParentSugarUsesPositionalPrompt(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":101,"code":"seedream-4-5-251128","categories":[{"taskSubType":103}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["chatPrompt"] != "sunset over tokyo" {
				t.Fatalf("unexpected prompt: %#v", body["chatPrompt"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_image_parent_1","status":0,"type":1,"subType":103}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"image", "sunset over tokyo",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_image_parent_1" {
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
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			fmt.Fprint(w, `{"ok":true,"data":[{"id":101,"code":"seedream-4-5-251128","isSupportImages":true,"uploadImageLimit":5,"categories":[{"taskSubType":103}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/media/upload":
			uploadSeen = true
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Fatalf("parse multipart form: %v", err)
			}
			fmt.Fprint(w, `{"ok":true,"data":{"id":"media_img2img_source_1","filename":"source.png","content_type":"image/png","size_bytes":8,"created_at":"2026-04-11T00:00:00Z","url":"https://media.popi.test/source.png","visibility":"unlisted"}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			jobSeen = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode task body: %v", err)
			}
			assertTaskModelIDOnlyForTest(t, body, 101)
			images := body["images"].([]any)
			if len(images) != 1 || images[0] != "https://media.popi.test/source.png" {
				t.Fatalf("unexpected images payload: %#v", body["images"])
			}
			if body["chatPrompt"] != "watercolor restyle" {
				t.Fatalf("unexpected chatPrompt: %#v", body["chatPrompt"])
			}
			metadata := decodeMetadataJSONForTest(t, body["metadata"])
			if metadata["strength"] != float64(0.6) {
				t.Fatalf("unexpected strength: %#v", metadata["strength"])
			}
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_img2img_1","status":0,"type":1,"subType":103}}`)
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
	if data["job_id"] != "task_img2img_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
	if data["execution_mode"] != "task-model-default" {
		t.Fatalf("unexpected execution_mode: %#v", data["execution_mode"])
	}
}

func TestImageTransformAliasSubmitsTaskRequest(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":101,"code":"seedream-4-5-251128","isSupportImages":true,"uploadImageLimit":5,"categories":[{"taskSubType":103}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			images := body["images"].([]any)
			if len(images) != 1 || images[0] != "https://example.com/source.png" {
				t.Fatalf("unexpected images payload: %#v", body["images"])
			}
			if body["chatPrompt"] != "restyle it" {
				t.Fatalf("unexpected chatPrompt: %#v", body["chatPrompt"])
			}
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_transform_1","status":0,"type":1,"subType":103}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"image", "transform",
		"--image", "https://example.com/source.png",
		"--prompt", "restyle it",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_transform_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
	if data["execution_mode"] != "task-model-default" {
		t.Fatalf("unexpected execution_mode: %#v", data["execution_mode"])
	}
}

func TestImageImg2ImgUploadsRemoteSourceAndReferenceImagesForFusion(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	var uploadRoles []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			fmt.Fprint(w, `{"ok":true,"data":[{"id":101,"code":"seedream-4-5-251128","isSupportImages":true,"uploadImageLimit":5,"categories":[{"taskSubType":103}]}]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/source.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("source-body"))
		case r.Method == http.MethodGet && r.URL.Path == "/subject.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("subject-body"))
		case r.Method == http.MethodGet && r.URL.Path == "/style.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("style-body"))
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/media/upload":
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Fatalf("parse multipart form: %v", err)
			}
			filename := r.FormValue("filename")
			uploadRoles = append(uploadRoles, filename)
			switch len(uploadRoles) {
			case 1:
				fmt.Fprint(w, `{"ok":true,"data":{"id":"media_source_uploaded","filename":"source.png","content_type":"image/png","size_bytes":11,"created_at":"2026-04-16T00:00:00Z","url":"https://media.popi.test/source.png","visibility":"unlisted"}}`)
			case 2:
				fmt.Fprint(w, `{"ok":true,"data":{"id":"media_ref_subject","filename":"subject.png","content_type":"image/png","size_bytes":12,"created_at":"2026-04-16T00:00:00Z","url":"https://media.popi.test/subject.png","visibility":"unlisted"}}`)
			case 3:
				fmt.Fprint(w, `{"ok":true,"data":{"id":"media_ref_style","filename":"style.png","content_type":"image/png","size_bytes":10,"created_at":"2026-04-16T00:00:00Z","url":"https://media.popi.test/style.png","visibility":"unlisted"}}`)
			default:
				t.Fatalf("unexpected extra upload filename=%q", filename)
			}
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			images := body["images"].([]any)
			if len(images) != 3 {
				t.Fatalf("unexpected images count: %#v", body["images"])
			}
			baseURL := "http://" + r.Host
			if images[0] != baseURL+"/source.png" || images[1] != baseURL+"/subject.png" || images[2] != baseURL+"/style.png" {
				t.Fatalf("unexpected images payload: %#v", body["images"])
			}
			metadata := decodeMetadataJSONForTest(t, body["metadata"])
			if metadata["negative_prompt"] != "extra people" {
				t.Fatalf("unexpected negative_prompt: %#v", metadata["negative_prompt"])
			}
			if metadata["preserve_composition"] != true {
				t.Fatalf("unexpected preserve_composition: %#v", metadata["preserve_composition"])
			}
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_transform_fusion","status":0,"type":1,"subType":103}}`)
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

	if len(uploadRoles) != 0 {
		t.Fatalf("expected no uploads for remote URLs, got %#v", uploadRoles)
	}
	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_transform_fusion" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestImageTransformModelOverrideSubmitsTaskRequest(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":101,"code":"seedream-4-5-251128","isSupportImages":true,"uploadImageLimit":5,"categories":[{"taskSubType":103}]}]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/media/detail":
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Query().Get("id") {
			case "art_source_1":
				fmt.Fprint(w, `{"ok":true,"data":{"id":"art_source_1","url":"https://media.popi.test/source.png"}}`)
			case "art_ref_1":
				fmt.Fprint(w, `{"ok":true,"data":{"id":"art_ref_1","url":"https://media.popi.test/ref-1.png"}}`)
			case "art_ref_2":
				fmt.Fprint(w, `{"ok":true,"data":{"id":"art_ref_2","url":"https://media.popi.test/ref-2.png"}}`)
			default:
				t.Fatalf("unexpected media detail id: %q", r.URL.Query().Get("id"))
			}
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			assertTaskModelIDOnlyForTest(t, body, 101)
			images := body["images"].([]any)
			if len(images) != 3 || images[0] != "https://media.popi.test/source.png" || images[1] != "https://media.popi.test/ref-1.png" || images[2] != "https://media.popi.test/ref-2.png" {
				t.Fatalf("unexpected images payload: %#v", body["images"])
			}
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_transform_override_1","status":0,"type":1,"subType":103}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"image", "transform",
		"--source-artifact-id", "art_source_1",
		"--identity-reference-artifact-id", "art_ref_1",
		"--style-reference-artifact-id", "art_ref_2",
		"--prompt", "restyle it",
		"--model", "101",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_transform_override_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
	if data["execution_mode"] != "task-model-override" {
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
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			fmt.Fprint(w, `{"ok":true,"data":[{"id":203,"code":"viduq2-pro-fast","isSupportImages":true,"resolution":["720P","1080P"],"categories":[{"taskSubType":202}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/media/upload":
			uploadSeen = true
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Fatalf("parse multipart form: %v", err)
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
			fmt.Fprint(w, `{"ok":true,"data":{"id":"media_source_1","filename":"source.png","content_type":"image/png","size_bytes":8,"created_at":"2026-04-11T00:00:00Z","url":"https://media.popi.test/source.png","visibility":"unlisted"}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			jobSeen = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode task body: %v", err)
			}
			assertTaskModelIDOnlyForTest(t, body, 203)
			images := body["images"].([]any)
			if len(images) != 1 || images[0] != "https://media.popi.test/source.png" {
				t.Fatalf("unexpected images payload: %#v", body["images"])
			}
			if body["chatPrompt"] != "slow push-in" {
				t.Fatalf("unexpected chatPrompt: %#v", body["chatPrompt"])
			}
			if body["subType"] != float64(202) {
				t.Fatalf("unexpected subType: %#v", body["subType"])
			}
			if body["resolution"] != "720P" {
				t.Fatalf("unexpected resolution: %#v", body["resolution"])
			}
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_video_generate_1","status":0,"type":2,"subType":202}}`)
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
	if data["job_id"] != "task_video_generate_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
	if data["execution_mode"] != "task-model-default" {
		t.Fatalf("unexpected execution_mode: %#v", data["execution_mode"])
	}
}

func TestVideoParentSugarUsesFromFlag(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":203,"code":"viduq2-pro-fast","isSupportImages":true,"resolution":["720P","1080P"],"categories":[{"taskSubType":202}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			images := body["images"].([]any)
			if len(images) != 1 || images[0] != "https://example.com/source.png" {
				t.Fatalf("unexpected images payload: %#v", body["images"])
			}
			if body["chatPrompt"] != "gentle wind motion" {
				t.Fatalf("unexpected chatPrompt: %#v", body["chatPrompt"])
			}
			if body["subType"] != float64(202) {
				t.Fatalf("unexpected subType: %#v", body["subType"])
			}
			if body["resolution"] != "720P" {
				t.Fatalf("unexpected resolution: %#v", body["resolution"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_video_parent_1","status":0,"type":2,"subType":202}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "--from", "https://example.com/source.png", "gentle wind motion",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_video_parent_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestVideoGeneratePromptOnlyReturnsCapabilityUnavailable(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	_, _, err := executeRootRaw(NewRootCmd("0.test"), []string{
		"video", "generate",
		"--prompt", "make a cinematic teaser",
	})
	if err == nil {
		t.Fatal("expected prompt-only video generate to be unavailable")
	}
	cliErr, ok := err.(*output.CLIError)
	if !ok || cliErr.Code != "CAPABILITY_UNAVAILABLE" {
		t.Fatalf("unexpected error: %#v", err)
	}
}

func TestVideoGenerateWithPromptEnhancerUsesMainSiteLLMChat(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":203,"code":"viduq2-pro-fast","isSupportImages":true,"categories":[{"taskSubType":202}]},{"id":501,"code":"doubao-seed-2-0-lite-260428","name":"Doubao seed 2.0 lite","isSupportImages":true,"categories":[{"taskSubType":501}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/llmChat":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["model"] != "doubao-seed-2-0-lite-260428" {
				t.Fatalf("unexpected prompt enhancer model: %#v", body["model"])
			}
			if body["aiModelId"] != float64(501) {
				t.Fatalf("unexpected aiModelId: %#v", body["aiModelId"])
			}
			messages := body["messages"].([]any)
			user := messages[1].(map[string]any)
			content := user["content"].([]any)
			if content[0].(map[string]any)["type"] != "text" {
				t.Fatalf("unexpected user content: %#v", content)
			}
			if !strings.Contains(content[0].(map[string]any)["text"].(string), "用户原始意图：让人物轻轻转头，镜头慢慢推进") {
				t.Fatalf("unexpected enhancer prompt: %#v", content[0])
			}
			if content[1].(map[string]any)["image_url"] != "https://example.com/source.png" {
				t.Fatalf("unexpected enhancer image_url: %#v", content[1])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"choices":[{"message":{"content":"保留人物姿态，头发轻微摆动，人物轻轻转头，镜头缓慢推进，背景有自然风动。","role":"assistant"}}],"model":"doubao-seed-2-0-lite-260428","id":"chatcmpl-demo"}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode task body: %v", err)
			}
			assertTaskModelIDOnlyForTest(t, body, 203)
			if body["chatPrompt"] != "保留人物姿态，头发轻微摆动，人物轻轻转头，镜头缓慢推进，背景有自然风动。" {
				t.Fatalf("unexpected enhanced prompt in task request: %#v", body["chatPrompt"])
			}
			images := body["images"].([]any)
			if len(images) != 1 || images[0] != "https://example.com/source.png" {
				t.Fatalf("unexpected task images: %#v", body["images"])
			}
			if body["subType"] != float64(202) {
				t.Fatalf("unexpected subType: %#v", body["subType"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_video_prompt_enhanced_1","status":0,"type":2,"subType":202}}`)
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
		"--prompt-enhancer-model", "501",
		"--model", "203",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_video_prompt_enhanced_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
	if data["resolved_prompt"] != "保留人物姿态，头发轻微摆动，人物轻轻转头，镜头缓慢推进，背景有自然风动。" {
		t.Fatalf("unexpected resolved_prompt: %#v", data["resolved_prompt"])
	}
	enhancement := data["prompt_enhancement"].(map[string]any)
	if enhancement["model_id"] != float64(501) {
		t.Fatalf("unexpected prompt enhancement model_id: %#v", enhancement["model_id"])
	}
	if enhancement["enhanced_prompt"] != data["resolved_prompt"] {
		t.Fatalf("expected prompt enhancement payload to surface enhanced prompt, got %#v", enhancement["enhanced_prompt"])
	}
}

func TestVideoGeneratePromptEnhancerDryRunShowsTwoStageRequests(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api_client/anime/ai/model/list" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":[{"id":203,"code":"viduq2-pro-fast","isSupportImages":true,"categories":[{"taskSubType":202}]},{"id":501,"code":"doubao-seed-2-0-lite-260428","name":"Doubao seed 2.0 lite","isSupportImages":true,"categories":[{"taskSubType":501}]}]}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "generate",
		"--image", "https://example.com/source.png",
		"--prompt", "make it cinematic",
		"--prompt-enhancer-model", "501",
		"--model", "203",
		"--dry-run",
	})

	data := resp["data"].(map[string]any)
	if data["execution_mode"] != "prompt-enhanced-image2video" {
		t.Fatalf("unexpected execution_mode: %#v", data["execution_mode"])
	}
	enhancement := data["prompt_enhancement"].(map[string]any)
	if enhancement["model_id"] != "501" {
		t.Fatalf("unexpected prompt enhancement model_id: %#v", enhancement["model_id"])
	}
	enhancementReq := enhancement["request"].(map[string]any)
	if enhancementReq["path"] != "/api_client/anime/task/llmChat" {
		t.Fatalf("unexpected llmChat dry-run path: %#v", enhancementReq["path"])
	}
	videoGeneration := data["video_generation"].(map[string]any)
	if videoGeneration["model_id"] != float64(203) {
		t.Fatalf("unexpected video_generation model_id: %#v", videoGeneration["model_id"])
	}
	request := videoGeneration["request"].(map[string]any)
	body := request["body"].(map[string]any)
	if body["chatPrompt"] != "(generated by prompt enhancer)" {
		t.Fatalf("unexpected placeholder prompt in dry-run: %#v", body["chatPrompt"])
	}
}

func TestVideoGeneratePromptEnhancerHydratesArtifactURLWhenAvailable(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":203,"code":"viduq2-pro-fast","isSupportImages":true,"categories":[{"taskSubType":202}]},{"id":501,"code":"doubao-seed-2-0-lite-260428","name":"Doubao seed 2.0 lite","isSupportImages":true,"categories":[{"taskSubType":501}]}]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/media/detail":
			if r.URL.Query().Get("id") != "art_source_1" {
				t.Fatalf("unexpected media detail id: %q", r.URL.Query().Get("id"))
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"art_source_1","url":"https://media.popi.test/source.png"}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/llmChat":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			content := body["messages"].([]any)[1].(map[string]any)["content"].([]any)
			if content[1].(map[string]any)["image_url"] != "https://media.popi.test/source.png" {
				t.Fatalf("expected hydrated artifact url, got %#v", content[1])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"choices":[{"message":{"content":"人物稳定站立，镜头轻微前推。","role":"assistant"}}]}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode task body: %v", err)
			}
			if body["chatPrompt"] != "人物稳定站立，镜头轻微前推。" {
				t.Fatalf("unexpected hydrated follow-up prompt: %#v", body["chatPrompt"])
			}
			if body["subType"] != float64(202) {
				t.Fatalf("unexpected subType: %#v", body["subType"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_video_hydrate_1","status":0,"type":2,"subType":202}}`)
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
		"--prompt-enhancer-model", "501",
		"--model", "203",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_video_hydrate_1" {
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
		t.Fatal("expected prompt-only parent video command to be unavailable")
	}
	cliErr, ok := err.(*output.CLIError)
	if !ok || cliErr.Code != "CAPABILITY_UNAVAILABLE" {
		t.Fatalf("unexpected error: %#v", err)
	}
}

func TestVideoActionTransferUsesJimengDreamActorPayload(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":205,"code":"jimeng_dreamactor_m20_gen_video","isSupportImages":true,"isSupportVideos":true,"categories":[{"taskSubType":205}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			assertTaskModelIDOnlyForTest(t, body, 205)
			if body["subType"] != float64(205) {
				t.Fatalf("unexpected subType: %#v", body["subType"])
			}
			images := body["images"].([]any)
			if len(images) != 1 || images[0] != "https://example.com/face.jpg" {
				t.Fatalf("unexpected images payload: %#v", body["images"])
			}
			videos := body["videos"].([]any)
			if len(videos) != 1 || videos[0] != "https://example.com/action.mp4" {
				t.Fatalf("unexpected videos payload: %#v", body["videos"])
			}
			metadata := decodeMetadataJSONForTest(t, body["metadata"])
			if metadata["action"] != "actionGenerate" {
				t.Fatalf("unexpected action metadata: %#v", metadata)
			}
			if metadata["cut_result_first_second_switch"] != true {
				t.Fatalf("unexpected cut switch: %#v", metadata)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_jimeng_action_transfer_1","status":0,"type":2,"subType":205}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
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
	if data["job_id"] != "task_jimeng_action_transfer_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
	if data["execution_mode"] != "task-model-default" {
		t.Fatalf("unexpected execution_mode: %#v", data["execution_mode"])
	}
}

func TestVideoActionTransferDryRunUploadsImageDataURL(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api_client/anime/ai/model/list" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":[{"id":205,"code":"jimeng_dreamactor_m20_gen_video","isSupportImages":true,"isSupportVideos":true,"categories":[{"taskSubType":205}]}]}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

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
	images := body["images"].([]any)
	if len(images) != 1 || images[0] != "(from api_client/media/upload.url)" {
		t.Fatalf("expected uploaded media placeholder, got %#v", body["images"])
	}
	if body["subType"] != float64(205) {
		t.Fatalf("unexpected subType: %#v", body["subType"])
	}
}

func TestVideoSeedanceUsesDefaultModelAndTaskFieldNames(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":203,"code":"huimeng-seedance-2.0","isSupportImages":true,"isSupportVideos":true,"isSupportAudios":true,"videoRatio":["16:9"],"categories":[{"taskSubType":203},{"taskSubType":204}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			assertTaskModelIDOnlyForTest(t, body, 203)
			if body["subType"] != float64(203) {
				t.Fatalf("unexpected subType: %#v", body["subType"])
			}
			if body["chatPrompt"] != "keep the motion style consistent" {
				t.Fatalf("unexpected chatPrompt: %#v", body["chatPrompt"])
			}
			images := body["images"].([]any)
			if len(images) != 1 || images[0] != "https://example.com/frame.jpg" {
				t.Fatalf("unexpected images: %#v", body["images"])
			}
			videos := body["videos"].([]any)
			if len(videos) != 1 || videos[0] != "https://example.com/ref.mp4" {
				t.Fatalf("unexpected videos: %#v", body["videos"])
			}
			voices := body["voices"].([]any)
			if len(voices) != 1 || voices[0] != "https://example.com/ref.mp3" {
				t.Fatalf("unexpected voices: %#v", body["voices"])
			}
			if metadataValue, exists := body["metadata"]; exists && metadataValue != nil {
				metadata := decodeMetadataJSONForTest(t, metadataValue)
				if metadata["return_last_frame"] != true || metadata["generate_audio"] != true {
					t.Fatalf("unexpected metadata: %#v", metadata)
				}
			}
			if body["aspectRatio"] != "16:9" || body["ratio"] != "16:9" {
				t.Fatalf("unexpected ratio payload: %#v", body)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_seedance_1","status":0,"type":2,"subType":203}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "seedance",
		"--prompt", "keep the motion style consistent",
		"--image", "https://example.com/frame.jpg",
		"--video", "https://example.com/ref.mp4",
		"--audio", "https://example.com/ref.mp3",
		"--ratio", "16:9",
		"--return-last-frame",
		"--generate-audio",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_seedance_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
	if data["execution_mode"] != "task-model-default" {
		t.Fatalf("unexpected execution_mode: %#v", data["execution_mode"])
	}
}

func TestVideoSeedanceTextOnlyDryRunReturnsCapabilityUnavailable(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())

	_, _, err := executeRootRaw(NewRootCmd("0.test"), []string{
		"--dry-run",
		"video", "seedance",
		"--prompt", "a cat chasing a butterfly",
		"--size", "720p",
		"--duration", "5",
		"--frames", "120",
		"--ratio", "adaptive",
		"--action", "textGenerate",
		"--seed", "42",
		"--service-tier", "flex",
		"--execution-expires-after", "600",
		"--draft",
		"--tools-json", `[{"type":"camera_control"}]`,
		"--safety-identifier", "safe-user-1",
	})
	if err == nil {
		t.Fatal("expected text-only seedance dry-run to fail")
	}
	cliErr, ok := err.(*output.CLIError)
	if !ok {
		t.Fatalf("expected CLIError, got %T", err)
	}
	if cliErr.Code != "CAPABILITY_UNAVAILABLE" {
		t.Fatalf("unexpected error code: %#v", cliErr.Code)
	}
}

func TestVideoSeedanceDryRunNormalizesFriendlyModelAlias(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api_client/anime/ai/model/list" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":[{"id":203,"code":"huimeng-seedance-2.0","isSupportImages":true,"categories":[{"taskSubType":203}]}]}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"--dry-run",
		"video", "seedance",
		"--prompt", "a cat chasing a butterfly",
		"--image", "https://example.com/frame.jpg",
	})

	data := resp["data"].(map[string]any)
	if data["model_id"] != float64(203) {
		t.Fatalf("unexpected model_id: %#v", data["model_id"])
	}
	request := data["request"].(map[string]any)
	if request["path"] != "/api_client/anime/task/create" {
		t.Fatalf("unexpected request path: %#v", request["path"])
	}
	body := request["body"].(map[string]any)
	if body["aiModelId"] != float64(203) || body["subType"] != float64(203) {
		t.Fatalf("unexpected dry-run body: %#v", body)
	}
}

func TestVideoSeedanceChecksSupportedModelsBeforeSubmittingAlias(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	seenModels := false
	seenSubmit := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			seenModels = true
			fmt.Fprint(w, `{"ok":true,"data":[{"id":203,"code":"huimeng-seedance-2.0","isSupportImages":true,"categories":[{"taskSubType":203}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			seenSubmit = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["aiModelId"] != float64(203) || body["subType"] != float64(203) {
				t.Fatalf("unexpected task body: %#v", body)
			}
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_seedance_alias_1","status":0,"type":2,"subType":203}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "seedance",
		"--prompt", "a cat chasing a butterfly",
		"--image", "https://example.com/frame.jpg",
	})

	if !seenModels || !seenSubmit {
		t.Fatalf("expected model discovery and submit, seenModels=%v seenSubmit=%v", seenModels, seenSubmit)
	}
	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_seedance_alias_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestVideoSeedanceFallsBackToSupportedModelForAlias(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			fmt.Fprint(w, `{"ok":true,"data":[{"id":204,"code":"doubao-seedance-2-0-fast-260128","isSupportImages":true,"categories":[{"taskSubType":203}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["aiModelId"] != float64(204) {
				t.Fatalf("unexpected aiModelId: %#v", body["aiModelId"])
			}
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_seedance_supported_1","status":0,"type":2,"subType":203}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "seedance",
		"--model", "204",
		"--prompt", "a cat chasing a butterfly",
		"--image", "https://example.com/frame.jpg",
	})

	data := resp["data"].(map[string]any)
	if data["model"] != float64(204) || data["execution_mode"] != "task-model-override" {
		t.Fatalf("unexpected task result: %#v", data)
	}
}

func TestVideoSeedanceStartEndFramesKeepsImageDataURLs(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	const firstFrame = "data:image/jpeg;base64,Zmlyc3QtZnJhbWU="
	const lastFrame = "data:image/jpeg;base64,bGFzdC1mcmFtZQ=="

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":204,"code":"huimeng-seedance-2.0","isSupportImages":true,"categories":[{"taskSubType":204}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/media/upload":
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Fatalf("parse multipart form: %v", err)
			}
			filename := r.FormValue("filename")
			switch filename {
			case "image.jpeg", "image.jfif":
				fmt.Fprintf(w, `{"ok":true,"data":{"id":"media_seedance_frame","filename":%q,"content_type":"image/jpeg","size_bytes":12,"created_at":"2026-06-10T00:00:00Z","url":"https://media.popi.test/frame-uploaded.jpg","visibility":"unlisted"}}`, filename)
			default:
				t.Fatalf("unexpected upload filename: %q", filename)
			}
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			assertTaskModelIDOnlyForTest(t, body, 204)
			images := body["images"].([]any)
			if len(images) != 2 || images[0] != "https://media.popi.test/frame-uploaded.jpg" || images[1] != "https://media.popi.test/frame-uploaded.jpg" {
				t.Fatalf("unexpected start/end images: %#v", body["images"])
			}
			if body["subType"] != float64(204) {
				t.Fatalf("unexpected subType: %#v", body["subType"])
			}
			if metadataValue, exists := body["metadata"]; exists && metadataValue != nil {
				metadata := decodeMetadataJSONForTest(t, metadataValue)
				if metadata["action"] != "firstTailGenerate" {
					t.Fatalf("unexpected action metadata: %#v", metadata)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_seedance_start_end_1","status":0,"type":2,"subType":204}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "seedance",
		"--prompt", "animate between the first and last frame",
		"--image", firstFrame,
		"--image", lastFrame,
		"--action", "firstTailGenerate",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_seedance_start_end_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestVideoSeedanceLastFrameFlagAppendsSecondImage(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list" {
			fmt.Fprint(w, `{"ok":true,"data":[{"id":204,"code":"huimeng-seedance-2.0","isSupportImages":true,"uploadImageLimit":2,"categories":[{"taskSubType":204}]}]}`)
			return
		}
		if r.Method != http.MethodPost || r.URL.Path != "/api_client/anime/task/create" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		images := body["images"].([]any)
		if len(images) != 2 || images[0] != "https://example.com/first.jpg" || images[1] != "https://example.com/last.jpg" {
			t.Fatalf("unexpected images: %#v", body["images"])
		}
		metadata := decodeMetadataJSONForTest(t, body["metadata"])
		if metadata["action"] != "firstTailGenerate" {
			t.Fatalf("unexpected action metadata: %#v", metadata)
		}
		fmt.Fprint(w, `{"ok":true,"data":{"id":"task_seedance_last_frame_1","status":0,"type":2,"subType":204}}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "seedance",
		"--prompt", "transition smoothly",
		"--image", "https://example.com/first.jpg",
		"--last-frame", "https://example.com/last.jpg",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_seedance_last_frame_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestVideoSeedanceImageModeDoesNotRequirePrompt(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":203,"code":"huimeng-seedance-2.0","isSupportImages":true,"categories":[{"taskSubType":203}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if _, ok := body["chatPrompt"]; ok {
				t.Fatalf("did not expect empty prompt to be sent: %#v", body["chatPrompt"])
			}
			images := body["images"].([]any)
			if len(images) != 1 || images[0] != "https://example.com/frame.jpg" {
				t.Fatalf("unexpected images: %#v", body["images"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_seedance_image_1","status":0,"type":2,"subType":203}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "seedance",
		"--image", "https://example.com/frame.jpg",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_seedance_image_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestVideoSeedanceWaitPollsTaskAndSurfacesDownloadURL(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	pollCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			fmt.Fprint(w, `{"ok":true,"data":[{"id":203,"code":"huimeng-seedance-2.0","isSupportImages":true,"categories":[{"taskSubType":203}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			assertTaskModelIDOnlyForTest(t, body, 203)
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_seedance_wait_1","status":0,"type":2,"subType":203}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/task/detail":
			pollCount++
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_seedance_wait_1","status":2,"type":2,"subType":203}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/task/downloadUrls":
			fmt.Fprint(w, `{"ok":true,"data":{"downloadUrls":["https://cdn.example.com/video.mp4"]}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "seedance",
		"--image", "https://example.com/frame.jpg",
		"--return-last-frame",
		"--wait",
		"--interval", "1",
	})

	if pollCount != 1 {
		t.Fatalf("unexpected poll count: %d", pollCount)
	}
	data := resp["data"].(map[string]any)
	if data["status"] != float64(2) {
		t.Fatalf("unexpected status: %#v", data["status"])
	}
	downloadURLs := data["download_urls"].([]any)
	if len(downloadURLs) != 1 || downloadURLs[0] != "https://cdn.example.com/video.mp4" {
		t.Fatalf("unexpected download_urls: %#v", data["download_urls"])
	}
}

func TestVideoSeedanceRejectsAudioWithoutImageOrVideo(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())

	_, _, err := executeRootRaw(NewRootCmd("0.test"), []string{
		"--dry-run",
		"video", "seedance",
		"--audio", "https://example.com/ref.mp3",
	})
	if err == nil {
		t.Fatal("expected audio-only Seedance input to fail")
	}
}

func TestVideoSeedanceRejectsInvalidToolsJSON(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())

	_, _, err := executeRootRaw(NewRootCmd("0.test"), []string{
		"--dry-run",
		"video", "seedance",
		"--prompt", "a cat chasing a butterfly",
		"--tools-json", `{"type":"camera_control"}`,
	})
	if err == nil {
		t.Fatal("expected invalid tools-json to fail")
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
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":301,"code":"speech-2.8-hd","categories":[{"taskSubType":301}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			assertSpeechTaskModelForTest(t, body, 301, "speech-2.8-hd")
			if body["chatPrompt"] != "hello from file" {
				t.Fatalf("unexpected text payload: %#v", body["chatPrompt"])
			}
			if body["voiceId"] != "male-qn-qingse" {
				t.Fatalf("unexpected default voiceId: %#v", body["voiceId"])
			}
			if body["origin"] != "web" || body["model"] != "speech-2.8-hd" || body["aiPlatform"] != "GATEWAY" {
				t.Fatalf("unexpected main site fields: %#v", body)
			}
			for _, key := range []string{"metadata", "projectId", "styleId", "width", "height"} {
				if _, ok := body[key]; ok {
					t.Fatalf("%s should be omitted from speech request: %#v", key, body[key])
				}
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_audio_tts_1","status":0,"type":3,"subType":301}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"audio", "tts",
		"--text-file", textPath,
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_audio_tts_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
	if data["execution_mode"] != "task-model-default" {
		t.Fatalf("unexpected execution_mode: %#v", data["execution_mode"])
	}
}

func TestAudioTTSAutofillsTaskFieldsFromModelList(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":301,"code":"speech-2.8-hd","name":"Popi Speech HD","aiModelCodeAlias":["minimax-speech-2.8-hd"],"categories":[{"taskSubType":301}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			assertSpeechTaskModelForTest(t, body, 301, "speech-2.8-hd")
			if body["aiModelId"] != float64(301) {
				t.Fatalf("unexpected aiModelId: %#v", body["aiModelId"])
			}
			if body["voiceId"] != "female_01" {
				t.Fatalf("unexpected voiceId: %#v", body["voiceId"])
			}
			extraTaskParams := objectFieldForTest(t, body["extraTaskParams"], "extraTaskParams")
			if extraTaskParams["speed"] != float64(1.2) || extraTaskParams["vol"] != float64(0.8) || extraTaskParams["pitch"] != float64(0) {
				t.Fatalf("unexpected extraTaskParams: %#v", extraTaskParams)
			}
			if extraTaskParams["language_boost"] != "Chinese" {
				t.Fatalf("unexpected language_boost: %#v", extraTaskParams["language_boost"])
			}
			voiceSetting := objectFieldForTest(t, extraTaskParams["voice_setting"], "voice_setting")
			if voiceSetting["emotion"] != "fearful" {
				t.Fatalf("unexpected voice emotion: %#v", voiceSetting)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_audio_autofill_1","status":0,"type":3,"subType":301}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"audio", "tts",
		"--model", "301",
		"--text", "你好，世界",
		"--voice", "female_01",
		"--language", "Chinese",
		"--emotion", "fearful",
		"--speed", "1.2",
		"--volume", "0.8",
		"--pitch", "0",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_audio_autofill_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestSpeechRejectsUnsupportedMainSiteFlags(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())

	_, _, err := executeRootRaw(NewRootCmd("0.test"), []string{
		"speech", "synthesize",
		"--text", "hello",
		"--format", "mp3",
	})
	if err == nil {
		t.Fatal("expected unsupported speech flag to fail")
	}
	cliErr, ok := err.(*output.CLIError)
	if !ok {
		t.Fatalf("expected CLIError, got %T", err)
	}
	if cliErr.Code != "VALIDATION_ERROR" || cliErr.Details["flag"] != "--format" {
		t.Fatalf("unexpected error: %#v", cliErr)
	}
}

func TestVideoImg2VideoCommandSubmitsTaskRequest(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":203,"code":"viduq2-pro-fast","isSupportImages":true,"categories":[{"taskSubType":202}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			images := body["images"].([]any)
			if len(images) != 1 || images[0] != "https://example.com/source.png" {
				t.Fatalf("unexpected images payload: %#v", body["images"])
			}
			if body["chatPrompt"] != "subtle camera move" {
				t.Fatalf("unexpected chatPrompt: %#v", body["chatPrompt"])
			}
			if body["subType"] != float64(202) {
				t.Fatalf("unexpected subType: %#v", body["subType"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_img2video_1","status":0,"type":2,"subType":202}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "img2video",
		"--image", "https://example.com/source.png",
		"--prompt", "subtle camera move",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_img2video_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestVideoImg2VideoAutofillsTaskFieldsFromModelList(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":27,"code":"kling-video-omni","name":"Kling Video O1","aiModelCodeAlias":["kling-video-o1"],"isSupportImages":true,"uploadImageLimit":1,"ratio":["16:9","9:16"],"resolution":["720P","1080P","4K"],"duration":[5],"categories":[{"taskSubType":202},{"taskSubType":203},{"taskSubType":204}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			assertTaskModelIDOnlyForTest(t, body, 27)
			if body["aiModelId"] != float64(27) {
				t.Fatalf("unexpected aiModelId: %#v", body["aiModelId"])
			}
			if body["subType"] != float64(203) {
				t.Fatalf("unexpected subType: %#v", body["subType"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_video_autofill_1","status":0,"type":2,"subType":203}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "img2video",
		"--model", "27",
		"--image", "https://example.com/source.png",
		"--prompt", "subtle camera move",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_video_autofill_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestVideoGenerateStartEndFramesSubmitsTaskRequest(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			fmt.Fprint(w, `{"ok":true,"data":[{"id":27,"code":"kling-video-omni","isSupportImages":true,"uploadImageLimit":2,"ratio":["16:9"],"resolution":["720P"],"duration":[5],"categories":[{"taskSubType":204}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			assertTaskModelIDOnlyForTest(t, body, 27)
			if body["subType"] != float64(204) {
				t.Fatalf("unexpected subType: %#v", body["subType"])
			}
			images := body["images"].([]any)
			if len(images) != 2 || images[0] != "https://example.com/first.png" || images[1] != "https://example.com/last.png" {
				t.Fatalf("unexpected start/end images: %#v", body["images"])
			}
			metadata := decodeMetadataJSONForTest(t, body["metadata"])
			if metadata["action"] != "firstTailGenerate" {
				t.Fatalf("unexpected metadata action: %#v", metadata)
			}
			if body["resolution"] != "720P" || body["duration"] != float64(5) {
				t.Fatalf("unexpected size/duration: %#v", body)
			}
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_video_start_end_1","status":0,"type":2,"subType":204}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "generate",
		"--image", "https://example.com/first.png",
		"--last-frame", "https://example.com/last.png",
		"--prompt", "transition naturally",
		"--duration", "5",
		"--size", "720p",
		"--model", "27",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_video_start_end_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestVideoGenerateStartEndFramesUploadsLocalFrames(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	tempDir := t.TempDir()
	firstPath := filepath.Join(tempDir, "first.png")
	lastPath := filepath.Join(tempDir, "last.png")
	if err := os.WriteFile(firstPath, []byte("first-frame"), 0o644); err != nil {
		t.Fatalf("write first frame: %v", err)
	}
	if err := os.WriteFile(lastPath, []byte("last-frame"), 0o644); err != nil {
		t.Fatalf("write last frame: %v", err)
	}

	uploadCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			fmt.Fprint(w, `{"ok":true,"data":[{"id":27,"code":"kling-video-omni","isSupportImages":true,"uploadImageLimit":2,"resolution":["720P"],"duration":[5],"categories":[{"taskSubType":204}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/media/upload":
			uploadCalls++
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Fatalf("parse multipart form: %v", err)
			}
			switch uploadCalls {
			case 1:
				fmt.Fprint(w, `{"ok":true,"data":{"id":"media_first_1","filename":"first.png","content_type":"image/png","size_bytes":11,"created_at":"2026-05-15T00:00:00Z","url":"https://media.popi.test/first.png","visibility":"unlisted"}}`)
			case 2:
				fmt.Fprint(w, `{"ok":true,"data":{"id":"media_last_1","filename":"last.png","content_type":"image/png","size_bytes":10,"created_at":"2026-05-15T00:00:00Z","url":"https://media.popi.test/last.png","visibility":"unlisted"}}`)
			default:
				t.Fatalf("unexpected upload call %d", uploadCalls)
			}
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			assertTaskModelIDOnlyForTest(t, body, 27)
			images := body["images"].([]any)
			if len(images) != 2 || images[0] != "https://media.popi.test/first.png" || images[1] != "https://media.popi.test/last.png" {
				t.Fatalf("unexpected start/end images: %#v", body["images"])
			}
			metadata := decodeMetadataJSONForTest(t, body["metadata"])
			if metadata["action"] != "firstTailGenerate" {
				t.Fatalf("unexpected metadata action: %#v", metadata)
			}
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_video_start_end_upload_1","status":0,"type":2,"subType":204}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "generate",
		"--image", firstPath,
		"--last-frame", lastPath,
		"--prompt", "A little girl grows up.",
		"--duration", "5",
		"--size", "720P",
		"--model", "27",
	})

	if uploadCalls != 2 {
		t.Fatalf("expected two uploads, got %d", uploadCalls)
	}
	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_video_start_end_upload_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestVideoFromImageAliasSubmitsOfficialRuntimeJob(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":27,"code":"kling-video-omni","name":"Kling Video O1","aiModelCodeAlias":["kling-video-o1"],"isSupportImages":true,"uploadImageLimit":1,"ratio":["16:9","9:16"],"resolution":["720P","1080P","4K"],"duration":[5],"categories":[{"taskSubType":202},{"taskSubType":203},{"taskSubType":204}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			assertTaskModelIDOnlyForTest(t, body, 27)
			if body["aiModelId"] != float64(27) {
				t.Fatalf("unexpected aiModelId: %#v", body["aiModelId"])
			}
			if body["subType"] != float64(203) {
				t.Fatalf("unexpected subType: %#v", body["subType"])
			}
			if body["aspectRatio"] != "16:9" || body["ratio"] != "16:9" {
				t.Fatalf("unexpected ratio payload: %#v", body)
			}
			if body["resolution"] != "720P" {
				t.Fatalf("unexpected resolution: %#v", body["resolution"])
			}
			if body["duration"] != float64(5) {
				t.Fatalf("unexpected duration: %#v", body["duration"])
			}
			images := body["images"].([]any)
			if len(images) != 1 || images[0] != "https://example.com/source.png" {
				t.Fatalf("unexpected images payload: %#v", body["images"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_video_autofill_1","status":0,"type":2,"subType":203}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "img2video",
		"--model", "27",
		"--image", "https://example.com/source.png",
		"--prompt", "subtle camera move",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_video_autofill_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestVideoImg2VideoAllowsZeroUploadImageLimitFromModelList(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":48,"code":"kling-v3","isSupportImages":true,"uploadImageLimit":0,"resolution":["1K","2K"],"duration":[5,6,7,8],"categories":[{"taskSubType":202}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			images := body["images"].([]any)
			if len(images) != 1 || images[0] != "https://example.com/source.png" {
				t.Fatalf("unexpected images payload: %#v", body["images"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_kling_v3_1","status":0,"type":2,"subType":202}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "img2video",
		"--model", "48",
		"--image", "https://example.com/source.png",
		"--prompt", "subtle camera move",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_kling_v3_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestVideoFromImageAliasSubmitsTaskRequest(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":203,"code":"viduq2-pro-fast","isSupportImages":true,"categories":[{"taskSubType":202}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["subType"] != float64(202) {
				t.Fatalf("unexpected subType: %#v", body["subType"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_from_image_1","status":0,"type":2,"subType":202}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "from-image",
		"--image", "https://example.com/source.png",
		"--prompt", "slow push-in",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_from_image_1" {
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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api_client/anime/ai/model/list" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":[{"id":203,"code":"viduq2-pro-fast","isSupportImages":true,"categories":[{"taskSubType":202}]}]}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"video", "generate",
		"--image", sourcePath,
		"--prompt", "gentle motion",
		"--dry-run",
	})

	data := resp["data"].(map[string]any)
	preflight := data["source_preflight_uploads"].([]any)[0].(map[string]any)
	if preflight["path"] != "/api_client/media/upload" {
		t.Fatalf("unexpected preflight path: %#v", preflight["path"])
	}
	request := data["request"].(map[string]any)
	body := request["body"].(map[string]any)
	images := body["images"].([]any)
	if len(images) != 1 || images[0] != "(from api_client/media/upload.url)" {
		t.Fatalf("unexpected dry-run images payload: %#v", body["images"])
	}
}

func TestImageImg2ImgDryRunShowsUploadPreflight(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	sourcePath := filepath.Join(t.TempDir(), "source.png")
	if err := os.WriteFile(sourcePath, []byte("png-body"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api_client/anime/ai/model/list" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":[{"id":101,"code":"seedream-4-5-251128","isSupportImages":true,"uploadImageLimit":5,"categories":[{"taskSubType":103}]}]}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"image", "img2img",
		"--image", sourcePath,
		"--prompt", "watercolor restyle",
		"--dry-run",
	})

	data := resp["data"].(map[string]any)
	preflight := data["source_preflight_uploads"].([]any)[0].(map[string]any)
	if preflight["path"] != "/api_client/media/upload" {
		t.Fatalf("unexpected preflight path: %#v", preflight["path"])
	}
	request := data["request"].(map[string]any)
	body := request["body"].(map[string]any)
	images := body["images"].([]any)
	if len(images) != 1 || images[0] != "(from api_client/media/upload.url)" {
		t.Fatalf("unexpected dry-run images payload: %#v", body["images"])
	}
	if body["chatPrompt"] != "watercolor restyle" {
		t.Fatalf("unexpected chatPrompt: %#v", body["chatPrompt"])
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
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":301,"code":"speech-2.8-hd","categories":[{"taskSubType":301}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["chatPrompt"] != "hello from speech alias" {
				t.Fatalf("unexpected text payload: %#v", body["chatPrompt"])
			}
			if body["voiceId"] != "male-qn-qingse" {
				t.Fatalf("unexpected default voiceId: %#v", body["voiceId"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_speech_alias_1","status":0,"type":3,"subType":301}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"speech", "synthesize",
		"--text-file", textPath,
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_speech_alias_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
	if data["execution_mode"] != "task-model-default" {
		t.Fatalf("unexpected execution_mode: %#v", data["execution_mode"])
	}
}

func TestSpeechSynthesizeModelOverrideUsesModelsInfer(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":301,"code":"speech-2.6","categories":[{"taskSubType":301}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			assertSpeechTaskModelForTest(t, body, 301, "speech-2.6")
			if body["chatPrompt"] != "hello from override" {
				t.Fatalf("unexpected text payload: %#v", body["chatPrompt"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_speech_model_override_1","status":0,"type":3,"subType":301}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"speech", "synthesize",
		"--text", "hello from override",
		"--model", "301",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_speech_model_override_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
	if data["execution_mode"] != "task-model-override" {
		t.Fatalf("unexpected execution_mode: %#v", data["execution_mode"])
	}
}

func TestMusicGenerateUsesDefaultMiniMaxModel(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":401,"code":"music-2.6","categories":[{"taskSubType":304}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			assertTaskModelIDOnlyForTest(t, body, 401)
			if body["chatPrompt"] != "Upbeat pop" {
				t.Fatalf("unexpected chatPrompt: %#v", body["chatPrompt"])
			}
			extraTaskParams := objectFieldForTest(t, body["extraTaskParams"], "extraTaskParams")
			if extraTaskParams["lyrics"] != "La la la" {
				t.Fatalf("unexpected lyrics payload: %#v", extraTaskParams["lyrics"])
			}
			assetDraft := objectFieldForTest(t, body["assetDraft"], "assetDraft")
			if assetDraft["title"] != "Upbeat pop" {
				t.Fatalf("unexpected asset title: %#v", assetDraft["title"])
			}
			if body["subType"] != float64(304) {
				t.Fatalf("unexpected subType: %#v", body["subType"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_music_generate_1","status":0,"type":3,"subType":304}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"music", "generate",
		"--prompt", "Upbeat pop",
		"--lyrics", "La la la",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_music_generate_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
	if data["execution_mode"] != "task-model-default" {
		t.Fatalf("unexpected execution_mode: %#v", data["execution_mode"])
	}
}

func TestMusicGenerateAutofillsTaskFieldsFromModelList(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":402,"code":"write_full_song","name":"Write Full Song","aiModelCodeAlias":["write_full_song"],"categories":[{"taskSubType":305}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			assertTaskModelIDOnlyForTest(t, body, 402)
			if body["aiModelId"] != float64(402) {
				t.Fatalf("unexpected aiModelId: %#v", body["aiModelId"])
			}
			if body["subType"] != float64(305) {
				t.Fatalf("unexpected subType: %#v", body["subType"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_music_autofill_1","status":0,"type":3,"subType":305}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"music", "generate",
		"--model", "402",
		"--prompt", "anthemic chorus",
		"--lyrics", "shine on",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_music_autofill_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestMusicRootSugarUsesPositionalPrompt(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":401,"code":"music-2.6","categories":[{"taskSubType":304}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["chatPrompt"] != "Warm morning folk" {
				t.Fatalf("unexpected prompt: %#v", body["chatPrompt"])
			}
			if _, ok := body["metadata"]; ok {
				t.Fatalf("metadata should be omitted from music request: %#v", body["metadata"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_music_root_1","status":0,"type":3,"subType":304}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"music", "Warm morning folk",
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_music_root_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestMusicGenerateRejectsUnsupportedMainSiteFlags(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())

	_, _, err := executeRootRaw(NewRootCmd("0.test"), []string{
		"music", "generate",
		"--prompt", "Upbeat pop",
		"--lyrics", "La la la",
		"--format", "mp3",
	})
	if err == nil {
		t.Fatal("expected unsupported music flag to fail")
	}
	cliErr, ok := err.(*output.CLIError)
	if !ok {
		t.Fatalf("expected CLIError, got %T", err)
	}
	if cliErr.Code != "VALIDATION_ERROR" || cliErr.Details["flag"] != "--format" {
		t.Fatalf("unexpected error: %#v", cliErr)
	}
}

func TestMusicGenerateDryRunLoadsLyricsFile(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	lyricsPath := filepath.Join(t.TempDir(), "lyrics.txt")
	if err := os.WriteFile(lyricsPath, []byte("line one\nline two"), 0o644); err != nil {
		t.Fatalf("write lyrics file: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api_client/anime/ai/model/list" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":[{"id":401,"code":"music-2.6","categories":[{"taskSubType":304}]}]}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"music", "generate",
		"--prompt", "Upbeat pop",
		"--lyrics-file", lyricsPath,
		"--dry-run",
	})

	data := resp["data"].(map[string]any)
	if data["model_id"] != float64(401) {
		t.Fatalf("unexpected model_id: %#v", data["model_id"])
	}
	request := data["request"].(map[string]any)
	body := request["body"].(map[string]any)
	extraTaskParams := objectFieldForTest(t, body["extraTaskParams"], "extraTaskParams")
	if extraTaskParams["lyrics"] != "line one\nline two" {
		t.Fatalf("unexpected lyrics payload: %#v", extraTaskParams["lyrics"])
	}
}

func TestMusicGenerateUsesMinimalMainSiteFields(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api_client/anime/ai/model/list" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":[{"id":401,"code":"music-2.6","categories":[{"taskSubType":304}]}]}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"music", "generate",
		"--prompt", "Warm vlog bed",
		"--title", "测试标题",
		"--dry-run",
	})

	data := resp["data"].(map[string]any)
	request := data["request"].(map[string]any)
	body := request["body"].(map[string]any)
	if body["subType"] != float64(304) {
		t.Fatalf("expected music subType=304, got %#v", body["subType"])
	}
	if body["origin"] != "web" {
		t.Fatalf("expected origin=web, got %#v", body["origin"])
	}
	if body["chatPrompt"] != "Warm vlog bed" {
		t.Fatalf("unexpected chatPrompt: %#v", body["chatPrompt"])
	}
	assetDraft := objectFieldForTest(t, body["assetDraft"], "assetDraft")
	if assetDraft["title"] != "测试标题" {
		t.Fatalf("unexpected title: %#v", assetDraft["title"])
	}
	for _, key := range []string{"metadata", "projectId", "styleId", "width", "height"} {
		if _, ok := body[key]; ok {
			t.Fatalf("%s should be omitted from music request: %#v", key, body[key])
		}
	}
}

func TestMusicGenerateUsesModelBackedSubType305WhenOnly305Supported(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api_client/anime/ai/model/list" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":[{"id":402,"code":"write_full_song","categories":[{"taskSubType":305}]}]}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"music", "generate",
		"--model", "402",
		"--prompt", "anthemic chorus",
		"--lyrics", "shine on",
		"--dry-run",
	})

	data := resp["data"].(map[string]any)
	request := data["request"].(map[string]any)
	body := request["body"].(map[string]any)
	if body["subType"] != float64(305) {
		t.Fatalf("expected music subType=305, got %#v", body["subType"])
	}
}
