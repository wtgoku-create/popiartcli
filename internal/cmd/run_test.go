package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunBridgeText2ImageRoutesToTaskCreate(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":101,"code":"Nano-banana-pro","resolution":["2K"],"ratio":["16:9"],"categories":[{"taskSubType":103}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["chatPrompt"] != "a cinematic tea shop at sunset" {
				t.Fatalf("unexpected prompt: %#v", body["chatPrompt"])
			}
			if body["subType"] != float64(103) {
				t.Fatalf("unexpected subtype: %#v", body["subType"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_run_text2image_1","status":0,"type":1,"subType":103}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"run", officialText2ImageSkillID,
		"--input", `{"prompt":"a cinematic tea shop at sunset","aspect_ratio":"16:9"}`,
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_run_text2image_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
	if data["requested_skill_id"] != officialText2ImageSkillID {
		t.Fatalf("unexpected requested_skill_id: %#v", data["requested_skill_id"])
	}
}

func TestRunBridgeImg2ImgResolvesArtifactAndImageSources(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":101,"code":"Nano-banana-pro","resolution":["2K"],"ratio":["9:16"],"isSupportImages":true,"categories":[{"taskSubType":103}]}]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/media/detail":
			if got := r.URL.Query().Get("id"); got != "art_ref_1" {
				t.Fatalf("unexpected artifact id lookup: %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"art_ref_1","url":"https://media.example.com/art_ref_1.png"}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			images := body["images"].([]any)
			if len(images) != 2 {
				t.Fatalf("unexpected images payload: %#v", body["images"])
			}
			if images[0] != "https://media.example.com/art_ref_1.png" || images[1] != "https://example.com/reference.png" {
				t.Fatalf("unexpected images order: %#v", images)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_run_img2img_1","status":0,"type":1,"subType":103}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"run", officialImage2ImageSkillID,
		"--input", `{"prompt":"keep subject","aspect_ratio":"9:16","image_url":"https://example.com/reference.png","reference_artifact_ids":["art_ref_1"]}`,
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_run_img2img_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestRunBridgeImage2VideoMapsLegacyInputToTaskCreate(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"id":15,"code":"viduq2-pro","resolution":["720P"],"duration":[5],"isSupportImages":true,"categories":[{"taskSubType":202}]}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["chatPrompt"] != "make it cinematic" {
				t.Fatalf("unexpected prompt: %#v", body["chatPrompt"])
			}
			if body["duration"] != float64(5) {
				t.Fatalf("unexpected duration: %#v", body["duration"])
			}
			images := body["images"].([]any)
			if len(images) != 1 || images[0] != "https://example.com/source.png" {
				t.Fatalf("unexpected images payload: %#v", body["images"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_run_image2video_1","status":0,"type":2,"subType":202}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"run", officialImage2VideoSkillID,
		"--input", `{"reference_image_url":"https://example.com/source.png","seconds":6,"prompt":"make it cinematic"}`,
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_run_image2video_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}

func TestRunBridgeTTSMapsToAudioTask(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

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
			if body["chatPrompt"] != "Hello from skill bridge" {
				t.Fatalf("unexpected text: %#v", body["chatPrompt"])
			}
			if body["voiceId"] != "female_01" {
				t.Fatalf("unexpected voiceId: %#v", body["voiceId"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_run_tts_1","status":0,"type":3,"subType":301}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"run", officialTTSMultimodelSkillID,
		"--input", `{"text":"Hello from skill bridge","voice":"female_01","format":"mp3"}`,
	})

	data := resp["data"].(map[string]any)
	if data["job_id"] != "task_run_tts_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}
