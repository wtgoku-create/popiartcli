package skillassets

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type EmbeddedSkill struct {
	ID                 string
	Name               string
	Description        string
	Tags               []string
	Version            string
	ModelType          string
	EstimatedDurationS int
	DefaultProfile     bool
	ProfileDescription string
	InputSchema        map[string]any
	OutputSchema       map[string]any
}

type embeddedSkillManifest struct {
	Name               string   `yaml:"name"`
	Description        string   `yaml:"description"`
	Tags               []string `yaml:"tags"`
	Version            string   `yaml:"version"`
	ModelType          string   `yaml:"model_type"`
	EstimatedDurationS int      `yaml:"estimated_duration_s"`
	DefaultProfile     bool     `yaml:"default_profile"`
	ProfileDescription string   `yaml:"profile_description"`
}

//go:embed */SKILL.md */input_schema.json */output_schema.json
var embeddedSkillFiles embed.FS

func LoadBundledSkills() ([]EmbeddedSkill, error) {
	skillDocs, err := fs.Glob(embeddedSkillFiles, "*/SKILL.md")
	if err != nil {
		return nil, fmt.Errorf("glob embedded skills: %w", err)
	}
	sort.Strings(skillDocs)

	items := make([]EmbeddedSkill, 0, len(skillDocs))
	for _, skillDocPath := range skillDocs {
		item, err := loadEmbeddedSkill(skillDocPath)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func loadEmbeddedSkill(skillDocPath string) (EmbeddedSkill, error) {
	docBytes, err := embeddedSkillFiles.ReadFile(skillDocPath)
	if err != nil {
		return EmbeddedSkill{}, fmt.Errorf("read embedded skill doc %s: %w", skillDocPath, err)
	}

	header, _, ok := splitFrontMatter(string(docBytes))
	if !ok {
		return EmbeddedSkill{}, fmt.Errorf("embedded skill doc %s is missing YAML frontmatter", skillDocPath)
	}

	var manifest embeddedSkillManifest
	if err := yaml.Unmarshal([]byte(header), &manifest); err != nil {
		return EmbeddedSkill{}, fmt.Errorf("parse embedded skill manifest %s: %w", skillDocPath, err)
	}
	if strings.TrimSpace(manifest.Name) == "" {
		return EmbeddedSkill{}, fmt.Errorf("embedded skill doc %s is missing name", skillDocPath)
	}
	if strings.TrimSpace(manifest.Description) == "" {
		return EmbeddedSkill{}, fmt.Errorf("embedded skill doc %s is missing description", skillDocPath)
	}

	root := path.Dir(skillDocPath)
	inputSchema, err := readEmbeddedSchema(path.Join(root, "input_schema.json"))
	if err != nil {
		return EmbeddedSkill{}, err
	}
	outputSchema, err := readEmbeddedSchema(path.Join(root, "output_schema.json"))
	if err != nil {
		return EmbeddedSkill{}, err
	}

	return EmbeddedSkill{
		ID:                 strings.TrimSpace(manifest.Name),
		Name:               strings.TrimSpace(manifest.Name),
		Description:        strings.TrimSpace(manifest.Description),
		Tags:               append([]string(nil), manifest.Tags...),
		Version:            strings.TrimSpace(manifest.Version),
		ModelType:          strings.TrimSpace(manifest.ModelType),
		EstimatedDurationS: manifest.EstimatedDurationS,
		DefaultProfile:     manifest.DefaultProfile,
		ProfileDescription: strings.TrimSpace(manifest.ProfileDescription),
		InputSchema:        inputSchema,
		OutputSchema:       outputSchema,
	}, nil
}

func readEmbeddedSchema(schemaPath string) (map[string]any, error) {
	data, err := embeddedSkillFiles.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("read embedded schema %s: %w", schemaPath, err)
	}

	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("parse embedded schema %s: %w", schemaPath, err)
	}
	return normalizeEmbeddedSchemaMap(schema), nil
}

func splitFrontMatter(text string) (string, string, bool) {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	if !strings.HasPrefix(normalized, "---\n") {
		return "", "", false
	}

	rest := normalized[len("---\n"):]
	index := strings.Index(rest, "\n---\n")
	if index < 0 {
		return "", "", false
	}

	return rest[:index], rest[index+len("\n---\n"):], true
}

func normalizeEmbeddedSchemaMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	for key, value := range src {
		dst[key] = normalizeEmbeddedSchemaValue(value)
	}
	return dst
}

func normalizeEmbeddedSchemaValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return normalizeEmbeddedSchemaMap(typed)
	case []any:
		stringItems := make([]string, 0, len(typed))
		allStrings := true
		normalizedItems := make([]any, 0, len(typed))
		for _, item := range typed {
			normalized := normalizeEmbeddedSchemaValue(item)
			normalizedItems = append(normalizedItems, normalized)
			text, ok := normalized.(string)
			if !ok {
				allStrings = false
				continue
			}
			stringItems = append(stringItems, text)
		}
		if allStrings {
			return stringItems
		}
		return normalizedItems
	default:
		return typed
	}
}
