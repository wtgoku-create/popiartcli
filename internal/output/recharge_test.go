package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

func TestShouldPromptRechargeMatchesBudgetSignals(t *testing.T) {
	tests := []struct {
		name string
		err  *CLIError
		want bool
	}{
		{
			name: "matches 402",
			err: &CLIError{
				Code:    "HTTP_402",
				Message: "payment required",
				Details: map[string]any{"status": 402},
			},
			want: true,
		},
		{
			name: "matches insufficient credits message",
			err: &CLIError{
				Code:    "BUDGET_EXCEEDED",
				Message: "insufficient credits for this job",
			},
			want: true,
		},
		{
			name: "matches chinese text",
			err: &CLIError{
				Code:    "BAD_REQUEST",
				Message: "当前积分不足，请先充值",
			},
			want: true,
		},
		{
			name: "ignores unrelated auth error",
			err: &CLIError{
				Code:    "UNAUTHENTICATED",
				Message: "token expired",
				Details: map[string]any{"status": 401},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldPromptRecharge(tt.err); got != tt.want {
				t.Fatalf("shouldPromptRecharge() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWriteErrorAddsRechargeMetadataWhenOpenSucceeds(t *testing.T) {
	prev := openRechargeURL
	t.Cleanup(func() { openRechargeURL = prev })

	openRechargeURL = func(target string) (bool, error) {
		if target != rechargeURL {
			t.Fatalf("unexpected recharge target: %s", target)
		}
		return true, nil
	}

	err := &CLIError{
		Code:    "BUDGET_EXCEEDED",
		Message: "insufficient credits",
	}

	var buf bytes.Buffer
	WriteError(&buf, err)

	var payload struct {
		OK    bool `json:"ok"`
		Error struct {
			Code             string `json:"code"`
			Message          string `json:"message"`
			RechargeURL      string `json:"recharge_url"`
			RechargePrompted bool   `json:"recharge_prompted"`
			RechargeOpened   bool   `json:"recharge_opened"`
		} `json:"error"`
	}
	if decodeErr := json.Unmarshal(buf.Bytes(), &payload); decodeErr != nil {
		t.Fatalf("unmarshal payload: %v", decodeErr)
	}
	if payload.OK {
		t.Fatal("expected ok=false payload")
	}
	if payload.Error.RechargeURL != rechargeURL {
		t.Fatalf("expected recharge URL %q, got %q", rechargeURL, payload.Error.RechargeURL)
	}
	if !payload.Error.RechargePrompted {
		t.Fatal("expected recharge_prompted=true")
	}
	if !payload.Error.RechargeOpened {
		t.Fatal("expected recharge_opened=true")
	}
}

func TestWriteErrorIncludesOpenFailureDetails(t *testing.T) {
	prev := openRechargeURL
	t.Cleanup(func() { openRechargeURL = prev })

	openRechargeURL = func(target string) (bool, error) {
		return false, errors.New("xdg-open missing")
	}

	err := &CLIError{
		Code:    "PAYMENT_REQUIRED",
		Message: "余额不足",
	}

	var buf bytes.Buffer
	WriteError(&buf, err)

	var payload map[string]any
	if decodeErr := json.Unmarshal(buf.Bytes(), &payload); decodeErr != nil {
		t.Fatalf("unmarshal payload: %v", decodeErr)
	}

	errorPayload, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error payload, got %#v", payload["error"])
	}
	if errorPayload["recharge_opened"] != false {
		t.Fatalf("expected recharge_opened=false, got %#v", errorPayload["recharge_opened"])
	}
	if errorPayload["recharge_open_error"] != "xdg-open missing" {
		t.Fatalf("unexpected recharge_open_error: %#v", errorPayload["recharge_open_error"])
	}
}
