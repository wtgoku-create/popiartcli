package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/api"
	"github.com/wtgoku-create/popiartcli/internal/config"
	"github.com/wtgoku-create/popiartcli/internal/output"
	"github.com/wtgoku-create/popiartcli/internal/popiart"
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

			client := api.NewClient(cfg.Endpoint, key)
			user, err := popiart.FetchCurrentUser(ctx, client)
			if err != nil {
				return err
			}

			if _, err := config.SavePatch(config.Patch{Token: &key}); err != nil {
				return output.NewError("CLI_ERROR", "保存 key 失败", map[string]any{"details": err.Error()})
			}

			return writeOutput(cmd, map[string]any{
				"user": types.User{
					ID:     user.ID,
					Email:  user.Email,
					Name:   user.Name,
					Scopes: user.Scopes,
				},
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
			user, err := popiart.FetchCurrentUser(context.Background(), currentClient())
			if err != nil {
				return err
			}
			return writeOutput(cmd, types.User{
				ID:     user.ID,
				Email:  user.Email,
				Name:   user.Name,
				Scopes: user.Scopes,
			})
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
			return output.NewError("UNSUPPORTED_IN_POPI_ART_MODE", "当前模式不支持轮换 key", map[string]any{
				"hint": "请改用 `popiart auth login --key <token>` 或 `popiart auth key set <token>`",
			})
		},
	}

	tokenCmd.AddCommand(tokenShowCmd, tokenSetCmd, tokenRotateCmd)
	authCmd.AddCommand(loginCmd, logoutCmd, whoamiCmd, tokenCmd)
	return authCmd
}

func maskToken(token string) string {
	if len(token) <= 4 {
		return "••••"
	}
	if len(token) <= 12 {
		return token[:2] + "••••"
	}
	return token[:8] + "••••" + token[len(token)-4:]
}
