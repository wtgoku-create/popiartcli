package seed

import "testing"

func TestMatchingBundledSkillSummariesFindsCreator(t *testing.T) {
	items := MatchingBundledSkillSummaries("", "creator")
	if len(items) != 1 {
		t.Fatalf("expected 1 bundled creator skill, got %d", len(items))
	}
	if items[0].ID != "popiskill-creator" {
		t.Fatalf("expected popiskill-creator, got %s", items[0].ID)
	}
}

func TestFindBundledSkillSchemaByID(t *testing.T) {
	schema, ok := FindBundledSkillSchema("popiskill-image-character-three-view-v1")
	if !ok {
		t.Fatal("expected bundled skill schema to exist")
	}
	if schema.InputSchema["type"] != "object" {
		t.Fatalf("expected object input schema, got %#v", schema.InputSchema["type"])
	}
}
