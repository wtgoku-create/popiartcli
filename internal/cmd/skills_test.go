package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wtgoku-create/popiartcli/internal/output"
	"github.com/wtgoku-create/popiartcli/internal/types"
)

func TestPaginateSkillSummaries(t *testing.T) {
	items := []types.SkillSummary{
		{ID: "local-1"},
		{ID: "remote-1"},
		{ID: "remote-2"},
	}

	page := paginateSkillSummaries(items, 2, 1)
	if len(page) != 2 {
		t.Fatalf("expected 2 items, got %d", len(page))
	}
	if page[0].ID != "remote-1" || page[1].ID != "remote-2" {
		t.Fatalf("unexpected page: %#v", page)
	}
}

func TestRemotePageSize(t *testing.T) {
	if got := remotePageSize(50, 10); got != 60 {
		t.Fatalf("expected 60, got %d", got)
	}
	if got := remotePageSize(0, 0); got != 1 {
		t.Fatalf("expected minimum page size 1, got %d", got)
	}
}

func TestMergeSkillSummariesPrefersPrimaryItems(t *testing.T) {
	merged := mergeSkillSummaries(
		[]types.SkillSummary{
			{ID: "popiskill-creator", Description: "remote"},
			{ID: "remote-2"},
		},
		[]types.SkillSummary{
			{ID: "popiskill-creator", Description: "local"},
			{Name: "local-only"},
		},
	)

	if len(merged) != 3 {
		t.Fatalf("expected 3 merged items, got %d", len(merged))
	}
	if merged[0].Description != "remote" {
		t.Fatalf("expected primary item to win, got %#v", merged[0])
	}
	if merged[2].Name != "local-only" {
		t.Fatalf("expected local-only item to be appended, got %#v", merged[2])
	}
}

func TestBundledSkillSummariesMissingOnRemoteFiltersExistingSkills(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/skills/popiskill-creator":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"popiskill-creator","name":"remote creator"}`)
		case "/skills/popiskill-image-character-three-view-v1":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"message":"not found"}`)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"message":"not found"}`)
		}
	}))
	defer server.Close()

	t.Setenv("POPIART_ENDPOINT", server.URL)

	items := []types.SkillSummary{
		{ID: "popiskill-creator"},
		{ID: "popiskill-image-character-three-view-v1"},
	}
	filtered, err := bundledSkillSummariesMissingOnRemote(context.Background(), items)
	if err != nil {
		t.Fatalf("bundledSkillSummariesMissingOnRemote returned error: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered item, got %d", len(filtered))
	}
	if filtered[0].ID != "popiskill-image-character-three-view-v1" {
		t.Fatalf("unexpected filtered item: %#v", filtered[0])
	}
}

func TestValidateBundledSkillRunReturnsLocalOnlyErrorWhenRemoteMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"message":"not found"}`)
	}))
	defer server.Close()

	t.Setenv("POPIART_ENDPOINT", server.URL)

	err := validateBundledSkillRun(context.Background(), "popiskill-creator")
	if err == nil {
		t.Fatal("expected local-only error, got nil")
	}
	cliErr, ok := err.(*output.CLIError)
	if !ok {
		t.Fatalf("expected CLIError, got %T", err)
	}
	if cliErr.Code != "LOCAL_ONLY_SKILL" {
		t.Fatalf("expected LOCAL_ONLY_SKILL, got %q", cliErr.Code)
	}
}
