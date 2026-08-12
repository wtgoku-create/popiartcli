package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/popiart"
)

func newVoicesCmd() *cobra.Command {
	voicesCmd := &cobra.Command{
		Use:   "voices",
		Short: "查询可用于语音合成的音色",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "列出可用音色并返回 voice_id",
		RunE: func(cmd *cobra.Command, args []string) error {
			pageSize := flagString(cmd, "page-size")
			if cmd.Flags().Changed("pageSize") {
				pageSize = flagString(cmd, "pageSize")
			}
			page, err := popiart.ListVoices(context.Background(), currentClient(), flagString(cmd, "page"), pageSize)
			if err != nil {
				return err
			}

			items := make([]map[string]any, 0, len(page.List))
			for _, voice := range page.List {
				if voice.VoiceID == "" {
					continue
				}
				item := map[string]any{
					"voice_id": voice.VoiceID,
				}
				if voice.Description != "" {
					item["description"] = voice.Description
				}
				if voice.Prompt != "" {
					item["prompt"] = voice.Prompt
				}
				if voice.Name != "" {
					item["name"] = voice.Name
				}
				if voice.VoiceType != "" {
					item["voice_type"] = voice.VoiceType
				}
				if voice.Source != "" {
					item["source"] = voice.Source
				}
				items = append(items, item)
			}

			return writeOutput(cmd, map[string]any{
				"default_voice_id": popiart.DefaultVoiceID,
				"items":            items,
				"page_info":        page.PageInfo,
				"total":            len(items),
			})
		},
	}
	listCmd.Flags().String("page", "1", "页码")
	listCmd.Flags().String("page-size", "20", "每页数量")
	listCmd.Flags().String("pageSize", "", "每页数量（兼容接口字段名）")

	voicesCmd.AddCommand(listCmd)
	return voicesCmd
}
