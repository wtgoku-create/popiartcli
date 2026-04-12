package cmd

import (
	"encoding/json"
	"testing"
)

func TestExportSchemaAnthropicSingleCommand(t *testing.T) {
	root := NewRootCmd("0.test")

	stdout, stderr, err := executeRootRaw(root, []string{
		"export-schema",
		"--command", "image generate",
		"--format", "anthropic",
	})
	if err != nil {
		t.Fatalf("export-schema failed: %v stderr=%s", err, stderr)
	}

	var tools []map[string]any
	if err := json.Unmarshal([]byte(stdout), &tools); err != nil {
		t.Fatalf("unmarshal export-schema output: %v output=%q", err, stdout)
	}
	if len(tools) != 1 {
		t.Fatalf("expected one tool, got %d", len(tools))
	}
	if tools[0]["name"] != "popiart_image_generate" {
		t.Fatalf("unexpected tool name: %#v", tools[0]["name"])
	}
	inputSchema := tools[0]["input_schema"].(map[string]any)
	required := inputSchema["required"].([]any)
	if len(required) == 0 || required[0] != "prompt" {
		t.Fatalf("expected prompt to be required, got %#v", required)
	}
	properties := inputSchema["properties"].(map[string]any)
	if properties["aspect_ratio"] == nil {
		t.Fatalf("expected aspect_ratio property, got %#v", properties)
	}
}

func TestExportSchemaIncludesRunnableParentSugarCommand(t *testing.T) {
	root := NewRootCmd("0.test")

	stdout, stderr, err := executeRootRaw(root, []string{
		"export-schema",
		"--command", "image",
		"--format", "generic",
	})
	if err != nil {
		t.Fatalf("export-schema failed: %v stderr=%s", err, stderr)
	}

	var tools []map[string]any
	if err := json.Unmarshal([]byte(stdout), &tools); err != nil {
		t.Fatalf("unmarshal export-schema output: %v output=%q", err, stdout)
	}
	if len(tools) != 1 {
		t.Fatalf("expected one tool, got %d", len(tools))
	}
	properties := tools[0]["input_schema"].(map[string]any)["properties"].(map[string]any)
	if properties["prompt"] == nil {
		t.Fatalf("expected prompt property, got %#v", properties)
	}
}

func TestExportSchemaOpenAICompletionCommandUsesShellEnum(t *testing.T) {
	root := NewRootCmd("0.test")

	stdout, stderr, err := executeRootRaw(root, []string{
		"export-schema",
		"--command", "completion",
		"--format", "openai",
	})
	if err != nil {
		t.Fatalf("export-schema failed: %v stderr=%s", err, stderr)
	}

	var tools []map[string]any
	if err := json.Unmarshal([]byte(stdout), &tools); err != nil {
		t.Fatalf("unmarshal export-schema output: %v output=%q", err, stdout)
	}
	if len(tools) != 1 {
		t.Fatalf("expected one tool, got %d", len(tools))
	}
	if tools[0]["type"] != "function" {
		t.Fatalf("expected openai function tool, got %#v", tools[0]["type"])
	}
	if tools[0]["strict"] != false {
		t.Fatalf("expected strict=false, got %#v", tools[0]["strict"])
	}
	parameters := tools[0]["parameters"].(map[string]any)
	properties := parameters["properties"].(map[string]any)
	shell := properties["shell"].(map[string]any)
	enumValues := shell["enum"].([]any)
	if len(enumValues) != 4 {
		t.Fatalf("expected shell enum values, got %#v", shell["enum"])
	}
}

