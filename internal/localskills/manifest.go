package localskills

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/wtgoku-create/popiartcli/internal/output"
	"github.com/wtgoku-create/popiartcli/internal/types"
)

type Execution struct {
	Mode            string `yaml:"mode" json:"mode"`
	RuntimeSkillID  string `yaml:"runtime_skill_id" json:"runtime_skill_id"`
	Runner          string `yaml:"runner" json:"runner"`
	PreferAuthCheck bool   `yaml:"prefer_auth_check" json:"prefer_auth_check"`
}

type Manifest struct {
	Slug                string    `yaml:"slug" json:"slug"`
	DisplayName         string    `yaml:"display_name" json:"display_name"`
	Description         string    `yaml:"description" json:"description"`
	Category            string    `yaml:"category" json:"category"`
	Version             string    `yaml:"version" json:"version"`
	Archive             string    `yaml:"archive" json:"archive"`
	InstallDir          string    `yaml:"install_dir" json:"install_dir"`
	PackageLayout       string    `yaml:"package_layout" json:"package_layout"`
	PackageRoot         string    `yaml:"package_root" json:"package_root"`
	Capabilities        []string  `yaml:"capabilities" json:"capabilities"`
	RequiresPopiartAuth bool      `yaml:"requires_popiart_auth" json:"requires_popiart_auth"`
	GeneratedBy         string    `yaml:"generated_by" json:"generated_by"`
	ModelType           string    `yaml:"model_type" json:"model_type"`
	EstimatedDurationS  int       `yaml:"estimated_duration_s" json:"estimated_duration_s"`
	InputSchemaPath     string    `yaml:"input_schema_path" json:"input_schema_path"`
	OutputSchemaPath    string    `yaml:"output_schema_path" json:"output_schema_path"`
	Execution           Execution `yaml:"execution" json:"execution"`
	Tags                []string  `yaml:"tags" json:"tags"`
}

type InstalledSkill struct {
	Manifest     Manifest
	RootDir      string
	ManifestPath string
	SkillDocPath string
	InputSchema  map[string]any
	OutputSchema map[string]any
}

func LoadInstalledSkill(rootDir string) (InstalledSkill, error) {
	rootDir = filepath.Clean(rootDir)
	manifest, manifestPath, skillDocPath, body, err := loadManifest(rootDir)
	if err != nil {
		return InstalledSkill{}, err
	}

	inputSchema, err := loadOptionalJSON(filepath.Join(rootDir, manifest.InputSchemaPath))
	if err != nil {
		return InstalledSkill{}, err
	}
	outputSchema, err := loadOptionalJSON(filepath.Join(rootDir, manifest.OutputSchemaPath))
	if err != nil {
		return InstalledSkill{}, err
	}

	if manifest.Description == "" {
		manifest.Description = firstMeaningfulParagraph(body)
	}
	if manifest.Description == "" {
		manifest.Description = manifest.DisplayName
	}
	if manifest.ModelType == "" {
		manifest.ModelType = inferModelType(manifest.Capabilities)
	}

	return InstalledSkill{
		Manifest:     manifest,
		RootDir:      rootDir,
		ManifestPath: manifestPath,
		SkillDocPath: skillDocPath,
		InputSchema:  inputSchema,
		OutputSchema: outputSchema,
	}, nil
}

func (s InstalledSkill) Summary(active bool) types.SkillSummary {
	return types.SkillSummary{
		ID:                  s.Manifest.Slug,
		Name:                defaultString(s.Manifest.DisplayName, s.Manifest.Slug),
		Description:         s.Manifest.Description,
		Tags:                s.tags(),
		Version:             s.Manifest.Version,
		ModelType:           s.Manifest.ModelType,
		EstimatedDurationS:  s.Manifest.EstimatedDurationS,
		Source:              "installed",
		InstallDir:          s.RootDir,
		LocalInstalled:      true,
		LocalActive:         active,
		ExecutionMode:       s.Manifest.Execution.Mode,
		RuntimeSkillID:      effectiveRuntimeSkillID(s.Manifest),
		RequiresPopiartAuth: s.Manifest.RequiresPopiartAuth,
	}
}

