package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

type CLIError struct {
	Code    string
	Message string
	Details map[string]any
}

func (e *CLIError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func NewError(code, message string, details map[string]any) *CLIError {
	if details == nil {
		details = map[string]any{}
	}
	return &CLIError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

func WriteData(w io.Writer, data any, plain bool) error {
	if plain {
		printPlain(w, data, 0)
		return nil
	}
	return json.NewEncoder(w).Encode(map[string]any{
		"ok":   true,
		"data": data,
	})
}

func WriteError(w io.Writer, err error) {
	WriteErrorWithMode(w, err, false)
}

func WriteErrorWithMode(w io.Writer, err error, plain bool) {
	cliErr, ok := err.(*CLIError)
	if !ok {
		cliErr = NewError("FATAL", err.Error(), nil)
	}
	maybeHandleRecharge(cliErr)

	if plain {
		fmt.Fprintln(w, cliErr.Message)
		if cliErr.Code != "" {
			fmt.Fprintf(w, "code: %s\n", cliErr.Code)
		}
		if len(cliErr.Details) > 0 {
			keys := make([]string, 0, len(cliErr.Details))
			for key := range cliErr.Details {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				fmt.Fprintf(w, "%s: %v\n", key, cliErr.Details[key])
			}
		}
		return
	}

	payload := map[string]any{
		"ok": false,
		"error": map[string]any{
			"code":    cliErr.Code,
			"message": cliErr.Message,
		},
	}

	if len(cliErr.Details) > 0 {
		for k, v := range cliErr.Details {
			payload["error"].(map[string]any)[k] = v
		}
	}

	_ = json.NewEncoder(w).Encode(payload)
}

func WantsPlainOutput(args []string) bool {
	for idx := 0; idx < len(args); idx++ {
		value := strings.TrimSpace(args[idx])
		switch {
		case value == "--plain":
			return true
		case value == "--output" && idx+1 < len(args):
			if strings.EqualFold(strings.TrimSpace(args[idx+1]), "plain") {
				return true
			}
			idx++
		case strings.HasPrefix(value, "--output="):
			mode := strings.TrimSpace(strings.TrimPrefix(value, "--output="))
			if strings.EqualFold(mode, "plain") {
				return true
			}
		}
	}
	return false
}

func printPlain(w io.Writer, data any, indent int) {
	prefix := ""
	for i := 0; i < indent; i++ {
		prefix += "  "
	}

	switch v := data.(type) {
	case nil:
		return
	case string:
		fmt.Fprintln(w, v)
	case []any:
		for _, item := range v {
			printPlain(w, item, indent)
		}
	case map[string]any:
		for key, value := range v {
			switch value.(type) {
			case map[string]any, []any:
				fmt.Fprintf(w, "%s%s:\n", prefix, key)
				printPlain(w, value, indent+1)
			default:
				fmt.Fprintf(w, "%s%s: %v\n", prefix, key, value)
			}
		}
	default:
		fmt.Fprintf(w, "%s%v\n", prefix, v)
	}
}
