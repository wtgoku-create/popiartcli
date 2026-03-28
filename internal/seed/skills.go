package seed

import (
	"strings"

	"github.com/wtgoku-create/popiartcli/internal/types"
)

type BundledSkill struct {
	ID                 string
	Name               string
	Description        string
	Tags               []string
	Version            string
	ModelType          string
	EstimatedDurationS int
	RepoPath           string
	GitHubURL          string
	InputSchema        map[string]any
	OutputSchema       map[string]any
}

var bundledSkills = []BundledSkill{
	{
		ID:                 "popiskill-creator",
		Name:               "popiskill-creator",
		Description:        "Create, adapt, bootstrap, and validate PopiArt skills through popiartcli and Popiart_skillhub.",
		Tags:               []string{"seed", "local", "bootstrap", "meta", "authoring"},
		Version:            "v1",
		ModelType:          "meta",
		EstimatedDurationS: 120,
		RepoPath:           "skills/popiskill-creator",
		GitHubURL:          "https://github.com/wtgoku-create/popiartcli/tree/main/skills/popiskill-creator",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"goal": map[string]any{
					"type":        "string",
					"description": "What the user wants to achieve with PopiArt, such as install, auth, architecture clarification, or skill authoring.",
				},
				"context": map[string]any{
					"type":        "string",
					"description": "Optional repo, workflow, or product context needed to ground the answer.",
				},
			},
			"required": []string{"goal"},
		},
		OutputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"steps": map[string]any{
					"type":        "array",
					"description": "Concrete setup, auth, authoring, or validation steps.",
					"items":       map[string]any{"type": "string"},
				},
				"commands": map[string]any{
					"type":        "array",
					"description": "Exact popiartcli or git commands to run next.",
					"items":       map[string]any{"type": "string"},
				},
			},
		},
	},
	{
		ID:                 "popiskill-image-character-three-view-v1",
		Name:               "popiskill-image-character-three-view-v1",
		Description:        "Generate a consistent full-body character three-view sheet with front, side, and back views from a character brief.",
		Tags:               []string{"seed", "local", "image", "character", "three-view"},
		Version:            "v1",
		ModelType:          "image",
		EstimatedDurationS: 180,
		RepoPath:           "skills/popiskill-image-character-three-view-v1",
		GitHubURL:          "https://github.com/wtgoku-create/popiartcli/tree/main/skills/popiskill-image-character-three-view-v1",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"character_prompt": map[string]any{
					"type":        "string",
					"description": "Character brief describing the same identity across front, side, and back views.",
				},
				"reference_artifact_ids": map[string]any{
					"type":        "array",
					"description": "Optional PopiArt artifact IDs used as reference.",
					"items":       map[string]any{"type": "string"},
				},
				"style":                    map[string]any{"type": "string"},
				"background_mode":          map[string]any{"type": "string"},
				"pose_mode":                map[string]any{"type": "string"},
				"views":                    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"include_items":            map[string]any{"type": "boolean"},
				"include_palette":          map[string]any{"type": "boolean"},
				"expression_count":         map[string]any{"type": "integer"},
				"include_seasonal_outfits": map[string]any{"type": "boolean"},
				"action_count":             map[string]any{"type": "integer"},
				"aspect_ratio":             map[string]any{"type": "string"},
				"notes":                    map[string]any{"type": "string"},
			},
			"required": []string{"character_prompt"},
		},
		OutputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"artifact_ids": map[string]any{
					"type":        "array",
					"description": "Generated sheet artifacts and any optional metadata artifacts.",
					"items":       map[string]any{"type": "string"},
				},
				"job_id": map[string]any{"type": "string"},
			},
		},
	},
	{
		ID:                 "popiskill-image-generate-edit-workflow-v1",
		Name:               "popiskill-image-generate-edit-workflow-v1",
		Description:        "Plan PopiArt image requests by preserving the original prompt, choosing text2image or artifact-based img2img, and mapping aspect-ratio or resolution intent into a stable execution size.",
		Tags:               []string{"seed", "local", "image", "text2image", "img2img", "workflow"},
		Version:            "v1",
		ModelType:          "image",
		EstimatedDurationS: 180,
		RepoPath:           "skills/popiskill-image-generate-edit-workflow-v1",
		GitHubURL:          "https://github.com/wtgoku-create/popiartcli/tree/main/skills/popiskill-image-generate-edit-workflow-v1",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"prompt": map[string]any{
					"type":        "string",
					"description": "The user's original full prompt or edit instruction. Preserve it verbatim and only append confirmed clarifications.",
				},
				"mode": map[string]any{
					"type":        "string",
					"description": "Optional workflow hint such as generate or edit, used for planning before choosing the runtime skill.",
				},
				"source_artifact_id": map[string]any{
					"type":        "string",
					"description": "Preferred PopiArt artifact used as the source image for img2img, especially for chat attachments or previous PopiArt outputs.",
				},
				"reference_image_url": map[string]any{
					"type":        "string",
					"description": "Fallback remote image URL when no source_artifact_id exists.",
				},
				"image_url": map[string]any{
					"type":        "string",
					"description": "Compatibility alias for a remote source image URL.",
				},
				"size": map[string]any{
					"type":        "string",
					"description": "Preferred execution parameter. Choose a stable size such as 1024x1024, 1536x1024, 1024x1536, 1792x1024, or 1024x1792.",
				},
				"aspect_ratio": map[string]any{
					"type":        "string",
					"description": "Optional planning hint such as 1:1, 16:9, 9:16, 4:3, 3:4, 3:2, 2:3, 5:4, 4:5, 21:9, or more extreme ratios like 1:4 and 4:1. Map it to the nearest safe size before execution.",
				},
				"resolution": map[string]any{
					"type":        "string",
					"description": "Optional planning hint such as 1K, 2K, or 4K. Do not assume the runtime skill accepts it directly; map it to a supported size when needed.",
				},
				"notes": map[string]any{
					"type":        "string",
					"description": "Optional extra constraints appended after user confirmation, such as style, camera language, forbidden elements, or copy requirements.",
				},
			},
			"required": []string{"prompt"},
		},
		OutputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"recommended_skill_id": map[string]any{
					"type":        "string",
					"description": "The runtime skill that should execute the request, such as popiskill-image-text2image-basic-v1 or popiskill-image-img2img-basic-v1.",
				},
				"job_id": map[string]any{
					"type":        "string",
					"description": "The PopiArt job id after execution.",
				},
				"artifact_ids": map[string]any{
					"type":        "array",
					"description": "Generated or edited image artifacts.",
					"items":       map[string]any{"type": "string"},
				},
			},
		},
	},
}

