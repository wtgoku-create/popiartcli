package cmd

import (
	"context"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/localskills"
	"github.com/wtgoku-create/popiartcli/internal/output"
	"github.com/wtgoku-create/popiartcli/internal/seed"
	"github.com/wtgoku-create/popiartcli/internal/types"
)

const (
	popiartSkillhubRepoName = "wtgoku-create/Popiart_skillhub"
	popiartSkillhubRepoURL  = "https://github.com/wtgoku-create/Popiart_skillhub"
)

func newSkillsCmd() *cobra.Command {
	skillsCmd := &cobra.Command{
		Use:   "skills",
		Short: "在注册表中发现可用技能",
		Long: "查询 PopiArt 的技能发现链路。\n\n" +
			"当前真正可执行的 runtime skill 注册表来自 popiartServer 的 /skills 接口；" +
			"公开定义参考仓库当前是 " + popiartSkillhubRepoName + " (" + popiartSkillhubRepoURL + ")。\n\n" +
			"`popiart bootstrap` 生成的 default profile 会同时写入远程发现查询和 CLI 内置 bundled seed skills。" +
			"`skills list/get/schema` 会按优先级合并远程 runtime、已安装本地 skill、CLI 内置 official runtime baseline，以及剩余 bundled seed 元数据。",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "列出所有可用技能",
		Long: "列出当前可发现的技能，并按以下优先级合并显示：远程 runtime skills、已安装本地 skills、CLI 内置 official runtime baseline、CLI bundled seed skills。\n\n" +
			"远程技能注册表由 popiartServer 提供；" +
			"公开 skill 定义当前以 " + popiartSkillhubRepoName + " 为参考；" +
			"default skillset 只是 bootstrap 生成的发现入口，不等于服务端已注册的可执行集合。",
		Example: "  popiart skills list\n" +
			"  popiart skills list --tag image\n" +
			"  popiart skills list --search alice --limit 20 --offset 0",
		RunE: func(cmd *cobra.Command, args []string) error {
			tag, _ := cmd.Flags().GetString("tag")
			search, _ := cmd.Flags().GetString("search")
			limit, err := parseSkillsPaginationFlag(cmd, "limit", 50)
			if err != nil {
				return err
			}
			offset, err := parseSkillsPaginationFlag(cmd, "offset", 0)
			if err != nil {
				return err
			}

			localState, err := loadInstalledSkillState()
			if err != nil {
				return err
			}

			officialItems := matchingOfficialRuntimeSkillSummaries(tag, search)
			localItems := seed.MatchingBundledSkillSummaries(tag, search)
			installedItems := installedSkillSummaries(localState, tag, search)

			var resp types.SkillListResponse
			remoteAvailable := true
			err = currentClient().GetJSON(context.Background(), "/skills", map[string]string{
				"tag":    tag,
				"search": search,
				"limit":  strconv.Itoa(remotePageSize(limit, offset)),
				"offset": "0",
			}, &resp)
			if err != nil {
				if cliErr, ok := err.(*output.CLIError); !ok || cliErr.Code != "NETWORK_ERROR" {
					return err
				}
				remoteAvailable = false
			}
			resp.Items = annotateRemoteSkillSummaries(resp.Items, localState)

			if remoteAvailable {
				localItems, err = bundledSkillSummariesMissingOnRemote(context.Background(), localItems)
				if err != nil {
					return err
				}
				installedItems, err = installedSkillSummariesMissingOnRemote(context.Background(), installedItems)
				if err != nil {
					return err
				}
			}
			merged := mergeSkillSummaries(resp.Items, installedItems, officialItems, localItems)
			resp.Items = paginateSkillSummaries(merged, limit, offset)
			resp.Total += len(installedItems) + len(localItems) + countMissingSkillSummaries(resp.Items, installedItems, localItems, officialItems)
			resp.Limit = limit
			resp.Offset = offset
			return writeOutput(cmd, resp)
		},
	}
	listCmd.Flags().StringP("tag", "t", "", "按标签过滤")
	listCmd.Flags().StringP("search", "s", "", "全文搜索")
	listCmd.Flags().String("limit", "50", "最大结果数量")
	listCmd.Flags().String("offset", "0", "分页偏移量")

	getCmd := &cobra.Command{
		Use:   "get <skill-id>",
		Short: "获取技能的完整模式和描述",
		Long: "读取某个 skill 的完整描述、输入输出约束和来源信息。\n\n" +
			"查找顺序是：active installed skill -> 远程 runtime skill -> CLI 内置 official runtime fallback -> bundled seed skill。\n" +
			"source 字段会标明结果来自 remote、installed、official-runtime 或 bundled-seed。",
		Example: "  popiart skills get popiskill-image-text2image-basic-v1\n" +
			"  popiart skills get <skill-id> --plain",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			skillID, err := normalizeSkillLookupID(args[0])
			if err != nil {
				return err
			}
			localState, err := loadInstalledSkillState()
			if err != nil {
				return err
			}
			if skill, ok := activeInstalledSkill(localState, skillID); ok {
				return writeOutput(cmd, skill.Skill(true))
			}

			var skill types.Skill
			if err := currentClient().GetJSON(context.Background(), "/skills/"+skillID, nil, &skill); err != nil {
				if cliErr, ok := err.(*output.CLIError); ok && (cliErr.Code == "NOT_FOUND" || cliErr.Code == "NETWORK_ERROR") {
					if installed, ok := localState.byID[strings.ToLower(strings.TrimSpace(skillID))]; ok {
						return writeOutput(cmd, installed.Skill(localState.isActive(installed.Manifest.Slug)))
					}
					if skill, ok := officialRuntimeSkillForID(skillID); ok {
						return writeOutput(cmd, skill)
					}
					if skill, ok := seed.FindBundledSkill(skillID); ok {
						return writeOutput(cmd, skill)
					}
				}
				return err
			}
			skill = applyOfficialRuntimeSkillOverlay(skill)
			skill = annotateRemoteSkill(skill, localState)
			return writeOutput(cmd, skill)
		},
	}

	schemaCmd := &cobra.Command{
		Use:   "schema <skill-id>",
		Short: "打印某个技能的输入/输出 JSON 模式",
		Long: "读取某个 skill 的输入输出 JSON schema。\n\n" +
			"schema 查找顺序和 `skills get` 相同：active installed skill -> 远程 runtime skill -> CLI 内置 official runtime fallback -> bundled seed skill。",
		Example: "  popiart skills schema popiskill-image-text2image-basic-v1\n" +
			"  popiart skills schema <skill-id> --plain",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			skillID, err := normalizeSkillLookupID(args[0])
			if err != nil {
				return err
			}
			localState, err := loadInstalledSkillState()
			if err != nil {
				return err
			}
			if skill, ok := activeInstalledSkill(localState, skillID); ok {
				return writeOutput(cmd, skill.Schema())
			}

			var schema types.SkillSchemaResponse
			if err := currentClient().GetJSON(context.Background(), "/skills/"+skillID+"/schema", nil, &schema); err != nil {
				if cliErr, ok := err.(*output.CLIError); ok && (cliErr.Code == "NOT_FOUND" || cliErr.Code == "NETWORK_ERROR") {
					if installed, ok := localState.byID[strings.ToLower(strings.TrimSpace(skillID))]; ok {
						return writeOutput(cmd, installed.Schema())
					}
					if schema, ok := officialRuntimeSkillSchemaForID(skillID); ok {
						return writeOutput(cmd, schema)
					}
					if schema, ok := seed.FindBundledSkillSchema(skillID); ok {
						return writeOutput(cmd, schema)
					}
				}
				return err
			}
			schema = applyOfficialRuntimeSchemaOverlay(skillID, schema)
			return writeOutput(cmd, schema)
		},
	}

	skillsCmd.AddCommand(listCmd, getCmd, schemaCmd, newSkillsPullCmd(), newSkillsInstallCmd(), newSkillsUseLocalCmd())
	return skillsCmd
}

