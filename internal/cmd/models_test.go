package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
)

func TestRequiredRouteKeyAcceptsLegacySkillType(t *testing.T) {
	cmd := &cobra.Command{}
	addRouteKeyFlags(cmd)
	if err := cmd.Flags().Set("skill-type", "image.img2img"); err != nil {
		t.Fatalf("set legacy flag: %v", err)
	}

	got, err := requiredRouteKey(cmd)
	if err != nil {
		t.Fatalf("requiredRouteKey returned error: %v", err)
	}
	if got != "image.img2img" {
		t.Fatalf("expected legacy route key, got %q", got)
	}
}

func TestRequiredRouteKeyRejectsConflictingFlags(t *testing.T) {
	cmd := &cobra.Command{}
	addRouteKeyFlags(cmd)
	if err := cmd.Flags().Set("route", "image.text2image"); err != nil {
		t.Fatalf("set route flag: %v", err)
	}
	if err := cmd.Flags().Set("skill-type", "image.img2img"); err != nil {
		t.Fatalf("set legacy flag: %v", err)
	}

	if _, err := requiredRouteKey(cmd); err == nil {
		t.Fatal("expected conflict error, got nil")
	}
}

func TestModelsRouteOverrideSetSendsRouteKey(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/models/routes/overrides" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body["project_id"] != "proj_demo" {
			t.Fatalf("unexpected project_id: %#v", body["project_id"])
		}
		if body["route_key"] != "image.text2image" {
			t.Fatalf("unexpected route_key: %#v", body["route_key"])
		}
		if body["skill_type"] != "image.text2image" {
			t.Fatalf("unexpected compatibility skill_type: %#v", body["skill_type"])
		}
		if body["model_id"] != "gemini-3-pro-image-preview" {
			t.Fatalf("unexpected model_id: %#v", body["model_id"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"data":{"project_id":"proj_demo","route_key":"image.text2image","model_id":"gemini-3-pro-image-preview"}}`))
	}))
	defer server.Close()

	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"models", "route-override", "set",
		"--project", "proj_demo",
		"--route", "image.text2image",
		"--model", "gemini-3-pro-image-preview",
	})

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %#v", resp["data"])
	}
	if data["route_key"] != "image.text2image" {
		t.Fatalf("unexpected response route_key: %#v", data["route_key"])
	}
}

func TestModelsListFiltersByCapability(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api_client/anime/ai/model/list" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("origin"); got != "web" {
			t.Fatalf("expected origin=web, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"data":[{"id":101,"code":"image-main","name":"Image Main","categories":[{"taskSubType":103}],"isSupportImages":true}]}`))
	}))
	defer server.Close()

	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"models", "list",
		"--capability", "text2image",
	})

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %#v", resp["data"])
	}
	items, ok := data["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("unexpected items payload: %#v", data["items"])
	}
	item := items[0].(map[string]any)
	if item["id"] != float64(101) {
		t.Fatalf("unexpected item payload: %#v", item)
	}
	for _, key := range []string{"code", "name", "ai_model_code_alias"} {
		if _, ok := item[key]; ok {
			t.Fatalf("%s should not be exposed in models list output: %#v", key, item[key])
		}
	}
}

func TestModelsListIncludesUploadLimits(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api_client/anime/ai/model/list" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("origin"); got != "web" {
			t.Fatalf("expected origin=web, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"data":[{"id":101,"code":"video-main","name":"Video Main","categories":[{"taskSubType":202}],"isSupportImages":true,"uploadImageLimit":0,"uploadVideoLimit":2,"uploadAudioLimit":1}]}`))
	}))
	defer server.Close()

	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"models", "list",
		"--capability", "image2video",
	})

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %#v", resp["data"])
	}
	items, ok := data["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("unexpected items payload: %#v", data["items"])
	}
	item := items[0].(map[string]any)
	if item["upload_image_limit"] != float64(0) {
		t.Fatalf("unexpected upload_image_limit: %#v", item["upload_image_limit"])
	}
	if item["upload_video_limit"] != float64(2) {
		t.Fatalf("unexpected upload_video_limit: %#v", item["upload_video_limit"])
	}
	if item["upload_audio_limit"] != float64(1) {
		t.Fatalf("unexpected upload_audio_limit: %#v", item["upload_audio_limit"])
	}
}

func TestModelsRoutesReturnsDefaultModelSelection(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api_client/anime/ai/model/list" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("origin"); got != "web" {
			t.Fatalf("expected origin=web, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"data":[{"id":301,"code":"speech-2.8-hd","categories":[{"taskSubType":301}]},{"id":401,"code":"music-2.6-free","categories":[{"taskSubType":301}]},{"id":501,"code":"doubao-seedance-2-0-260128","categories":[{"taskSubType":203},{"taskSubType":204}]},{"id":601,"code":"jimeng_dreamactor_m20_gen_video","categories":[{"taskSubType":205}]}]}`))
	}))
	defer server.Close()

	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"models", "routes",
		"--route", "video.seedance",
	})

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %#v", resp["data"])
	}
	items, ok := data["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("unexpected items payload: %#v", data["items"])
	}
	item := items[0].(map[string]any)
	if item["command"] != "video.seedance" {
		t.Fatalf("unexpected route summary: %#v", item)
	}
	if item["default_ai_model_code"] != "doubao-seedance-2-0-260128" {
		t.Fatalf("unexpected default model code: %#v", item)
	}
	if item["resolved_ai_model_id"] != "501" {
		t.Fatalf("unexpected resolved model id: %#v", item)
	}
}
