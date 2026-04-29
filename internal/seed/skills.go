package seed

import (
	"fmt"
	"strings"

	"github.com/wtgoku-create/popiartcli/internal/types"
	skillassets "github.com/wtgoku-create/popiartcli/popiskills"
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
	DefaultProfile     bool
	ProfileDescription string
}

var bundledSkills = mustLoadBundledSkills()

func SeedSkillsForProfile() []SeedSkill {
	items := make([]SeedSkill, 0, len(bundledSkills))
	for _, skill := range bundledSkills {
		if !skill.DefaultProfile {
			continue
		}
		description := skill.Description
		if strings.TrimSpace(skill.ProfileDescription) != "" {
			description = skill.ProfileDescription
		}
		items = append(items, SeedSkill{
			Name:        skill.ID,
			RepoPath:    skill.RepoPath,
			GitHubURL:   skill.GitHubURL,
			Description: description,
		})
	}
	return items
}

func mustLoadBundledSkills() []BundledSkill {
	items, err := skillassets.LoadBundledSkills()
	if err != nil {
		panic(fmt.Sprintf("load bundled seed skills: %v", err))
	}

	bundled := make([]BundledSkill, 0, len(items))
	for _, item := range items {
		repoPath := "popiskills/" + item.ID
		bundled = append(bundled, BundledSkill{
			ID:                 item.ID,
			Name:               item.Name,
			Description:        item.Description,
			Tags:               append([]string(nil), item.Tags...),
			Version:            item.Version,
			ModelType:          item.ModelType,
			EstimatedDurationS: item.EstimatedDurationS,
			RepoPath:           repoPath,
			GitHubURL:          "https://github.com/wtgoku-create/popiartcli/tree/main/" + repoPath,
			InputSchema:        cloneMap(item.InputSchema),
			OutputSchema:       cloneMap(item.OutputSchema),
			DefaultProfile:     item.DefaultProfile,
			ProfileDescription: item.ProfileDescription,
		})
	}
	return bundled
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
		Source:             "bundled-seed",
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
		Source:             "bundled-seed",
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
