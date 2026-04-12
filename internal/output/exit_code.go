package output

import "strings"

func ExitCode(err error) int {
	if err == nil {
		return 0
	}

	cliErr, ok := err.(*CLIError)
	if !ok {
		return 1
	}

	code := strings.ToUpper(strings.TrimSpace(cliErr.Code))
	switch code {
	case "BAD_REQUEST", "VALIDATION_ERROR", "INPUT_ERROR", "INPUT_NOT_FOUND", "INPUT_PARSE_ERROR", "LOCAL_SKILL_INVALID", "NOT_FOUND", "NO_PROJECT", "CONFLICT", "LOCAL_SKILL_UNSUPPORTED", "LOCAL_ONLY_SKILL", "UNSUPPORTED_INSTALL", "CAPABILITY_UNAVAILABLE":
		return 2
	case "UNAUTHENTICATED", "FORBIDDEN":
		return 3
	case "RATE_LIMITED":
		return 4
	case "POLL_TIMEOUT":
		return 5
	case "NETWORK_ERROR", "SERVICE_UNAVAILABLE", "SERVER_ERROR", "HTTP_ERROR":
		return 6
	case "JOB_FAILED", "UPDATE_FAILED", "RUNTIME_SKILL_PLACEHOLDER":
		return 7
	case "CONTENT_FILTERED", "CONTENT_POLICY", "CONTENT_BLOCKED":
		return 10
	default:
		return 1
	}
}