func parseSkillsPaginationFlag(cmd *cobra.Command, name string, fallback int) (int, error) {
	raw := strings.TrimSpace(flagString(cmd, name))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, output.NewError("VALIDATION_ERROR", "无效的分页参数", map[string]any{
			"flag":  name,
			"value": raw,
			"hint":  "请传入大于等于 0 的整数",
		})
	}
	return value, nil
}

func normalizeSkillLookupID(raw string) (string, error) {
	skillID := strings.TrimSpace(raw)
	if skillID == "" {
		return "", output.NewError("VALIDATION_ERROR", "skill_id 不能为空", map[string]any{
			"hint": "请传入类似 popiskill-image-text2image-basic-v1 的 skill id",
		})
	}
	return skillID, nil
}

func remotePageSize(limit, offset int) int {
	size := limit + offset
	if size < 1 {
		return 1
	}
	return size
}

func paginateSkillSummaries(items []types.SkillSummary, limit, offset int) []types.SkillSummary {
	if offset >= len(items) {
		return []types.SkillSummary{}
	}
	if offset < 0 {
		offset = 0
	}
	if limit < 0 {
		limit = 0
	}
	end := len(items)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return items[offset:end]
}

func mergeSkillSummaries(groups ...[]types.SkillSummary) []types.SkillSummary {
	seen := map[string]bool{}
	var size int
	for _, group := range groups {
		size += len(group)
	}
	merged := make([]types.SkillSummary, 0, size)
	appendUnique := func(items []types.SkillSummary) {
		for _, item := range items {
			key := strings.ToLower(strings.TrimSpace(item.ID))
			if key == "" {
				key = strings.ToLower(strings.TrimSpace(item.Name))
			}
			if key != "" && seen[key] {
				continue
			}
			if key != "" {
				seen[key] = true
			}
			merged = append(merged, item)
		}
	}
	for _, group := range groups {
		appendUnique(group)
	}
	return merged
}

func countMissingSkillSummaries(existingGroups ...[]types.SkillSummary) int {
	if len(existingGroups) == 0 {
		return 0
	}

	seen := map[string]bool{}
	for _, group := range existingGroups[:len(existingGroups)-1] {
		for _, item := range group {
			key := skillSummaryMergeKey(item)
			if key != "" {
				seen[key] = true
			}
		}
	}

	var count int
	for _, item := range existingGroups[len(existingGroups)-1] {
		key := skillSummaryMergeKey(item)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		count++
	}
	return count
}

