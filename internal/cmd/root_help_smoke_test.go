package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestTopLevelCommandHelpSmoke(t *testing.T) {
	commands := []string{
		"audio",
		"artifacts",
		"auth",
		"bootstrap",
		"budget",
		"completion",
		"image",
		"jobs",
		"mcp",
		"media",
		"models",
		"project",
		"run",
		"setup",
		"skills",
		"update",
		"video",
	}

	for _, name := range commands {
		t.Run(name, func(t *testing.T) {
			root := NewRootCmd("0.test")
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			root.SetOut(&stdout)
			root.SetErr(&stderr)
			root.SetContext(context.Background())
			root.SetArgs([]string{name, "--help"})

			if err := root.Execute(); err != nil {
				t.Fatalf("%s --help error = %v", name, err)
			}

			combined := stdout.String() + "\n" + stderr.String()
			if !strings.Contains(combined, name) {
				t.Fatalf("%s --help output did not mention command name; output=%q", name, combined)
			}
		})
	}
}