func TestExportSchemaSpeechSynthesizeDefaultsToMiniMaxSpeechModel(t *testing.T) {
	root := NewRootCmd("0.test")

	stdout, stderr, err := executeRootRaw(root, []string{
		"export-schema",
		"--command", "speech synthesize",
		"--format", "openai",
	})
	if err != nil {
		t.Fatalf("export-schema failed: %v stderr=%s", err, stderr)
	}

	var tools []map[string]any
	if err := json.Unmarshal([]byte(stdout), &tools); err != nil {
		t.Fatalf("unmarshal export-schema output: %v output=%q", err, stdout)
	}
	parameters := tools[0]["parameters"].(map[string]any)
	properties := parameters["properties"].(map[string]any)
	model := properties["model"].(map[string]any)
	if model["default"] != defaultMiniMaxSpeechModelID {
		t.Fatalf("expected default MiniMax speech model, got %#v", model["default"])
	}
	if properties["pronunciation"] == nil || properties["subtitles"] == nil {
		t.Fatalf("expected speech-specific flags, got %#v", properties)
	}
}

func TestExportSchemaOmitsHiddenAndDeprecatedFlags(t *testing.T) {
	root := NewRootCmd("0.test")

	stdout, stderr, err := executeRootRaw(root, []string{
		"export-schema",
		"--command", "auth login",
		"--format", "generic",
	})
	if err != nil {
		t.Fatalf("export-schema failed: %v stderr=%s", err, stderr)
	}

	var tools []map[string]any
	if err := json.Unmarshal([]byte(stdout), &tools); err != nil {
		t.Fatalf("unmarshal export-schema output: %v output=%q", err, stdout)
	}
	properties := tools[0]["input_schema"].(map[string]any)["properties"].(map[string]any)
	if properties["token"] != nil {
		t.Fatalf("expected hidden token flag to be omitted, got %#v", properties["token"])
	}

	stdout, stderr, err = executeRootRaw(root, []string{
		"export-schema",
		"--command", "models route-override set",
		"--format", "generic",
	})
	if err != nil {
		t.Fatalf("export-schema failed: %v stderr=%s", err, stderr)
	}
	tools = nil
	if err := json.Unmarshal([]byte(stdout), &tools); err != nil {
		t.Fatalf("unmarshal export-schema output: %v output=%q", err, stdout)
	}
	properties = tools[0]["input_schema"].(map[string]any)["properties"].(map[string]any)
	if properties["skill_type"] != nil {
		t.Fatalf("expected deprecated skill_type flag to be omitted, got %#v", properties["skill_type"])
	}
	if properties["route"] == nil || properties["project"] == nil || properties["model"] == nil {
		t.Fatalf("expected current route/project/model flags, got %#v", properties)
	}
}

func TestExportSchemaUnknownCommandFails(t *testing.T) {
	root := NewRootCmd("0.test")

	_, _, err := executeRootRaw(root, []string{
		"export-schema",
		"--command", "does not exist",
	})
	if err == nil {
		t.Fatal("expected export-schema to fail for unknown command")
	}
}

func TestExportSchemaMusicGenerateIncludesPromptLyricsOptions(t *testing.T) {
	root := NewRootCmd("0.test")

	stdout, stderr, err := executeRootRaw(root, []string{
		"export-schema",
		"--command", "music generate",
		"--format", "openai",
	})
	if err != nil {
		t.Fatalf("export-schema failed: %v stderr=%s", err, stderr)
	}

	var tools []map[string]any
	if err := json.Unmarshal([]byte(stdout), &tools); err != nil {
		t.Fatalf("unmarshal export-schema output: %v output=%q", err, stdout)
	}
	parameters := tools[0]["parameters"].(map[string]any)
	if parameters["oneOf"] == nil {
		t.Fatalf("expected oneOf requirement, got %#v", parameters)
	}
	properties := parameters["properties"].(map[string]any)
	if properties["lyrics_optimizer"] == nil || properties["instrumental"] == nil {
		t.Fatalf("expected music-specific flags, got %#v", properties)
	}
	model := properties["model"].(map[string]any)
	if model["default"] != defaultMiniMaxMusicModelID {
		t.Fatalf("expected default MiniMax music model, got %#v", model["default"])
	}
}