func SeedSkillsForProfile() []SeedSkill {
	return []SeedSkill{
		{
			Name:        "popiskill-creator",
			RepoPath:    "skills/popiskill-creator",
			GitHubURL:   "https://github.com/wtgoku-create/popiartcli/tree/main/skills/popiskill-creator",
			Description: "Bootstrap skill for installing popiartcli, authenticating with a PopiArt key, understanding the unified gateway boundary, and turning creator workflows into PopiArt skills.",
		},
		{
			Name:        "popiskill-image-generate-edit-workflow-v1",
			RepoPath:    "skills/popiskill-image-generate-edit-workflow-v1",
			GitHubURL:   "https://github.com/wtgoku-create/popiartcli/tree/main/skills/popiskill-image-generate-edit-workflow-v1",
			Description: "Default PopiArt image workflow skill for mapping user requests to text2image or artifact-based img2img runs.",
		},
	}
}

func MatchingBundledSkillSummaries(tag, search string) []types.SkillSummary {
	matches := filterBundledSkills(tag, search)
	items := make([]types.SkillSummary, 0, len(matches))
	for _, skill := range matches {
		items = append(items, skill.Summary())
	}
	return items
}

func FindBundledSkill(id string) (types.Skill, bool) {
	for _, skill := range bundledSkills {
		if skill.matchesID(id) {
			return skill.Skill(), true
		}
	}
	return types.Skill{}, false
}

func FindBundledSkillSchema(id string) (types.SkillSchemaResponse, bool) {
	for _, skill := range bundledSkills {
		if skill.matchesID(id) {
			return types.SkillSchemaResponse{
				InputSchema:  cloneMap(skill.InputSchema),
				OutputSchema: cloneMap(skill.OutputSchema),
			}, true
		}
	}
	return types.SkillSchemaResponse{}, false
}

func filterBundledSkills(tag, search string) []BundledSkill {
	items := make([]BundledSkill, 0, len(bundledSkills))
	for _, skill := range bundledSkills {
		if skill.matchesFilter(tag, search) {
			items = append(items, skill)
		}
	}
	return items
}

func (s BundledSkill) Summary() types.SkillSummary {
	return types.SkillSummary{
		ID:                 s.ID,
		Name:               s.Name,
		Description:        s.Description,
		Tags:               append([]string(nil), s.Tags...),
		Version:            s.Version,
		ModelType:          s.ModelType,
		EstimatedDurationS: s.EstimatedDurationS,
	}
}

func (s BundledSkill) Skill() types.Skill {
	return types.Skill{
		ID:                 s.ID,
		Name:               s.Name,
		Description:        s.Description,
		Tags:               append([]string(nil), s.Tags...),
		Version:            s.Version,
		ModelType:          s.ModelType,
		EstimatedDurationS: s.EstimatedDurationS,
		InputSchema:        cloneMap(s.InputSchema),
		OutputSchema:       cloneMap(s.OutputSchema),
	}
}

func (s BundledSkill) matchesID(id string) bool {
	needle := strings.TrimSpace(strings.ToLower(id))
	return needle != "" && (strings.ToLower(s.ID) == needle || strings.ToLower(s.Name) == needle)
}

func (s BundledSkill) matchesFilter(tag, search string) bool {
	if !matchesTag(s.Tags, tag) {
		return false
	}
	return matchesSearch(s, search)
}

func matchesTag(tags []string, tag string) bool {
	needle := strings.TrimSpace(strings.ToLower(tag))
	if needle == "" {
		return true
	}
	for _, value := range tags {
		if strings.EqualFold(value, needle) {
			return true
		}
	}
	return false
}

func matchesSearch(skill BundledSkill, search string) bool {
	needle := strings.TrimSpace(strings.ToLower(search))
	if needle == "" {
		return true
	}
	if strings.Contains(strings.ToLower(skill.ID), needle) {
		return true
	}
	if strings.Contains(strings.ToLower(skill.Name), needle) {
		return true
	}
	if strings.Contains(strings.ToLower(skill.Description), needle) {
		return true
	}
	for _, tag := range skill.Tags {
		if strings.Contains(strings.ToLower(tag), needle) {
			return true
		}
	}
	return false
}

func cloneMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}