func skillSummaryMergeKey(item types.SkillSummary) string {
	key := strings.ToLower(strings.TrimSpace(item.ID))
	if key == "" {
		key = strings.ToLower(strings.TrimSpace(item.Name))
	}
	return key
}

func bundledSkillSummariesMissingOnRemote(ctx context.Context, items []types.SkillSummary) ([]types.SkillSummary, error) {
	filtered := make([]types.SkillSummary, 0, len(items))
	for _, item := range items {
		exists, err := remoteSkillExists(ctx, item.ID)
		if err != nil {
			return nil, err
		}
		if !exists {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

func installedSkillSummariesMissingOnRemote(ctx context.Context, items []types.SkillSummary) ([]types.SkillSummary, error) {
	filtered := make([]types.SkillSummary, 0, len(items))
	for _, item := range items {
		if item.LocalActive {
			filtered = append(filtered, item)
			continue
		}
		exists, err := remoteSkillExists(ctx, item.ID)
		if err != nil {
			return nil, err
		}
		if !exists {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

func remoteSkillExists(ctx context.Context, skillID string) (bool, error) {
	var remoteSkill types.Skill
	if err := currentClient().GetJSON(ctx, "/skills/"+skillID, nil, &remoteSkill); err != nil {
		if cliErr, ok := err.(*output.CLIError); ok && cliErr.Code == "NOT_FOUND" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

type installedSkillState struct {
	byID   map[string]localskills.InstalledSkill
	active map[string]bool
}

func loadInstalledSkillState() (installedSkillState, error) {
	state, err := localskills.LoadState()
	if err != nil {
		return installedSkillState{}, err
	}
	items, err := localskills.ListInstalled()
	if err != nil {
		return installedSkillState{}, err
	}
	index := make(map[string]localskills.InstalledSkill, len(items))
	for _, item := range items {
		index[strings.ToLower(strings.TrimSpace(item.Manifest.Slug))] = item
	}
	return installedSkillState{
		byID:   index,
		active: state.Active,
	}, nil
}

func (s installedSkillState) isActive(slug string) bool {
	if s.active == nil {
		return false
	}
	return s.active[strings.TrimSpace(slug)]
}

func activeInstalledSkill(state installedSkillState, skillID string) (localskills.InstalledSkill, bool) {
	skill, ok := state.byID[strings.ToLower(strings.TrimSpace(skillID))]
	if !ok {
		return localskills.InstalledSkill{}, false
	}
	if !state.isActive(skill.Manifest.Slug) {
		return localskills.InstalledSkill{}, false
	}
	return skill, true
}

func installedSkillSummaries(state installedSkillState, tag, search string) []types.SkillSummary {
	items := make([]types.SkillSummary, 0, len(state.byID))
	for _, skill := range state.byID {
		if !skill.MatchesFilter(tag, search) {
			continue
		}
		items = append(items, skill.Summary(state.isActive(skill.Manifest.Slug)))
	}
	return items
}

func annotateRemoteSkillSummaries(items []types.SkillSummary, state installedSkillState) []types.SkillSummary {
	annotated := make([]types.SkillSummary, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Source) == "" {
			item.Source = "remote"
		}
		key := strings.ToLower(strings.TrimSpace(item.ID))
		if installed, ok := state.byID[key]; ok {
			if state.isActive(installed.Manifest.Slug) {
				continue
			}
			item.InstallDir = installed.RootDir
			item.LocalInstalled = true
			item.Source = "remote"
			item.ExecutionMode = installed.Manifest.Execution.Mode
			item.RuntimeSkillID = localskillsEffectiveRuntimeSkillID(installed)
			item.RequiresPopiartAuth = installed.Manifest.RequiresPopiartAuth
		}
		item = applyOfficialRuntimeSkillSummaryOverlay(item)
		annotated = append(annotated, item)
	}
	return annotated
}

func annotateRemoteSkill(skill types.Skill, state installedSkillState) types.Skill {
	if strings.TrimSpace(skill.Source) == "" {
		skill.Source = "remote"
	}
	key := strings.ToLower(strings.TrimSpace(skill.ID))
	installed, ok := state.byID[key]
	if !ok || state.isActive(installed.Manifest.Slug) {
		return skill
	}
	skill.Source = "remote"
	skill.InstallDir = installed.RootDir
	skill.LocalInstalled = true
	skill.ExecutionMode = installed.Manifest.Execution.Mode
	skill.RuntimeSkillID = localskillsEffectiveRuntimeSkillID(installed)
	skill.RequiresPopiartAuth = installed.Manifest.RequiresPopiartAuth
	return skill
}

func localskillsEffectiveRuntimeSkillID(skill localskills.InstalledSkill) string {
	if strings.TrimSpace(skill.Manifest.Execution.RuntimeSkillID) != "" {
		return strings.TrimSpace(skill.Manifest.Execution.RuntimeSkillID)
	}
	return skill.Manifest.Slug
}
