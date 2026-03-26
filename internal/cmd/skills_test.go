package cmd

import (
	"testing"

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
