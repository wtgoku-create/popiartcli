package cmd

import (
	"context"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/output"
	"github.com/wtgoku-create/popiartcli/internal/seed"
	"github.com/wtgoku-create/popiartcli/internal/types"
)

func newSkillsCmd() *cobra.Command {
	skillsCmd := &cobra.Command{
		Use:   "skills",
		Short: "在注册表中发现可用技能",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "列出所有可用技能",
		RunE: func(cmd *cobra.Command, args []string) error {
			tag, _ := cmd.Flags().GetString("tag")
			search, _ := cmd.Flags().GetString("search")
			limitRaw, _ := cmd.Flags().GetString("limit")
			offsetRaw, _ := cmd.Flags().GetString("offset")
			limit := parseNonNegativeInt(limitRaw, 50)
			offset := parseNonNegativeInt(offsetRaw, 0)

			localItems := seed.MatchingBundledSkillSummaries(tag, search)

			var resp types.SkillListResponse
			if err := currentClient().GetJSON(context.Background(), "/skills", map[string]string{
				"tag":    tag,
				"search": search,
				"limit":  strconv.Itoa(remotePageSize(limit, offset)),
				"offset": "0",
			}, &resp); err != nil {
				return err
			}
			localItems, err := bundledSkillSummariesMissingOnRemote(context.Background(), localItems)
			if err != nil {
				return err
			}
			merged := mergeSkillSummaries(resp.Items, localItems)
			resp.Items = paginateSkillSummaries(merged, limit, offset)
			resp.Total += len(localItems)
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var skill types.Skill
			if err := currentClient().GetJSON(context.Background(), "/skills/"+args[0], nil, &skill); err != nil {
				if cliErr, ok := err.(*output.CLIError); ok && cliErr.Code == "NOT_FOUND" {
					if skill, ok := seed.FindBundledSkill(args[0]); ok {
						return writeOutput(cmd, skill)
					}
				}
				return err
			}
			return writeOutput(cmd, skill)
		},
	}

	schemaCmd := &cobra.Command{
		Use:   "schema <skill-id>",
		Short: "打印某个技能的输入/输出 JSON 模式",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var schema types.SkillSchemaResponse
			if err := currentClient().GetJSON(context.Background(), "/skills/"+args[0]+"/schema", nil, &schema); err != nil {
				if cliErr, ok := err.(*output.CLIError); ok && cliErr.Code == "NOT_FOUND" {
					if schema, ok := seed.FindBundledSkillSchema(args[0]); ok {
						return writeOutput(cmd, schema)
					}
				}
				return err
			}
			return writeOutput(cmd, schema)
		},
	}

	skillsCmd.AddCommand(listCmd, getCmd, schemaCmd)
	return skillsCmd
}

func parseNonNegativeInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return fallback
	}
	return value
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

func mergeSkillSummaries(primary []types.SkillSummary, secondary []types.SkillSummary) []types.SkillSummary {
	seen := map[string]bool{}
	merged := make([]types.SkillSummary, 0, len(primary)+len(secondary))
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
	appendUnique(primary)
	appendUnique(secondary)
	return merged
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
