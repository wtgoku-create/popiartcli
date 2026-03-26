package cmd

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"

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
			resp.Items = paginateSkillSummaries(append(localItems, resp.Items...), limit, offset)
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
			if skill, ok := seed.FindBundledSkill(args[0]); ok {
				return writeOutput(cmd, skill)
			}
			var skill types.Skill
			if err := currentClient().GetJSON(context.Background(), "/skills/"+args[0], nil, &skill); err != nil {
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
			if schema, ok := seed.FindBundledSkillSchema(args[0]); ok {
				return writeOutput(cmd, schema)
			}
			var schema types.SkillSchemaResponse
			if err := currentClient().GetJSON(context.Background(), "/skills/"+args[0]+"/schema", nil, &schema); err != nil {
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
