package cmd

import (
	"context"

	"github.com/spf13/cobra"

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
			limit, _ := cmd.Flags().GetString("limit")
			offset, _ := cmd.Flags().GetString("offset")

			var resp types.SkillListResponse
			if err := currentClient().GetJSON(context.Background(), "/skills", map[string]string{
				"tag":    tag,
				"search": search,
				"limit":  limit,
				"offset": offset,
			}, &resp); err != nil {
				return err
			}
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
				return err
			}
			return writeOutput(cmd, schema)
		},
	}

	skillsCmd.AddCommand(listCmd, getCmd, schemaCmd)
	return skillsCmd
}
