package input

import (
	"os"
	"testing"
)

func TestResolveReadsFromStdin(t *testing.T) {
	oldStdin := os.Stdin
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	t.Cleanup(func() {
		os.Stdin = oldStdin
		reader.Close()
	})
	os.Stdin = reader

	if _, err := writer.WriteString(`{"goal":"from-stdin"}`); err != nil {
		t.Fatalf("write stdin: %v", err)
	}
	writer.Close()

	value, err := Resolve("-")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	payload, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected map payload, got %T", value)
	}
	if payload["goal"] != "from-stdin" {
		t.Fatalf("expected goal from stdin, got %#v", payload["goal"])
	}
}
