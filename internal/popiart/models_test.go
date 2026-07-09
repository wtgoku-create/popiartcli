package popiart

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wtgoku-create/popiartcli/internal/api"
	"github.com/wtgoku-create/popiartcli/internal/output"
)

func TestFetchModelsAndResolveCandidateModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api_client/anime/ai/model/list" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("origin"); got != "web" {
			t.Fatalf("expected origin=web, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":[{"id":101,"code":"seedance-main","aiModelCodeAlias":"seedance-alias","videoRatio":["16:9"],"duration":[5,10],"isSupportImages":true,"categories":[{"taskSubType":203}]}]}`)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "pk-demo")
	models, err := FetchModels(context.Background(), client)
	if err != nil {
		t.Fatalf("FetchModels returned error: %v", err)
	}
	model, err := ResolveCandidateModel(models, "101", nil, 203)
	if err != nil {
		t.Fatalf("ResolveCandidateModel returned error: %v", err)
	}
	if model.Code != "seedance-main" {
		t.Fatalf("unexpected model: %#v", model)
	}
}

func TestResolveCandidateModelRequiresExplicitModelID(t *testing.T) {
	models := []Model{
		{
			ID:               101,
			Code:             "seedance-main",
			AIModelCodeAlias: StringList{"seedance-alias"},
			Categories:       []ModelCategory{{TaskSubType: 203}},
		},
	}

	if _, err := ResolveCandidateModel(models, "seedance-alias", nil, 203); err == nil {
		t.Fatal("expected alias lookup to fail for explicit --model")
	}

	model, err := ResolveCandidateModel(models, "", []string{"seedance-alias"}, 203)
	if err != nil {
		t.Fatalf("expected default alias lookup to keep working, got %v", err)
	}
	if model.ID != 101 {
		t.Fatalf("unexpected default model: %#v", model)
	}
}

func TestValidateModelSupportRejectsUnsupportedSubtype(t *testing.T) {
	model := Model{
		Code:            "seedance-main",
		IsSupportImages: true,
		Categories:      []ModelCategory{{TaskSubType: 203}},
	}

	err := ValidateModelSupport(model, ModelValidationSpec{
		SubType:        204,
		RequiresImages: true,
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	cliErr, ok := err.(*output.CLIError)
	if !ok {
		t.Fatalf("expected CLIError, got %T", err)
	}
	if cliErr.Code != "MODEL_SUBTYPE_UNSUPPORTED" {
		t.Fatalf("unexpected error code: %q", cliErr.Code)
	}
}

func TestValidateModelSupportAllowsAnyMatchingAllowedSubtype(t *testing.T) {
	model := Model{
		Code:       "video-main",
		Categories: []ModelCategory{{TaskSubType: 203}},
	}

	err := ValidateModelSupport(model, ModelValidationSpec{
		AllowedSubTypes: []int{202, 203, 204},
	})
	if err != nil {
		t.Fatalf("expected allowed subtype set to pass, got %v", err)
	}
}

func TestPreferredSubTypeReturnsFirstSupported(t *testing.T) {
	model := Model{
		Code:       "video-main",
		Categories: []ModelCategory{{TaskSubType: 203}, {TaskSubType: 204}},
	}

	if got := PreferredSubType(model, []int{202, 204, 203}); got != 204 {
		t.Fatalf("expected preferred subtype 204, got %d", got)
	}
}

func TestValidateModelSupportTreatsZeroUploadLimitAsUnlimited(t *testing.T) {
	zero := 0
	model := Model{
		Code:             "image-main",
		IsSupportImages:  true,
		UploadImageLimit: FlexibleLimit{Value: &zero},
		Categories:       []ModelCategory{{TaskSubType: 103}},
	}

	err := ValidateModelSupport(model, ModelValidationSpec{
		SubType:        103,
		RequiresImages: true,
		ImageCount:     1,
	})
	if err != nil {
		t.Fatalf("expected zero upload limit to pass as unlimited, got %v", err)
	}
}

func TestValidateModelSupportRejectsImageCountOverPositiveLimit(t *testing.T) {
	imageLimit := 1
	model := Model{
		Code:             "image-main",
		IsSupportImages:  true,
		UploadImageLimit: FlexibleLimit{Value: &imageLimit},
		Categories:       []ModelCategory{{TaskSubType: 103}},
	}

	err := ValidateModelSupport(model, ModelValidationSpec{
		SubType:        103,
		RequiresImages: true,
		ImageCount:     2,
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	cliErr, ok := err.(*output.CLIError)
	if !ok {
		t.Fatalf("expected CLIError, got %T", err)
	}
	if cliErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("unexpected error code: %q", cliErr.Code)
	}
	if cliErr.Message != "参考图数量超过模型限制" {
		t.Fatalf("unexpected error message: %q", cliErr.Message)
	}
}

func TestValidateModelSupportRejectsVideoCountOverPositiveLimitAndAllowsAudioZeroUnlimited(t *testing.T) {
	videoLimit := 1
	audioLimit := 0
	model := Model{
		Code:             "seedance-main",
		IsSupportVideos:  true,
		IsSupportAudios:  true,
		UploadVideoLimit: FlexibleLimit{Value: &videoLimit},
		UploadAudioLimit: FlexibleLimit{Value: &audioLimit},
		Categories:       []ModelCategory{{TaskSubType: 203}},
	}

	err := ValidateModelSupport(model, ModelValidationSpec{
		SubType:        203,
		RequiresVideos: true,
		RequiresAudios: true,
		VideoCount:     2,
		AudioCount:     1,
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	cliErr, ok := err.(*output.CLIError)
	if !ok {
		t.Fatalf("expected CLIError, got %T", err)
	}
	if cliErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("unexpected error code: %q", cliErr.Code)
	}
	if cliErr.Message != "参考视频数量超过模型限制" {
		t.Fatalf("unexpected error message: %q", cliErr.Message)
	}

	err = ValidateModelSupport(Model{
		Code:             "audio-main",
		IsSupportAudios:  true,
		UploadAudioLimit: FlexibleLimit{Value: &audioLimit},
		Categories:       []ModelCategory{{TaskSubType: 203}},
	}, ModelValidationSpec{
		SubType:        203,
		RequiresAudios: true,
		AudioCount:     1,
	})
	if err != nil {
		t.Fatalf("expected zero audio upload limit to pass as unlimited, got %v", err)
	}
}

func TestFetchModelsSupportsWebEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api_client/anime/ai/model/list" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("origin"); got != "web" {
			t.Fatalf("expected origin=web, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[{"id":202,"code":"image-main","categories":[{"taskSubType":103}],"isSupportImages":true}],"message":"ok","status":"0000"}`)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "pk-demo")
	models, err := FetchModels(context.Background(), client)
	if err != nil {
		t.Fatalf("FetchModels returned error: %v", err)
	}
	if len(models) != 1 || models[0].Code != "image-main" {
		t.Fatalf("unexpected models payload: %#v", models)
	}
}

