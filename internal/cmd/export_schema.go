package cmd

import (
	"encoding/json"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/wtgoku-create/popiartcli/internal/output"
)

const (
	exportSchemaFormatAnthropic = "anthropic"
	exportSchemaFormatOpenAI    = "openai"
	exportSchemaFormatGeneric   = "generic"
)

var useArgPattern = regexp.MustCompile(`<([^>]+)>`)
var enumTokenPattern = regexp.MustCompile(`[A-Za-z0-9._-]+`)

type genericToolSchema struct {
	Name        string         `json:"name"`
	CommandPath string         `json:"command_path"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

type anthropicToolSchema struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

type openAIToolSchema struct {
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters"`
	Strict      bool           `json:"strict"`
}

func newExportSchemaCmd() *cobra.Command {
	var format string
	var commandPath string

	exportCmd := &cobra.Command{
		Use:   "export-schema",
		Short: "导出 PopiArt CLI 命令的 tool JSON schema",
		Long: "从 Cobra 命令树导出 PopiArt CLI 自身的命令结构，用于动态注册为 Anthropic / OpenAI 兼容工具。\n\n" +
			"默认导出所有可执行 leaf 命令；传入 `--command \"video generate\"` 可只导出一个命令。\n" +
			"该命令直接输出原始 JSON schema，而不是标准 `{ ok, data }` envelope。",
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := normalizeExportSchemaFormat(format)
			if err != nil {
				return err
			}

			payload, err := exportCLISchemas(cmd.Root(), commandPath, format)
			if err != nil {
				return err
			}
			return writeRawJSON(cmd.OutOrStdout(), payload)
		},
	}

	exportCmd.Flags().StringVar(&format, "format", exportSchemaFormatAnthropic, "导出格式: anthropic | openai | generic")
	exportCmd.Flags().StringVar(&commandPath, "command", "", "仅导出指定命令，例如 \"video generate\" 或 \"models route-override set\"")
	return exportCmd
}

func normalizeExportSchemaFormat(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", exportSchemaFormatAnthropic:
		return exportSchemaFormatAnthropic, nil
	case exportSchemaFormatOpenAI:
		return exportSchemaFormatOpenAI, nil
	case exportSchemaFormatGeneric:
		return exportSchemaFormatGeneric, nil
	default:
		return "", output.NewError("VALIDATION_ERROR", "不支持的 export-schema 格式", map[string]any{
			"format":    raw,
			"supported": []string{exportSchemaFormatAnthropic, exportSchemaFormatOpenAI, exportSchemaFormatGeneric},
		})
	}
}

func exportCLISchemas(root *cobra.Command, requestedPath, format string) (any, error) {
	commands := exportableLeafCommands(root)
	if requestedPath != "" {
		command, ok := findExportableCommand(commands, requestedPath)
		if !ok {
			paths := make([]string, 0, len(commands))
			for _, item := range commands {
				paths = append(paths, exportCommandPath(item))
			}
			sort.Strings(paths)
			return nil, output.NewError("NOT_FOUND", "未找到要导出的命令", map[string]any{
				"command":   requestedPath,
				"supported": paths,
			})
		}
		commands = []*cobra.Command{command}
	}

	generic := make([]genericToolSchema, 0, len(commands))
	for _, command := range commands {
		generic = append(generic, buildGenericToolSchema(command))
	}

	switch format {
	case exportSchemaFormatGeneric:
		return generic, nil
	case exportSchemaFormatOpenAI:
		items := make([]openAIToolSchema, 0, len(generic))
		for _, item := range generic {
			items = append(items, openAIToolSchema{
				Type:        "function",
				Name:        item.Name,
				Description: item.Description,
				Parameters:  item.InputSchema,
				// OpenAI docs recommend explicit strict mode configuration. We keep false
				// so optional CLI flags remain optional when these schemas are reused.
				Strict: false,
			})
		}
		return items, nil
	default:
		items := make([]anthropicToolSchema, 0, len(generic))
		for _, item := range generic {
			items = append(items, anthropicToolSchema{
				Name:        item.Name,
				Description: item.Description,
				InputSchema: item.InputSchema,
			})
		}
		return items, nil
	}
}

func exportableLeafCommands(root *cobra.Command) []*cobra.Command {
	var out []*cobra.Command
	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		if cmd == nil || !cmd.IsAvailableCommand() || cmd.Hidden {
			return
		}
		if cmd.Runnable() && shouldExportCommand(cmd) {
			out = append(out, cmd)
		}
		for _, child := range cmd.Commands() {
			walk(child)
		}
	}
	walk(root)

	sort.Slice(out, func(i, j int) bool {
		return exportCommandPath(out[i]) < exportCommandPath(out[j])
	})
	return out
}