func (s InstalledSkill) Skill(active bool) types.Skill {
	return types.Skill{
		ID:                  s.Manifest.Slug,
		Name:                defaultString(s.Manifest.DisplayName, s.Manifest.Slug),
		Description:         s.Manifest.Description,
		Tags:                s.tags(),
		Version:             s.Manifest.Version,
		InputSchema:         cloneMap(s.InputSchema),
		OutputSchema:        cloneMap(s.OutputSchema),
		ModelType:           s.Manifest.ModelType,
		EstimatedDurationS:  s.Manifest.EstimatedDurationS,
		Source:              "installed",
		InstallDir:          s.RootDir,
		LocalInstalled:      true,
		LocalActive:         active,
		ExecutionMode:       s.Manifest.Execution.Mode,
		RuntimeSkillID:      effectiveRuntimeSkillID(s.Manifest),
		RequiresPopiartAuth: s.Manifest.RequiresPopiartAuth,
	}
}

func (s InstalledSkill) Schema() types.SkillSchemaResponse {
	return types.SkillSchemaResponse{
		InputSchema:  cloneMap(s.InputSchema),
		OutputSchema: cloneMap(s.OutputSchema),
	}
}

func (s InstalledSkill) MatchesFilter(tag, search string) bool {
	if !matchesTag(s.tags(), tag) {
		return false
	}
	needle := strings.TrimSpace(strings.ToLower(search))
	if needle == "" {
		return true
	}
	fields := []string{
		s.Manifest.Slug,
		s.Manifest.DisplayName,
		s.Manifest.Description,
		s.Manifest.Category,
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), needle) {
			return true
		}
	}
	for _, value := range s.tags() {
		if strings.Contains(strings.ToLower(value), needle) {
			return true
		}
	}
	return false
}

func effectiveRuntimeSkillID(manifest Manifest) string {
	if strings.TrimSpace(manifest.Execution.RuntimeSkillID) != "" {
		return strings.TrimSpace(manifest.Execution.RuntimeSkillID)
	}
	return manifest.Slug
}