func TestFetchModelsSupportsNumericVideoRatioValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api_client/anime/ai/model/list" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("origin"); got != "web" {
			t.Fatalf("expected origin=web, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[{"id":303,"code":"video-main","videoRatio":[16,9,"21:9"],"categories":[{"taskSubType":203}]}],"message":"ok","status":"0000"}`)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "pk-demo")
	models, err := FetchModels(context.Background(), client)
	if err != nil {
		t.Fatalf("FetchModels returned error: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("unexpected models length: %#v", models)
	}
	if got := []string(models[0].VideoRatio); len(got) != 3 || got[0] != "16" || got[1] != "9" || got[2] != "21:9" {
		t.Fatalf("unexpected videoRatio payload: %#v", got)
	}
}

func TestFetchModelsSupportsArrayUploadLimits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api_client/anime/ai/model/list" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("origin"); got != "web" {
			t.Fatalf("expected origin=web, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":[{"id":404,"code":"image-main","uploadImageLimit":[1,4],"uploadVideoLimit":["0"],"uploadAudioLimit":null,"categories":[{"taskSubType":103}]}]}`)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "pk-demo")
	models, err := FetchModels(context.Background(), client)
	if err != nil {
		t.Fatalf("FetchModels returned error: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("unexpected models length: %#v", models)
	}
	if models[0].UploadImageLimit.Value == nil || *models[0].UploadImageLimit.Value != 4 {
		t.Fatalf("unexpected uploadImageLimit normalization: %#v", models[0].UploadImageLimit.Value)
	}
	if models[0].UploadVideoLimit.Value == nil || *models[0].UploadVideoLimit.Value != 0 {
		t.Fatalf("unexpected uploadVideoLimit normalization: %#v", models[0].UploadVideoLimit.Value)
	}
	if models[0].UploadAudioLimit.Value != nil {
		t.Fatalf("expected nil uploadAudioLimit, got %#v", models[0].UploadAudioLimit.Value)
	}
}

func TestResolveCandidateModelPrefersSubtypeCompatibleDuplicate(t *testing.T) {
	models := []Model{
		{
			ID:               1,
			Code:             "Nano-banana-pro",
			AIModelCodeAlias: StringList{"gemini-3-pro-image-preview"},
			Categories:       []ModelCategory{{TaskSubType: 103, Status: intPtr(0)}},
		},
		{
			ID:               23,
			Code:             "Nano-banana-pro",
			AIModelCodeAlias: StringList{"gemini-3-pro-image-preview"},
			Categories:       []ModelCategory{{TaskSubType: 106, Status: intPtr(1)}},
		},
		{
			ID:               57,
			Code:             "doubao-seedream-4-5-251128",
			AIModelCodeAlias: StringList{"doubao-seedream-4-5-251128"},
			Categories:       []ModelCategory{{TaskSubType: 103, Status: intPtr(1)}},
		},
	}

	model, err := ResolveCandidateModel(models, "", []string{"seedream-4-5-251128", "gemini-3-pro-image-preview"}, 103)
	if err != nil {
		t.Fatalf("expected subtype-compatible duplicate to resolve, got %v", err)
	}
	if model.ID != 1 {
		t.Fatalf("expected status=0 compatible duplicate to be selected, got %#v", model)
	}
}

func intPtr(v int) *int {
	return &v
}