func shouldExportCommand(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	switch cmd.Name() {
	case "help", "export-schema":
		return false
	default:
		return true
	}
}

func findExportableCommand(commands []*cobra.Command, requested string) (*cobra.Command, bool) {
	normalized := normalizeRequestedCommandPath(requested)
	for _, command := range commands {
		if normalizeRequestedCommandPath(exportCommandPath(command)) == normalized {
			return command, true
		}
		if normalizeRequestedCommandPath(command.CommandPath()) == normalized {
			return command, true
		}
	}
	return nil, false
}

func normalizeRequestedCommandPath(path string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(path)), " ")
}

func exportCommandPath(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}
	path := strings.TrimSpace(cmd.CommandPath())
	rootName := strings.TrimSpace(cmd.Root().Name())
	path = strings.TrimSpace(strings.TrimPrefix(path, rootName))
	return strings.TrimSpace(path)
}

func buildGenericToolSchema(cmd *cobra.Command) genericToolSchema {
	return genericToolSchema{
		Name:        exportToolName(exportCommandPath(cmd)),
		CommandPath: exportCommandPath(cmd),
		Description: exportToolDescription(cmd),
		InputSchema: exportInputSchema(cmd),
	}
}

func exportToolName(commandPath string) string {
	name := "popiart_" + strings.ReplaceAll(commandPath, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")
	name = regexp.MustCompile(`[^A-Za-z0-9_]+`).ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")
	return name
}

func exportToolDescription(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}
	description := strings.TrimSpace(cmd.Short)
	if description == "" {
		description = strings.TrimSpace(cmd.Long)
	}
	if description == "" {
		description = "PopiArt CLI command: " + cmd.CommandPath()
	}
	return description
}

func exportInputSchema(cmd *cobra.Command) map[string]any {
	properties := map[string]any{}
	required := []string{}
	seen := map[string]bool{}

	addProperty := func(name string, schema map[string]any, isRequired bool) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		properties[name] = schema
		if isRequired {
			required = append(required, name)
		}
	}

	for _, arg := range positionalArgsFromUse(cmd.Use) {
		schema := map[string]any{
			"type":        "string",
			"description": "Positional argument for `popiart " + exportCommandPath(cmd) + "`.",
		}
		if len(arg.Enum) > 0 {
			schema["enum"] = arg.Enum
		}
		addProperty(arg.Name, schema, true)
	}

	collectFlagSchemas(cmd.InheritedFlags(), addProperty)
	collectFlagSchemas(cmd.Flags(), addProperty)

	out := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		sort.Strings(required)
		out["required"] = required
	}
	return applyExportSchemaOverlay(exportCommandPath(cmd), out)
}

func applyExportSchemaOverlay(commandPath string, schema map[string]any) map[string]any {
	requiredSet := map[string]bool{}
	if raw, ok := schema["required"].([]string); ok {
		for _, item := range raw {
			requiredSet[item] = true
		}
	} else if raw, ok := schema["required"].([]any); ok {
		for _, item := range raw {
			if text, ok := item.(string); ok {
				requiredSet[text] = true
			}
		}
	}

	addRequired := func(items ...string) {
		for _, item := range items {
			if item != "" {
				requiredSet[item] = true
			}
		}
	}

	switch commandPath {
	case "image", "image generate":
		addRequired("prompt")
	case "image img2img", "image transform":
		addRequired("prompt")
		schema["oneOf"] = []map[string]any{
			{"required": []string{"image"}},
			{"required": []string{"source_artifact_id"}},
		}
	case "image describe":
		schema["oneOf"] = []map[string]any{
			{"required": []string{"image"}},
			{"required": []string{"from"}},
			{"required": []string{"source_artifact_id"}},
		}
	case "video", "video generate", "video img2video", "video from-image":
		schema["oneOf"] = []map[string]any{
			{"required": []string{"image"}},
			{"required": []string{"from"}},
			{"required": []string{"source_artifact_id"}},
		}
	case "video action-transfer":
		addRequired("image")
		addRequired("video")
	case "audio tts", "speech synthesize":
		schema["oneOf"] = []map[string]any{
			{"required": []string{"text"}},
			{"required": []string{"text_file"}},
		}
	case "music", "music generate":
		schema["oneOf"] = []map[string]any{
			{"required": []string{"prompt"}},
			{"required": []string{"lyrics"}},
			{"required": []string{"lyrics_file"}},
		}
	}

	if len(requiredSet) == 0 {
		delete(schema, "required")
		return schema
	}

	required := make([]string, 0, len(requiredSet))
	for item := range requiredSet {
		required = append(required, item)
	}
	sort.Strings(required)
	schema["required"] = required
	return schema
}

