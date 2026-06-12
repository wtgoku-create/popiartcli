package popiart

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wtgoku-create/popiartcli/internal/api"
)

func TestFetchCurrentUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api_client/users/user/info" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer pk-demo" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":{"id":"user_1","email":"demo@popi.art","name":"Demo","scopes":["creator"]}}`)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "pk-demo")
	user, err := FetchCurrentUser(context.Background(), client)
	if err != nil {
		t.Fatalf("FetchCurrentUser returned error: %v", err)
	}
	if user.Email != "demo@popi.art" {
		t.Fatalf("unexpected user payload: %#v", user)
	}
}

func TestFetchCurrentUserSupportsWebEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api_client/users/user/info" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":{"user":{"id":10561,"email":"web@popi.art","name":"Web User"}},"message":"ok","status":"0000"}`)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "pk-demo")
	user, err := FetchCurrentUser(context.Background(), client)
	if err != nil {
		t.Fatalf("FetchCurrentUser returned error: %v", err)
	}
	if user.ID != "10561" || user.Email != "web@popi.art" || user.Name != "Web User" {
		t.Fatalf("unexpected web user payload: %#v", user)
	}
}

func TestFetchCurrentUserReturnsWebEnvelopeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message":"invalid user","status":"4000"}`)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "pk-demo")
	_, err := FetchCurrentUser(context.Background(), client)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
