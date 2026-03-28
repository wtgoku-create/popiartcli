package localskills

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wtgoku-create/popiartcli/internal/config"
	"github.com/wtgoku-create/popiartcli/internal/output"
)

type State struct {
	Active map[string]bool `json:"active,omitempty"`
}

func ListInstalled() ([]InstalledSkill, error) {
	dir := config.InstalledSkillsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, output.NewError("CLI_ERROR", "读取本地 skill 目录失败", map[string]any{
			"path":    dir,
			"details": err.Error(),
		})
	}

	items := make([]InstalledSkill, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skill, err := LoadInstalledSkill(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		items = append(items, skill)
	}

	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Manifest.Slug) < strings.ToLower(items[j].Manifest.Slug)
	})
	return items, nil
}

func FindInstalled(slug string) (InstalledSkill, bool, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return InstalledSkill{}, false, nil
	}

	skill, err := LoadInstalledSkill(filepath.Join(config.InstalledSkillsDir(), slug))
	if err != nil {
		if cliErr, ok := err.(*output.CLIError); ok && cliErr.Code == "LOCAL_SKILL_INVALID" {
			return InstalledSkill{}, false, nil
		}
		if errors.Is(err, os.ErrNotExist) {
			return InstalledSkill{}, false, nil
		}
		return InstalledSkill{}, false, err
	}
	return skill, true, nil
}

func LoadState() (State, error) {
	path := config.SkillStatePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return State{Active: map[string]bool{}}, nil
		}
		return State{}, output.NewError("CLI_ERROR", "读取本地 skill 状态失败", map[string]any{
			"path":    path,
			"details": err.Error(),
		})
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, output.NewError("CLI_ERROR", "解析本地 skill 状态失败", map[string]any{
			"path":    path,
			"details": err.Error(),
		})
	}
	if state.Active == nil {
		state.Active = map[string]bool{}
	}
	return state, nil
}

func SaveState(state State) error {
	if state.Active == nil {
		state.Active = map[string]bool{}
	}

	if err := os.MkdirAll(filepath.Dir(config.SkillStatePath()), 0o700); err != nil {
		return output.NewError("CLI_ERROR", "创建本地 skill 状态目录失败", map[string]any{
			"details": err.Error(),
		})
	}

	file, err := os.OpenFile(config.SkillStatePath(), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return output.NewError("CLI_ERROR", "写入本地 skill 状态失败", map[string]any{
			"path":    config.SkillStatePath(),
			"details": err.Error(),
		})
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(state); err != nil {
		return output.NewError("CLI_ERROR", "编码本地 skill 状态失败", map[string]any{
			"path":    config.SkillStatePath(),
			"details": err.Error(),
		})
	}
	return nil
}

func IsActive(slug string) (bool, error) {
	state, err := LoadState()
	if err != nil {
		return false, err
	}
	return state.Active[strings.TrimSpace(slug)], nil
}

func Activate(slug string) error {
	state, err := LoadState()
	if err != nil {
		return err
	}
	if state.Active == nil {
		state.Active = map[string]bool{}
	}
	state.Active[strings.TrimSpace(slug)] = true
	return SaveState(state)
}
