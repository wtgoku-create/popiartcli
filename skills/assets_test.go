package skillassets

import "testing"

func TestLoadBundledSkills(t *testing.T) {
	items, err := LoadBundledSkills()
	if err != nil {
		t.Fatalf("LoadBundledSkills returned error: %v", err)
	}
	if len(items) != 7 {
		t.Fatalf("expected 7 embedded bundled skills, got %d", len(items))
	}

	requiredIDs := map[string]bool{
		"popiskill-image-text2image-basic-v1":                      false,
		"popiskill-image-img2img-basic-v1":                         false,
		"popiskill-image-img2img-popistudio-alice-showcase-v1":     false,
		"popiskill-video-image2video-basic-v1":                     false,
		"popiskill-video-image2video-popistudio-alice-showcase-v1": false,
		"popiskill-audio-tts-multimodel-v1":                        false,
		"popiskill-audio-stt-local-v1":                             false,
	}
	for i := range items {
		requiredIDs[items[i].ID] = true
		if !items[i].DefaultProfile {
			t.Fatalf("expected %s to be part of default profile", items[i].ID)
		}
		if len(items[i].InputSchema) == 0 || len(items[i].OutputSchema) == 0 {
			t.Fatalf("expected schemas for %s to be embedded, got %#v %#v", items[i].ID, items[i].InputSchema, items[i].OutputSchema)
		}
	}
	for skillID, found := range requiredIDs {
		if !found {
			t.Fatalf("expected embedded bundled skill %s", skillID)
		}
	}
}
