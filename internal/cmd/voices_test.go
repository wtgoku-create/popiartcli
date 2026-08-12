package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVoicesListReturnsVoiceIDs(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api_client/anime/voiceLibrary/list" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("page") != "1" || r.URL.Query().Get("pageSize") != "20" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":{"pageInfo":{"page":1,"pageSize":20,"pageCount":1,"total":1},"list":[{"id":5,"voiceId":"male-qn-qingse","voiceType":"system","source":"platform","desp":"青涩青年音色","prompt":"青涩青年音色"}]},"message":"ok","status":"0000"}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{"voices", "list"})
	data := resp["data"].(map[string]any)
	if data["default_voice_id"] != "male-qn-qingse" {
		t.Fatalf("unexpected default_voice_id: %#v", data["default_voice_id"])
	}
	items := data["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("unexpected items: %#v", data["items"])
	}
	item := items[0].(map[string]any)
	if item["voice_id"] != "male-qn-qingse" {
		t.Fatalf("unexpected voice_id: %#v", item["voice_id"])
	}
	if item["description"] != "青涩青年音色" {
		t.Fatalf("unexpected description: %#v", item["description"])
	}
}

func TestVoicesListAcceptsPageSizeAlias(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api_client/anime/voiceLibrary/list" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("pageSize") != "50" {
			t.Fatalf("unexpected pageSize query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":{"pageInfo":{"page":1,"pageSize":50,"pageCount":0,"total":0},"list":[]},"message":"ok","status":"0000"}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{"voices", "list", "--pageSize", "50"})
	data := resp["data"].(map[string]any)
	if data["total"] != float64(0) {
		t.Fatalf("unexpected total: %#v", data["total"])
	}
}
