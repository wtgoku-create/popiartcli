package seed

import "testing"

func TestMatchingBundledSkillSummariesFindsAliceShowcase(t *testing.T) {
	items := MatchingBundledSkillSummaries("image", "alice")
	if len(items) != 1 {
		t.Fatalf("expected 1 bundled alice showcase skill, got %d", len(items))
	}
	if items[0].ID != "popiskill-image-img2img-popistudio-alice-showcase-v1" {
		t.Fatalf("expected Alice showcase skill, got %s", items[0].ID)
	}
	if items[0].Source != "bundled-seed" {
		t.Fatalf("expected bundled-seed source, got %#v", items[0].Source)
	}
}

func TestFindBundledSkillSchemaByID(t *testing.T) {
	schema, ok := FindBundledSkillSchema("popiskill-video-image2video-basic-v1")
	if !ok {
		t.Fatal("expected bundled skill schema to exist")
	}
	if schema.InputSchema["type"] != "object" {
		t.Fatalf("expected object input schema, got %#v", schema.InputSchema["type"])
	}
}

func TestFindBundledSkillByIDIncludesBundledSeedSource(t *testing.T) {
	skill, ok := FindBundledSkill("popiskill-video-image2video-basic-v1")
	if !ok {
		t.Fatal("expected bundled skill to exist")
	}
	if skill.Source != "bundled-seed" {
		t.Fatalf("expected bundled-seed source, got %#v", skill.Source)
	}
}

func TestMatchingBundledSkillSummariesFindsSTTSkill(t *testing.T) {
	items := MatchingBundledSkillSummaries("", "transcribe")
	if len(items) != 1 {
		t.Fatalf("expected 1 bundled stt skill, got %d", len(items))
	}
	if items[0].ID != "popiskill-audio-stt-local-v1" {
		t.Fatalf("expected popiskill-audio-stt-local-v1, got %s", items[0].ID)
	}
}

func TestSeedSkillsForProfileIncludesAllOfficialRuntimeSkills(t *testing.T) {
	items := SeedSkillsForProfile()
	found := map[string]bool{
		"popiskill-image-text2image-basic-v1":                      false,
		"popiskill-image-img2img-basic-v1":                         false,
		"popiskill-image-img2img-popistudio-alice-showcase-v1":     false,
		"popiskill-video-image2video-basic-v1":                     false,
		"popiskill-video-image2video-popistudio-alice-showcase-v1": false,
		"popiskill-audio-tts-multimodel-v1":                        false,
		"popiskill-audio-stt-local-v1":                             false,
	}
	for _, item := range items {
		if _, ok := found[item.Name]; ok {
			found[item.Name] = true
		}
	}
	for skillID, ok := range found {
		if !ok {
			t.Fatalf("expected default seed skill profile to include %s", skillID)
		}
	}
}

func TestImage2VideoSchemaIncludesArtifactAndTimingHints(t *testing.T) {
	schema, ok := FindBundledSkillSchema("popiskill-video-image2video-basic-v1")
	if !ok {
		t.Fatal("expected bundled image2video schema to exist")
	}

	properties, ok := schema.InputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties map, got %#v", schema.InputSchema["properties"])
	}

	for _, key := range []string{"source_artifact_id", "reference_image_url", "duration_s", "seconds", "camera_motion", "aspect_ratio"} {
		if _, ok := properties[key]; !ok {
			t.Fatalf("expected image2video schema to include %q", key)
		}
	}
}
