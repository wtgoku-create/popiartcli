package skillassets

import "testing"

func TestLoadBundledSkills(t *testing.T) {
	items, err := LoadBundledSkills()
	if err != nil {
		t.Fatalf("LoadBundledSkills returned error: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 embedded bundled skills, got %d", len(items))
	}

	var creator, workflow *EmbeddedSkill
	for i := range items {
		switch items[i].ID {
		case "popiskill-creator":
			creator = &items[i]
		case "popiskill-image-generate-edit-workflow-v1":
			workflow = &items[i]
		}
	}

	if creator == nil {
		t.Fatal("expected embedded creator skill")
	}
	if !creator.DefaultProfile {
		t.Fatal("expected creator to be part of default profile")
	}
	if len(creator.InputSchema) == 0 || len(creator.OutputSchema) == 0 {
		t.Fatalf("expected creator schemas to be embedded, got %#v %#v", creator.InputSchema, creator.OutputSchema)
	}

	if workflow == nil {
		t.Fatal("expected embedded image workflow skill")
	}
	if !workflow.DefaultProfile {
		t.Fatal("expected image workflow to be part of default profile")
	}
}
