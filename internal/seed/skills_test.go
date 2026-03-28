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

func TestMatchingBundledSkillSummariesFindsImageWorkflow(t *testing.T) {
	items := MatchingBundledSkillSummaries("", "artifact-based")
	if len(items) != 1 {
		t.Fatalf("expected 1 bundled image workflow skill, got %d", len(items))
	}
	if items[0].ID != "popiskill-image-generate-edit-workflow-v1" {
		t.Fatalf("expected popiskill-image-generate-edit-workflow-v1, got %s", items[0].ID)
	}
}

func TestSeedSkillsForProfileIncludesImageWorkflow(t *testing.T) {
	items := SeedSkillsForProfile()
	found := false
	for _, item := range items {
		if item.Name == "popiskill-image-generate-edit-workflow-v1" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected default seed skill profile to include the image workflow skill")
	}
}

func TestImageWorkflowSchemaIncludesArtifactAndPlanningHints(t *testing.T) {
	schema, ok := FindBundledSkillSchema("popiskill-image-generate-edit-workflow-v1")
	if !ok {
		t.Fatal("expected bundled image workflow schema to exist")
	}

	properties, ok := schema.InputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties map, got %#v", schema.InputSchema["properties"])
	}

	for _, key := range []string{"prompt", "source_artifact_id", "reference_image_url", "size", "aspect_ratio", "resolution"} {
		if _, ok := properties[key]; !ok {
			t.Fatalf("expected workflow schema to include %q", key)
		}
	}

	required, ok := schema.InputSchema["required"].([]string)
	if !ok {
		t.Fatalf("expected required fields to be []string, got %#v", schema.InputSchema["required"])
	}
	if len(required) != 1 || required[0] != "prompt" {
		t.Fatalf("expected only prompt to be required, got %#v", required)
	}
}
