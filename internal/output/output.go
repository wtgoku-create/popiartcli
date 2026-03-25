package output

import (
	"encoding/json"
	"fmt"
	"io"
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
	cliErr, ok := err.(*CLIError)
	if !ok {
		cliErr = NewError("FATAL", err.Error(), nil)
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
