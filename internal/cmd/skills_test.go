package cmd

import (
	"bytes"
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
			{ID: officialText2ImageSkillID, Description: "remote"},
			{ID: "remote-2"},
		},
		[]types.SkillSummary{
			{ID: officialText2ImageSkillID, Description: "local"},
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
		case "/skills/" + officialText2ImageSkillID:
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"id":"%s","name":"remote text2image"}`, officialText2ImageSkillID)
		case "/skills/" + officialAliceImageShowcaseSkillID:
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
		{ID: officialText2ImageSkillID},
		{ID: officialAliceImageShowcaseSkillID},
	}
	filtered, err := bundledSkillSummariesMissingOnRemote(context.Background(), items)
	if err != nil {
		t.Fatalf("bundledSkillSummariesMissingOnRemote returned error: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered item, got %d", len(filtered))
	}
	if filtered[0].ID != officialAliceImageShowcaseSkillID {
		t.Fatalf("unexpected filtered item: %#v", filtered[0])
	}
}

func TestValidateBundledSkillRunAllowsOfficialRuntimeSkillWhenRemoteMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"message":"not found"}`)
	}))
	defer server.Close()

	t.Setenv("POPIART_ENDPOINT", server.URL)

	if err := validateBundledSkillRun(context.Background(), officialImage2VideoSkillID); err != nil {
		t.Fatalf("expected official runtime skill to bypass local-only validation, got %v", err)
	}
}

func TestSkillsListRejectsInvalidPaginationFlag(t *testing.T) {
	root := NewRootCmd("0.test")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"skills", "list", "--limit", "abc"})
	root.SetContext(context.Background())

	err := root.Execute()
	if err == nil {
		t.Fatal("expected validation error for invalid limit flag")
	}
	cliErr, ok := err.(*output.CLIError)
	if !ok {
		t.Fatalf("expected CLIError, got %T", err)
	}
	if cliErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %q", cliErr.Code)
	}
}
