package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSkillsListIncludesBuiltInOfficialImage2VideoSkill(t *testing.T) {
	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{"skills", "list", "--search", "image2video"})
	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	if len(items) == 0 {
		t.Fatal("expected bundled official skills in list")
	}
}

func TestSkillsGetFallsBackToBuiltInOfficialImage2VideoSkill(t *testing.T) {
	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{"skills", "get", officialImage2VideoSkillID})
	data := resp["data"].(map[string]any)
	if data["id"] != officialImage2VideoSkillID {
		t.Fatalf("unexpected skill id: %#v", data["id"])
	}
}

func TestSkillsSchemaOverlaysOfficialImage2VideoPlaceholder(t *testing.T) {
	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{"skills", "schema", officialImage2VideoSkillID})
	data := resp["data"].(map[string]any)
	if _, ok := data["input_schema"].(map[string]any); !ok {
		t.Fatalf("expected input_schema, got %#v", data)
	}
}

func TestRunOfficialImage2VideoUsesFallbackModelForUnsupportedPrimaryDuration(t *testing.T) {
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
			if body["duration"] != float64(5) {
				t.Fatalf("unexpected duration: %#v", body["duration"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_official_image2video_1","status":0,"type":2,"subType":202}}`)
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
	if data["job_id"] != "task_official_image2video_1" {
		t.Fatalf("unexpected job_id: %#v", data["job_id"])
	}
}
