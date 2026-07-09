package popiart

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wtgoku-create/popiartcli/internal/api"
	"github.com/wtgoku-create/popiartcli/internal/output"
)

func TestWaitForTaskReturnsCompletedTask(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api_client/anime/task/detail":
			attempts++
			if attempts == 1 {
				fmt.Fprint(w, `{"ok":true,"data":{"id":"task_1","status":1}}`)
				return
			}
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_1","status":2}}`)
		case "/api_client/anime/task/downloadUrls":
			fmt.Fprint(w, `{"ok":true,"data":{"downloadUrls":["https://cdn.popi.art/result.png"]}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "pk-demo")
	task, err := WaitForTask(context.Background(), client, "task_1", time.Millisecond, 3)
	if err != nil {
		t.Fatalf("WaitForTask returned error: %v", err)
	}
	if task.Identifier() != "task_1" {
		t.Fatalf("unexpected task identifier: %#v", task)
	}
	if len(task.DownloadURLs) != 1 {
		t.Fatalf("unexpected download urls: %#v", task.DownloadURLs)
	}
}

func TestGetTaskDownloadURLsSupportsObjectItemsWithURLField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api_client/anime/task/downloadUrls" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[{"name":"6904.jpeg","url":"https://cdn.popi.art/6904.jpeg"}],"message":"ok","status":"0000"}`)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "pk-demo")
	urls, err := GetTaskDownloadURLs(context.Background(), client, "2317")
	if err != nil {
		t.Fatalf("GetTaskDownloadURLs returned error: %v", err)
	}
	if len(urls) != 1 || urls[0] != "https://cdn.popi.art/6904.jpeg" {
		t.Fatalf("unexpected download urls: %#v", urls)
	}
}

func TestWaitForTaskReturnsFailureDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api_client/anime/task/detail" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":{"id":"task_2","status":-2,"user_error_tip_msg":"insufficient credits"}}`)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "pk-demo")
	_, err := WaitForTask(context.Background(), client, "task_2", time.Millisecond, 1)
	if err == nil {
		t.Fatal("expected JOB_FAILED error, got nil")
	}
	cliErr, ok := err.(*output.CLIError)
	if !ok {
		t.Fatalf("expected CLIError, got %T", err)
	}
	if cliErr.Code != "JOB_FAILED" {
		t.Fatalf("unexpected error code: %q", cliErr.Code)
	}
	if cliErr.Message != "insufficient credits" {
		t.Fatalf("unexpected error message: %q", cliErr.Message)
	}
}

func TestCreateTaskSupportsNumericIdentifier(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api_client/anime/task/create" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":{"id":98765,"status":0,"type":1,"subType":103},"message":"ok","status":"0000"}`)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "pk-demo")
	task, err := CreateTask(context.Background(), client, TaskRequest{Type: 1, SubType: 103})
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}
	if task.Identifier() != "98765" {
		t.Fatalf("unexpected task identifier: %#v", task)
	}
}

func TestCreateTaskSupportsArrayResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api_client/anime/task/create" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[{"id":"task_array_1","status":0,"type":1,"subType":103}],"message":"ok","status":"0000"}`)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "pk-demo")
	task, err := CreateTask(context.Background(), client, TaskRequest{Type: 1, SubType: 103})
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}
	if task.Identifier() != "task_array_1" {
		t.Fatalf("unexpected task identifier: %#v", task)
	}
}

func TestWaitForTaskSupportsStringStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api_client/anime/task/detail":
			fmt.Fprint(w, `{"data":{"id":6001,"status":"2","type":"3","subType":"301"},"message":"ok","status":"0000"}`)
		case "/api_client/anime/task/downloadUrls":
			fmt.Fprint(w, `{"data":{"downloadUrls":["https://cdn.popi.art/result.mp3"]},"message":"ok","status":"0000"}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "pk-demo")
	task, err := WaitForTask(context.Background(), client, "6001", time.Millisecond, 1)
	if err != nil {
		t.Fatalf("WaitForTask returned error: %v", err)
	}
	if task.Identifier() != "6001" || int(task.Status) != 2 || int(task.Type) != 3 || int(task.SubType) != 301 {
		t.Fatalf("unexpected task payload: %#v", task)
	}
}

func TestCreateTaskSerializesMetadataAsJSONString(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api_client/anime/task/create" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		metadata, ok := body["metadata"].(string)
		if !ok {
			t.Fatalf("expected metadata string, got %#v", body["metadata"])
		}
		if metadata != `{"format":"mp3"}` {
			t.Fatalf("unexpected metadata payload: %q", metadata)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":{"id":7001,"status":0,"type":3,"subType":301},"message":"ok","status":"0000"}`)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "pk-demo")
	task, err := CreateTask(context.Background(), client, TaskRequest{
		Type:        3,
		SubType:     301,
		AIModelCode: "speech-2.8-hd",
		AIModelID:   41,
		ChatPrompt:  "hello",
		Metadata:    map[string]any{"format": "mp3"},
	})
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}
	if task.Identifier() != "7001" {
		t.Fatalf("unexpected task identifier: %#v", task)
	}
}

func TestCreateTaskSerializesAlignedTaskFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api_client/anime/task/create" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if _, exists := body["aiPlatform"]; exists {
			t.Fatalf("aiPlatform should not be sent: %#v", body["aiPlatform"])
		}
		for _, key := range []string{"model", "aiModelCode", "aiModelCodeAlias", "aiModelname"} {
			if _, exists := body[key]; exists {
				t.Fatalf("%s should not be sent: %#v", key, body[key])
			}
		}
		expected := map[string]any{
			"projectId":   float64(-1),
			"type":        float64(2),
			"subType":     float64(203),
			"aiModelId":   float64(15),
			"styleId":     float64(0),
			"width":       float64(1280),
			"height":      float64(720),
			"chatPrompt":  "demo prompt",
			"aspectRatio": "16:9",
			"resolution":  "720P",
			"batchSize":   float64(1),
			"duration":    float64(5),
		}
		for key, want := range expected {
			if body[key] != want {
				t.Fatalf("unexpected %s: got=%#v want=%#v", key, body[key], want)
			}
		}
		images, ok := body["images"].([]any)
		if !ok || len(images) != 1 || images[0] != "https://example.com/source.png" {
			t.Fatalf("unexpected images payload: %#v", body["images"])
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":{"id":7002,"status":0,"type":2,"subType":203},"message":"ok","status":"0000"}`)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "pk-demo")
	task, err := CreateTask(context.Background(), client, TaskRequest{
		Type:             2,
		SubType:          203,
		ProjectID:        -1,
		Model:            "viduq2-pro",
		AIModelCode:      "viduq2-pro",
		AIModelCodeAlias: "viduq2-pro",
		AIModelID:        15,
		StyleID:          0,
		Width:            1280,
		Height:           720,
		ChatPrompt:       "demo prompt",
		Images:           []string{"https://example.com/source.png"},
		AspectRatio:      "16:9",
		Resolution:       "720P",
		Duration:         5,
		BatchSize:        1,
	})
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}
	if task.Identifier() != "7002" {
		t.Fatalf("unexpected task identifier: %#v", task)
	}
}
