package output

import "testing"

func TestExitCodeMapping(t *testing.T) {
	tests := []struct {
		code string
		want int
	}{
		{code: "VALIDATION_ERROR", want: 2},
		{code: "NOT_FOUND", want: 2},
		{code: "CAPABILITY_UNAVAILABLE", want: 2},
		{code: "UNAUTHENTICATED", want: 3},
		{code: "RATE_LIMITED", want: 4},
		{code: "POLL_TIMEOUT", want: 5},
		{code: "NETWORK_ERROR", want: 6},
		{code: "JOB_FAILED", want: 7},
		{code: "RUNTIME_SKILL_PLACEHOLDER", want: 7},
		{code: "CONTENT_POLICY", want: 10},
		{code: "SOMETHING_ELSE", want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			if got := ExitCode(NewError(tt.code, "demo", nil)); got != tt.want {
				t.Fatalf("ExitCode(%q) = %d, want %d", tt.code, got, tt.want)
			}
		})
	}
}
