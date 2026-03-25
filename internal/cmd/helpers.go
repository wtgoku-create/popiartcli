package cmd

import (
	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/api"
)

func flagString(cmd *cobra.Command, name string) string {
	value, _ := cmd.Flags().GetString(name)
	return value
}

func flagBool(cmd *cobra.Command, name string) bool {
	value, _ := cmd.Flags().GetBool(name)
	return value
}

func apiOpts(token, accept string) api.RequestOptions {
	return api.RequestOptions{
		Token:  token,
		Accept: accept,
	}
}
