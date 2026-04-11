package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/api"
	"github.com/wtgoku-create/popiartcli/internal/config"
	"github.com/wtgoku-create/popiartcli/internal/output"
	"github.com/wtgoku-create/popiartcli/internal/types"
)

func newAuthCmd() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "管理身份验证和 API key",
	}

	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "验证并保存一个 API key",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			cfg := config.Load()

			key, _ := cmd.Flags().GetString("key")
			tokenAlias, _ := cmd.Flags().GetString("token")
			if key == "" {
				key = tokenAlias
			}

			if key == "" {
				if nonInteractiveMode(cmd) {
					return invalidFlagValueError("--key", "", "当前处于 --non-interactive 模式，请显式传入 --key")
				}
				value, err := promptPasswordTo(cmd.ErrOrStderr(), "Key: ")
				if err != nil {
					return output.NewError("CLI_ERROR", "读取 key 失败", map[string]any{"details": err.Error()})
				}
				key = value
			}

			client := api.NewClient(cfg.Endpoint, "")
			var resp types.LoginResponse
			if err := client.PostJSON(ctx, "/auth/login", map[string]any{
				"key": key,
			}, &resp); err != nil {
				return err
			}

			savedKey := resp.Token
			if savedKey == "" {
				savedKey = resp.Key
			}
			if savedKey == "" {
				savedKey = key
			}

			if _, err := config.SavePatch(config.Patch{Token: &savedKey}); err != nil {
				return output.NewError("CLI_ERROR", "保存 key 失败", map[string]any{"details": err.Error()})
			}

			return writeOutput(cmd, map[string]any{
				"user":      resp.User,
				"key_saved": true,
			})
		},
	}
	loginCmd.Flags().StringP("key", "k", "", "直接输入 API key")
	loginCmd.Flags().String("token", "", "兼容旧用法：等同于 --key")
	_ = loginCmd.Flags().MarkHidden("token")

	logoutCmd := &cobra.Command{
		Use:   "logout",
		Short: "撤销当前会话 key",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Load()
			if cfg.Token == "" {
				return writeOutput(cmd, map[string]any{
					"logged_out":        true,
					"was_authenticated": false,
				})
			}

			client := currentClient()
			_ = client.PostJSON(context.Background(), "/auth/logout", map[string]any{}, nil)
			empty := ""
			if _, err := config.SavePatch(config.Patch{Token: &empty}); err != nil {
				return output.NewError("CLI_ERROR", "清除令牌失败", map[string]any{"details": err.Error()})
			}
			return writeOutput(cmd, map[string]any{"logged_out": true})
		},
	}

	whoamiCmd := &cobra.Command{
		Use:   "whoami",
		Short: "显示当前已验证的用户",
		RunE: func(cmd *cobra.Command, args []string) error {
			var me types.AuthSession
			if err := currentClient().GetJSON(context.Background(), "/auth/me", nil, &me); err != nil {
				return err
			}
			if me.User == nil && me.ID != "" {
				return writeOutput(cmd, types.User{
					ID:     me.ID,
					Email:  me.Email,
					Name:   me.Name,
					Scopes: me.Scopes,
				})
			}
			return writeOutput(cmd, me)
		},
	}

	tokenCmd := &cobra.Command{
		Use:     "key",
		Aliases: []string{"token"},
		Short:   "管理 API key",
	}

	tokenShowCmd := &cobra.Command{
		Use:   "show",
		Short: "打印存储的 key（已脱敏）",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Load()
			if cfg.Token == "" {
				return requireTokenError()
			}
			masked := maskToken(cfg.Token)
			return writeOutput(cmd, map[string]any{
				"key":    masked,
				"config": config.Path(),
			})
		},
	}

	tokenSetCmd := &cobra.Command{
		Use:   "set <key>",
		Short: "直接存储 key 而不进行验证",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			if _, err := config.SavePatch(config.Patch{Token: &key}); err != nil {
				return output.NewError("CLI_ERROR", "保存 key 失败", map[string]any{"details": err.Error()})
			}
			return writeOutput(cmd, map[string]any{"key_saved": true})
		},
	}

	tokenRotateCmd := &cobra.Command{
		Use:   "rotate",
		Short: "签发新 key 并撤销旧 key",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp types.LoginResponse
			if err := currentClient().PostJSON(context.Background(), "/auth/token/rotate", map[string]any{}, &resp); err != nil {
				return err
			}
			token := resp.Token
			if _, err := config.SavePatch(config.Patch{Token: &token}); err != nil {
				return output.NewError("CLI_ERROR", "保存新 key 失败", map[string]any{"details": err.Error()})
			}
			return writeOutput(cmd, map[string]any{"key_rotated": true})
		},
	}

	tokenCmd.AddCommand(tokenShowCmd, tokenSetCmd, tokenRotateCmd)
	authCmd.AddCommand(loginCmd, logoutCmd, whoamiCmd, tokenCmd)
	return authCmd
}

func maskToken(token string) string {
	if len(token) <= 12 {
		return token
	}
	return token[:8] + "••••" + token[len(token)-4:]
}