func collectFlagSchemas(flags *pflag.FlagSet, addProperty func(string, map[string]any, bool)) {
	if flags == nil {
		return
	}
	flags.VisitAll(func(flag *pflag.Flag) {
		if flag == nil || flag.Hidden || flag.Deprecated != "" || shouldSkipExportedFlag(flag.Name) {
			return
		}
		name := strings.ReplaceAll(flag.Name, "-", "_")
		schema := flagJSONSchema(flag)
		required := false
		if _, ok := flag.Annotations[cobra.BashCompOneRequiredFlag]; ok {
			required = true
		}
		addProperty(name, schema, required)
	})
}

func shouldSkipExportedFlag(name string) bool {
	switch strings.TrimSpace(name) {
	case "", "help", "output", "plain", "non-interactive", "quiet", "yes", "dry-run", "async", "no-color":
		return true
	default:
		return false
	}
}

func flagJSONSchema(flag *pflag.Flag) map[string]any {
	schema := map[string]any{
		"description": strings.TrimSpace(flag.Usage),
	}

	switch strings.ToLower(flag.Value.Type()) {
	case "bool":
		schema["type"] = "boolean"
		if value, err := strconv.ParseBool(flag.DefValue); err == nil {
			schema["default"] = value
		}
	case "float64":
		schema["type"] = "number"
		if value, err := strconv.ParseFloat(flag.DefValue, 64); err == nil && value != 0 {
			schema["default"] = value
		}
	case "int", "int32", "int64":
		schema["type"] = "integer"
		if value, err := strconv.Atoi(flag.DefValue); err == nil && value != 0 {
			schema["default"] = value
		}
	case "stringarray", "stringslice":
		schema["type"] = "array"
		items := map[string]any{"type": "string"}
		if enum := enumFromUsage(flag.Usage); len(enum) > 0 {
			items["enum"] = enum
		}
		schema["items"] = items
	default:
		schema["type"] = "string"
		if flag.DefValue != "" {
			schema["default"] = flag.DefValue
		}
		if enum := enumFromUsage(flag.Usage); len(enum) > 0 {
			schema["enum"] = enum
		}
	}

	return schema
}

type positionalArgSpec struct {
	Name string
	Enum []string
}

func positionalArgsFromUse(use string) []positionalArgSpec {
	matches := useArgPattern.FindAllStringSubmatch(use, -1)
	out := make([]positionalArgSpec, 0, len(matches))
	for idx, match := range matches {
		if len(match) < 2 {
			continue
		}
		token := strings.TrimSpace(match[1])
		enum := enumFromToken(token)
		name := positionalNameFromToken(token, idx)
		out = append(out, positionalArgSpec{
			Name: name,
			Enum: enum,
		})
	}
	return out
}

func positionalNameFromToken(token string, index int) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return "arg_" + strconv.Itoa(index+1)
	}
	if strings.Contains(token, "|") {
		enum := enumFromToken(token)
		if isShellEnum(enum) {
			return "shell"
		}
		return "arg_" + strconv.Itoa(index+1)
	}
	return strings.ReplaceAll(token, "-", "_")
}

func enumFromToken(token string) []string {
	if !strings.Contains(token, "|") {
		return nil
	}
	parts := strings.Split(token, "|")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		values = append(values, part)
	}
	if len(values) < 2 {
		return nil
	}
	return values
}

func isShellEnum(values []string) bool {
	if len(values) == 0 {
		return false
	}
	set := map[string]bool{}
	for _, value := range values {
		set[strings.ToLower(strings.TrimSpace(value))] = true
	}
	return set["bash"] || set["zsh"] || set["fish"] || set["powershell"]
}

func enumFromUsage(usage string) []string {
	if !strings.Contains(usage, "|") {
		return nil
	}
	candidates := enumTokenPattern.FindAllString(usage, -1)
	if len(candidates) < 2 {
		return nil
	}
	values := make([]string, 0, len(candidates))
	seen := map[string]bool{}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if strings.EqualFold(candidate, "例如") || strings.EqualFold(candidate, "default") {
			continue
		}
		key := strings.ToLower(candidate)
		if seen[key] {
			continue
		}
		seen[key] = true
		values = append(values, candidate)
	}
	if len(values) < 2 {
		return nil
	}
	return values
}

func writeRawJSON(w io.Writer, payload any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}
