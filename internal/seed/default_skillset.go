package seed

import "time"

type DefaultSkillset struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	GeneratedAt string              `json:"generated_at"`
	Source      string              `json:"source"`
	Queries     []DefaultSkillQuery `json:"queries"`
	SeedSkills  []SeedSkill         `json:"seed_skills,omitempty"`
}

type DefaultSkillQuery struct {
	Label  string `json:"label"`
	Tag    string `json:"tag"`
	Limit  int    `json:"limit"`
	Search string `json:"search,omitempty"`
}

type SeedSkill struct {
	Name        string `json:"name"`
	RepoPath    string `json:"repo_path"`
	GitHubURL   string `json:"github_url"`
	Description string `json:"description"`
}

func NewDefaultSkillset(generatedAt time.Time) DefaultSkillset {
	return DefaultSkillset{
		Name:        "default",
		Description: "Starter discovery profile for PopiArt's remote skill registry, plus bundled seed skills shipped with popiartcli.",
		GeneratedAt: generatedAt.UTC().Format(time.RFC3339),
		Source:      "popiart bootstrap",
		Queries: []DefaultSkillQuery{
			{Label: "Image", Tag: "image", Limit: 20},
			{Label: "Video", Tag: "video", Limit: 20},
			{Label: "Audio", Tag: "audio", Limit: 20},
			{Label: "Trending", Search: "popular", Limit: 20},
		},
		SeedSkills: SeedSkillsForProfile(),
	}
}