func loadManifest(rootDir string) (Manifest, string, string, string, error) {
	skillDocPath := filepath.Join(rootDir, "SKILL.md")
	skillDocBytes, skillDocErr := os.ReadFile(skillDocPath)

	candidates := []string{
		filepath.Join(rootDir, "popiart-skill.yaml"),
		filepath.Join(rootDir, "popiart-skill.yml"),
		filepath.Join(rootDir, "popiart-skill.json"),
	}

	var manifestBytes []byte
	var manifestPath string
	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err == nil {
			manifestBytes = data
			manifestPath = candidate
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return Manifest{}, "", "", "", output.NewError("CLI_ERROR", "读取本地 skill manifest 失败", map[string]any{
				"path":    candidate,
				"details": err.Error(),
			})
		}
	}

	body := ""
	if len(manifestBytes) == 0 {
		if skillDocErr != nil {
			if errors.Is(skillDocErr, os.ErrNotExist) {
				return Manifest{}, "", "", "", output.NewError("LOCAL_SKILL_INVALID", "本地 skill 缺少 manifest 和 SKILL.md", map[string]any{
					"dir": rootDir,
				})
			}
			return Manifest{}, "", "", "", output.NewError("CLI_ERROR", "读取本地 skill 说明失败", map[string]any{
				"path":    skillDocPath,
				"details": skillDocErr.Error(),
			})
		}

		header, markdownBody, ok := splitFrontMatter(string(skillDocBytes))
		if !ok {
			return Manifest{}, "", "", "", output.NewError("LOCAL_SKILL_INVALID", "SKILL.md 缺少 YAML frontmatter", map[string]any{
				"path": skillDocPath,
			})
		}
		manifestBytes = []byte(header)
		manifestPath = skillDocPath
		body = markdownBody
	} else {
		if skillDocErr != nil {
			if errors.Is(skillDocErr, os.ErrNotExist) {
				return Manifest{}, "", "", "", output.NewError("LOCAL_SKILL_INVALID", "本地 skill 缺少 SKILL.md", map[string]any{
					"dir": rootDir,
				})
			}
			return Manifest{}, "", "", "", output.NewError("CLI_ERROR", "读取本地 skill 说明失败", map[string]any{
				"path":    skillDocPath,
				"details": skillDocErr.Error(),
			})
		}
		body = string(skillDocBytes)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(manifestBytes, &manifest); err != nil {
		return Manifest{}, "", "", "", output.NewError("LOCAL_SKILL_INVALID", "解析本地 skill manifest 失败", map[string]any{
			"path":    manifestPath,
			"details": err.Error(),
		})
	}

	manifest.Slug = strings.TrimSpace(manifest.Slug)
	if manifest.Slug == "" {
		return Manifest{}, "", "", "", output.NewError("LOCAL_SKILL_INVALID", "本地 skill manifest 缺少 slug", map[string]any{
			"path": manifestPath,
		})
	}
	if manifest.DisplayName == "" {
		manifest.DisplayName = manifest.Slug
	}
	if manifest.InputSchemaPath == "" && fileExists(filepath.Join(rootDir, "input_schema.json")) {
		manifest.InputSchemaPath = "input_schema.json"
	}
	if manifest.OutputSchemaPath == "" && fileExists(filepath.Join(rootDir, "output_schema.json")) {
		manifest.OutputSchemaPath = "output_schema.json"
	}
	manifest.PackageLayout = strings.TrimSpace(strings.ToLower(manifest.PackageLayout))
	if manifest.PackageLayout == "" {
		manifest.PackageLayout = "unrooted"
	}
	manifest.Execution.Mode = strings.TrimSpace(strings.ToLower(manifest.Execution.Mode))
	if manifest.Execution.Runner == "" {
		manifest.Execution.Runner = "popiart"
	}

	return manifest, manifestPath, skillDocPath, body, nil
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

func loadOptionalJSON(path string) (map[string]any, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, output.NewError("LOCAL_SKILL_INVALID", "本地 skill 缺少 schema 文件", map[string]any{
				"path": path,
			})
		}
		return nil, output.NewError("CLI_ERROR", "读取本地 schema 失败", map[string]any{
			"path":    path,
			"details": err.Error(),
		})
	}

	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, output.NewError("LOCAL_SKILL_INVALID", "解析本地 schema 失败", map[string]any{
			"path":    path,
			"details": err.Error(),
		})
	}
	return schema, nil
}

func inferModelType(capabilities []string) string {
	if len(capabilities) == 0 {
		return "meta"
	}
	return strings.TrimSpace(capabilities[0])
}

func firstMeaningfulParagraph(body string) string {
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			line = strings.TrimSpace(line[2:])
		}
		if line != "" {
			return line
		}
	}
	return ""
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

func (s InstalledSkill) tags() []string {
	tags := make([]string, 0, len(s.Manifest.Tags)+len(s.Manifest.Capabilities)+3)
	seen := map[string]bool{}
	appendTag := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		key := strings.ToLower(value)
		if seen[key] {
			return
		}
		seen[key] = true
		tags = append(tags, value)
	}
	appendTag("installed")
	appendTag("local")
	for _, value := range s.Manifest.Tags {
		appendTag(value)
	}
	for _, value := range s.Manifest.Capabilities {
		appendTag(value)
	}
	if s.Manifest.Category != "" {
		appendTag(s.Manifest.Category)
	}
	return tags
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func cloneMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	for key, value := range src {
		dst[key] = deepClone(value)
	}
	return dst
}

func deepClone(value any) any {
	switch current := value.(type) {
	case map[string]any:
		return cloneMap(current)
	case []any:
		items := make([]any, 0, len(current))
		for _, item := range current {
			items = append(items, deepClone(item))
		}
		return items
	default:
		return current
	}
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
