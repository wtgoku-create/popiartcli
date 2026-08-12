package popiart

import (
	"context"
	"strings"

	"github.com/wtgoku-create/popiartcli/internal/api"
)

const DefaultVoiceID = "male-qn-qingse"

type VoiceLibraryPage struct {
	PageInfo map[string]any `json:"pageInfo,omitempty"`
	List     []VoiceObject  `json:"list,omitempty"`
}

type VoiceObject struct {
	ID               int64  `json:"id,omitempty"`
	UserID           int64  `json:"userId,omitempty"`
	AIModelID        int64  `json:"aiModelId,omitempty"`
	AIModelCode      string `json:"aiModelCode,omitempty"`
	AIModelCodeAlias string `json:"aiModelCodeAlias,omitempty"`
	VoiceID          string `json:"voiceId,omitempty"`
	VoiceType        string `json:"voiceType,omitempty"`
	Source           string `json:"source,omitempty"`
	IsPublic         bool   `json:"isPublic,omitempty"`
	Name             string `json:"name,omitempty"`
	Cover            string `json:"cover,omitempty"`
	AudioURL         string `json:"audioUrl,omitempty"`
	Description      string `json:"desp,omitempty"`
	Prompt           string `json:"prompt,omitempty"`
	UserLiked        bool   `json:"userLiked,omitempty"`
}

func ListVoices(ctx context.Context, client *api.Client, page, pageSize string) (VoiceLibraryPage, error) {
	query := map[string]string{
		"page":     strings.TrimSpace(page),
		"pageSize": strings.TrimSpace(pageSize),
	}

	var payload VoiceLibraryPage
	if err := client.GetJSON(ctx, "/api_client/anime/voiceLibrary/list", query, &payload); err != nil {
		return VoiceLibraryPage{}, NormalizeAPIError(err)
	}
	return payload, nil
}
