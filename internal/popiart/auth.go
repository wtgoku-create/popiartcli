package popiart

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/wtgoku-create/popiartcli/internal/api"
)

type userInfoEnvelope struct {
	User  userInfoPayload `json:"user"`
	ID    any             `json:"id"`
	Email string          `json:"email"`
	Name  string          `json:"name"`
}

type userInfoPayload struct {
	ID    any    `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// FetchCurrentUser 使用 Bearer token 调用主站 user/info 接口，并兼容直出与 data.user 两种结构。
func FetchCurrentUser(ctx context.Context, client *api.Client) (UserInfo, error) {
	var payload userInfoEnvelope
	if err := client.GetJSON(ctx, "/api_client/users/user/info", nil, &payload); err != nil {
		return UserInfo{}, NormalizeAPIError(err)
	}

	id := payload.User.ID
	email := payload.User.Email
	name := payload.User.Name
	if email == "" && name == "" && id == nil {
		id = payload.ID
		email = payload.Email
		name = payload.Name
	}

	return UserInfo{
		ID:    stringifyUserID(id),
		Email: email,
		Name:  name,
	}, nil
}

// stringifyUserID 把主站返回的数字或字符串用户 ID 统一转成 CLI 输出使用的字符串。
func stringifyUserID(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case json.Number:
		return typed.String()
	default:
		return ""
	}
}
