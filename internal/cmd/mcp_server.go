package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/wtgoku-create/popiartcli/internal/config"
	"github.com/wtgoku-create/popiartcli/internal/output"
	"github.com/wtgoku-create/popiartcli/internal/seed"
	"github.com/wtgoku-create/popiartcli/internal/types"
)

const mcpProtocolVersion = "2025-03-26"

type mcpToolDefinition struct {
	Name         string
	Title        string
	Description  string
	InputSchema  map[string]any
	OutputSchema map[string]any
	Annotations  map[string]any
	Handler      func(context.Context, map[string]any) (any, error)
}

type mcpJSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpJSONRPCResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      any              `json:"id"`
	Result  any              `json:"result,omitempty"`
	Error   *mcpJSONRPCError `json:"error,omitempty"`
}

type mcpJSONRPCError struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data,omitempty"`
}

type mcpToolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type mcpToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type mcpToolResult struct {
	Content           []mcpToolContent `json:"content"`
	StructuredContent map[string]any   `json:"structuredContent,omitempty"`
	IsError           bool             `json:"isError,omitempty"`
}

type mcpServer struct {
	version     string
	reader      *bufio.Reader
	writer      *bufio.Writer
	logWriter   io.Writer
	frameMode   string
	initialized bool
	toolsByName map[string]mcpToolDefinition
	toolList    []mcpToolDefinition
}

func runMCPServer(stdin io.Reader, stdout io.Writer, stderr io.Writer, version string) error {
	server := &mcpServer{
		version:     version,
		reader:      bufio.NewReader(stdin),
		writer:      bufio.NewWriter(stdout),
		logWriter:   stderr,
		toolsByName: map[string]mcpToolDefinition{},
	}
	server.toolList = mcpToolDefinitions()
	for _, tool := range server.toolList {
		server.toolsByName[tool.Name] = tool
	}
	return server.serve()
}

func (s *mcpServer) serve() error {
	for {
		payload, err := s.readFrame()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if len(bytes.TrimSpace(payload)) == 0 {
			continue
		}

		var responses []mcpJSONRPCResponse
		trimmed := bytes.TrimSpace(payload)
		if len(trimmed) > 0 && trimmed[0] == '[' {
			var batch []json.RawMessage
			if err := json.Unmarshal(trimmed, &batch); err != nil {
				if writeErr := s.writeMessage(mcpJSONRPCResponse{
					JSONRPC: "2.0",
					ID:      nil,
					Error:   newMCPProtocolError(-32700, "Parse error", map[string]any{"details": err.Error()}),
				}); writeErr != nil {
					return writeErr
				}
				continue
			}
			if len(batch) == 0 {
				if writeErr := s.writeMessage(mcpJSONRPCResponse{
					JSONRPC: "2.0",
					ID:      nil,
					Error:   newMCPProtocolError(-32600, "Invalid Request", nil),
				}); writeErr != nil {
					return writeErr
				}
				continue
			}
			for _, item := range batch {
				response := s.handleMessage(item)
				if response != nil {
					responses = append(responses, *response)
				}
			}
			if len(responses) > 0 {
				if err := s.writeMessage(responses); err != nil {
					return err
				}
			}
			continue
		}

		response := s.handleMessage(trimmed)
		if response == nil {
			continue
		}
		if err := s.writeMessage(response); err != nil {
			return err
		}
	}
}

func (s *mcpServer) handleMessage(raw json.RawMessage) *mcpJSONRPCResponse {
	var request mcpJSONRPCRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		return &mcpJSONRPCResponse{
			JSONRPC: "2.0",
			ID:      nil,
			Error:   newMCPProtocolError(-32700, "Parse error", map[string]any{"details": err.Error()}),
		}
	}
	if request.JSONRPC != "2.0" || request.Method == "" {
		return &mcpJSONRPCResponse{
			JSONRPC: "2.0",
			ID:      parseMCPID(request.ID),
			Error:   newMCPProtocolError(-32600, "Invalid Request", nil),
		}
	}

	isNotification := len(request.ID) == 0
	switch request.Method {
	case "notifications/initialized":
		s.initialized = true
		return nil
	case "notifications/cancelled":
		return nil
	case "initialize":
		s.initialized = true
		return &mcpJSONRPCResponse{
			JSONRPC: "2.0",
			ID:      parseMCPID(request.ID),
			Result: map[string]any{
				"protocolVersion": mcpProtocolVersion,
				"capabilities": map[string]any{
					"tools": map[string]any{
						"listChanged": false,
					},
				},
				"serverInfo": map[string]any{
					"name":    popiartMCPServerName,
					"version": defaultString(s.version, "dev"),
				},
				"instructions": "Use PopiArt for creator skill discovery, runtime skill execution, jobs, and artifacts.",
			},
		}
	case "ping":
		if isNotification {
			return nil
		}
		return &mcpJSONRPCResponse{
			JSONRPC: "2.0",
			ID:      parseMCPID(request.ID),
			Result:  map[string]any{},
		}
	case "tools/list":
		if isNotification {
			return nil
		}
		return &mcpJSONRPCResponse{
			JSONRPC: "2.0",
			ID:      parseMCPID(request.ID),
			Result: map[string]any{
				"tools": s.protocolTools(),
			},
		}
	case "tools/call":
		if isNotification {
			return nil
		}
		var params mcpToolCallParams
		if len(request.Params) > 0 {
			if err := json.Unmarshal(request.Params, &params); err != nil {
				return &mcpJSONRPCResponse{
					JSONRPC: "2.0",
					ID:      parseMCPID(request.ID),
					Error:   newMCPProtocolError(-32602, "Invalid params", map[string]any{"details": err.Error()}),
				}
			}
		}
		if params.Name == "" {
			return &mcpJSONRPCResponse{
				JSONRPC: "2.0",
				ID:      parseMCPID(request.ID),
				Error:   newMCPProtocolError(-32602, "Invalid params", map[string]any{"hint": "`name` is required"}),
			}
		}
		tool, ok := s.toolsByName[params.Name]
		if !ok {
			return &mcpJSONRPCResponse{
				JSONRPC: "2.0",
				ID:      parseMCPID(request.ID),
				Error:   newMCPProtocolError(-32602, "Unknown tool", map[string]any{"name": params.Name}),
			}
		}
		if params.Arguments == nil {
			params.Arguments = map[string]any{}
		}
		data, err := tool.Handler(context.Background(), params.Arguments)
		if err != nil {
			return &mcpJSONRPCResponse{
				JSONRPC: "2.0",
				ID:      parseMCPID(request.ID),
				Result:  buildMCPErrorToolResult(err),
			}
		}
		return &mcpJSONRPCResponse{
			JSONRPC: "2.0",
			ID:      parseMCPID(request.ID),
			Result:  buildMCPSuccessToolResult(data),
		}
	default:
		if isNotification {
			return nil
		}
		return &mcpJSONRPCResponse{
			JSONRPC: "2.0",
			ID:      parseMCPID(request.ID),
			Error:   newMCPProtocolError(-32601, "Method not found", map[string]any{"method": request.Method}),
		}
	}
}

func (s *mcpServer) protocolTools() []map[string]any {
	tools := make([]map[string]any, 0, len(s.toolList))
	for _, def := range s.toolList {
		tool := map[string]any{
			"name":        def.Name,
			"description": def.Description,
			"inputSchema": def.InputSchema,
		}
		if def.Title != "" {
			tool["title"] = def.Title
		}
		if def.OutputSchema != nil {
			tool["outputSchema"] = def.OutputSchema
		}
		if def.Annotations != nil {
			tool["annotations"] = def.Annotations
		}
		tools = append(tools, tool)
	}
	return tools
}

func mcpToolDefinitions() []mcpToolDefinition {
	return []mcpToolDefinition{
		{
			Name:        "list_skills",
			Title:       "List Skills",
			Description: "List discoverable PopiArt skills.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"tag":    map[string]any{"type": "string"},
					"search": map[string]any{"type": "string"},
					"limit":  map[string]any{"type": "integer", "minimum": 0},
					"offset": map[string]any{"type": "integer", "minimum": 0},
				},
			},
			OutputSchema: map[string]any{"type": "object"},
			Annotations:  readOnlyToolAnnotations(),
			Handler:      listSkillsTool,
		},
		{
			Name:        "get_skill",
			Title:       "Get Skill",
			Description: "Get the full PopiArt skill document.",
			InputSchema: objectSchemaWithRequired(map[string]any{
				"skill_id": map[string]any{"type": "string"},
			}, "skill_id"),
			OutputSchema: map[string]any{"type": "object"},
			Annotations:  readOnlyToolAnnotations(),
			Handler:      getSkillTool,
		},
		{
			Name:        "get_skill_schema",
			Title:       "Get Skill Schema",
			Description: "Get the input and output schema for a PopiArt skill.",
			InputSchema: objectSchemaWithRequired(map[string]any{
				"skill_id": map[string]any{"type": "string"},
			}, "skill_id"),
			OutputSchema: map[string]any{"type": "object"},
			Annotations:  readOnlyToolAnnotations(),
			Handler:      getSkillSchemaTool,
		},
		{
			Name:        "run_skill",
			Title:       "Run Skill",
			Description: "Submit a PopiArt skill run and return a job id.",
			InputSchema: objectSchemaWithRequired(map[string]any{
				"skill_id":        map[string]any{"type": "string"},
				"input":           map[string]any{"type": "object"},
				"priority":        map[string]any{"type": "string"},
				"idempotency_key": map[string]any{"type": "string"},
				"project_id":      map[string]any{"type": "string"},
			}, "skill_id", "input"),
			OutputSchema: map[string]any{"type": "object"},
			Handler:      runSkillTool,
		},
		{
			Name:        "get_job",
			Title:       "Get Job",
			Description: "Fetch the current state of a PopiArt job.",
			InputSchema: objectSchemaWithRequired(map[string]any{
				"job_id": map[string]any{"type": "string"},
			}, "job_id"),
			OutputSchema: map[string]any{"type": "object"},
			Annotations:  readOnlyToolAnnotations(),
			Handler:      getJobTool,
		},
		{
			Name:        "wait_job",
			Title:       "Wait Job",
			Description: "Wait for a PopiArt job to reach a terminal state.",
			InputSchema: objectSchemaWithRequired(map[string]any{
				"job_id":          map[string]any{"type": "string"},
				"interval_millis": map[string]any{"type": "integer", "minimum": 1},
			}, "job_id"),
			OutputSchema: map[string]any{"type": "object"},
			Handler:      waitJobTool,
		},
		{
			Name:        "get_job_logs",
			Title:       "Get Job Logs",
			Description: "Read logs for a PopiArt job.",
			InputSchema: objectSchemaWithRequired(map[string]any{
				"job_id": map[string]any{"type": "string"},
			}, "job_id"),
			OutputSchema: map[string]any{"type": "object"},
			Annotations:  readOnlyToolAnnotations(),
			Handler:      getJobLogsTool,
		},
		{
			Name:        "list_artifacts",
			Title:       "List Artifacts",
			Description: "List artifacts created by a PopiArt job.",
			InputSchema: objectSchemaWithRequired(map[string]any{
				"job_id": map[string]any{"type": "string"},
			}, "job_id"),
			OutputSchema: map[string]any{"type": "object"},
			Annotations:  readOnlyToolAnnotations(),
			Handler:      listArtifactsTool,
		},
		{
			Name:        "pull_artifact",
			Title:       "Pull Artifact",
			Description: "Download a PopiArt artifact to a local file path.",
			InputSchema: objectSchemaWithRequired(map[string]any{
				"artifact_id": map[string]any{"type": "string"},
				"out":         map[string]any{"type": "string"},
			}, "artifact_id"),
			OutputSchema: map[string]any{"type": "object"},
			Handler:      pullArtifactTool,
		},
		{
			Name:         "whoami",
			Title:        "Who Am I",
			Description:  "Read the authenticated PopiArt user.",
			InputSchema:  map[string]any{"type": "object", "properties": map[string]any{}},
			OutputSchema: map[string]any{"type": "object"},
			Annotations:  readOnlyToolAnnotations(),
			Handler:      whoamiTool,
		},
		{
			Name:         "current_project",
			Title:        "Current Project",
			Description:  "Read the currently configured PopiArt project and local config path.",
			InputSchema:  map[string]any{"type": "object", "properties": map[string]any{}},
			OutputSchema: map[string]any{"type": "object"},
			Annotations:  readOnlyToolAnnotations(),
			Handler:      currentProjectTool,
		},
	}
}

func listSkillsTool(ctx context.Context, args map[string]any) (any, error) {
	tag := optionalStringArg(args, "tag")
	search := optionalStringArg(args, "search")
	limit, err := optionalIntArg(args, "limit", 50)
	if err != nil {
		return nil, err
	}
	offset, err := optionalIntArg(args, "offset", 0)
	if err != nil {
		return nil, err
	}

	localItems := seed.MatchingBundledSkillSummaries(tag, search)
	var resp types.SkillListResponse
	if err := currentClient().GetJSON(ctx, "/skills", map[string]string{
		"tag":    tag,
		"search": search,
		"limit":  strconv.Itoa(remotePageSize(limit, offset)),
		"offset": "0",
	}, &resp); err != nil {
		return nil, err
	}
	localItems, err = bundledSkillSummariesMissingOnRemote(ctx, localItems)
	if err != nil {
		return nil, err
	}
	merged := mergeSkillSummaries(resp.Items, localItems)
	resp.Items = paginateSkillSummaries(merged, limit, offset)
	resp.Total += len(localItems)
	resp.Limit = limit
	resp.Offset = offset
	return resp, nil
}

func getSkillTool(ctx context.Context, args map[string]any) (any, error) {
	skillID, err := requiredStringArg(args, "skill_id")
	if err != nil {
		return nil, err
	}
	var skill types.Skill
	if err := currentClient().GetJSON(ctx, "/skills/"+skillID, nil, &skill); err != nil {
		if cliErr, ok := err.(*output.CLIError); ok && cliErr.Code == "NOT_FOUND" {
			if skill, ok := seed.FindBundledSkill(skillID); ok {
				return skill, nil
			}
		}
		return nil, err
	}
	return skill, nil
}

func getSkillSchemaTool(ctx context.Context, args map[string]any) (any, error) {
	skillID, err := requiredStringArg(args, "skill_id")
	if err != nil {
		return nil, err
	}
	var schema types.SkillSchemaResponse
	if err := currentClient().GetJSON(ctx, "/skills/"+skillID+"/schema", nil, &schema); err != nil {
		if cliErr, ok := err.(*output.CLIError); ok && cliErr.Code == "NOT_FOUND" {
			if schema, ok := seed.FindBundledSkillSchema(skillID); ok {
				return schema, nil
			}
		}
		return nil, err
	}
	return schema, nil
}

func runSkillTool(ctx context.Context, args map[string]any) (any, error) {
	skillID, err := requiredStringArg(args, "skill_id")
	if err != nil {
		return nil, err
	}
	payload, err := requiredMapArg(args, "input")
	if err != nil {
		return nil, err
	}
	if err := validateBundledSkillRun(skillID); err != nil {
		return nil, err
	}

	cfg := config.Load()
	body := map[string]any{
		"skill_id": skillID,
		"input":    payload,
		"priority": defaultString(optionalStringArg(args, "priority"), "normal"),
	}
	if cfg.Project != "" {
		body["project_id"] = cfg.Project
	}
	if projectID := optionalStringArg(args, "project_id"); projectID != "" {
		body["project_id"] = projectID
	}
	if idempotencyKey := optionalStringArg(args, "idempotency_key"); idempotencyKey != "" {
		body["idempotency_key"] = idempotencyKey
	}

	var job types.Job
	if err := currentClient().PostJSON(ctx, "/jobs", body, &job); err != nil {
		return nil, err
	}
	return job, nil
}

func getJobTool(ctx context.Context, args map[string]any) (any, error) {
	jobID, err := requiredStringArg(args, "job_id")
	if err != nil {
		return nil, err
	}
	var job types.Job
	if err := currentClient().GetJSON(ctx, "/jobs/"+jobID, nil, &job); err != nil {
		return nil, err
	}
	return job, nil
}

func waitJobTool(ctx context.Context, args map[string]any) (any, error) {
	jobID, err := requiredStringArg(args, "job_id")
	if err != nil {
		return nil, err
	}
	intervalMillis, err := optionalIntArg(args, "interval_millis", 2000)
	if err != nil {
		return nil, err
	}
	if intervalMillis <= 0 {
		return nil, output.NewError("VALIDATION_ERROR", "interval_millis 必须大于 0", map[string]any{
			"interval_millis": intervalMillis,
		})
	}
	job, err := waitForJobQuiet(ctx, currentClient(), jobID, time.Duration(intervalMillis)*time.Millisecond, 300)
	if err != nil {
		return nil, err
	}
	return job, nil
}

func getJobLogsTool(ctx context.Context, args map[string]any) (any, error) {
	jobID, err := requiredStringArg(args, "job_id")
	if err != nil {
		return nil, err
	}
	var logs []types.LogEntry
	if err := currentClient().GetJSON(ctx, "/jobs/"+jobID+"/logs", nil, &logs); err != nil {
		return nil, err
	}
	return map[string]any{
		"job_id": jobID,
		"logs":   logs,
	}, nil
}

func listArtifactsTool(ctx context.Context, args map[string]any) (any, error) {
	jobID, err := requiredStringArg(args, "job_id")
	if err != nil {
		return nil, err
	}
	var artifacts types.ArtifactListResponse
	if err := currentClient().GetJSON(ctx, "/jobs/"+jobID+"/artifacts", nil, &artifacts); err != nil {
		return nil, err
	}
	return map[string]any{
		"job_id": jobID,
		"items":  artifacts.Items,
	}, nil
}

func pullArtifactTool(ctx context.Context, args map[string]any) (any, error) {
	artifactID, err := requiredStringArg(args, "artifact_id")
	if err != nil {
		return nil, err
	}
	var meta types.Artifact
	if err := currentClient().GetJSON(ctx, "/artifacts/"+artifactID, nil, &meta); err != nil {
		return nil, err
	}

	res, err := currentClient().Stream(ctx, "GET", "/artifacts/"+artifactID+"/content", apiOpts("", "application/octet-stream"))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	outPath := optionalStringArg(args, "out")
	if outPath == "" {
		filename := meta.Filename
		if filename == "" {
			filename = "artifact-" + artifactID
		}
		outPath = filepath.Join(config.Dir(), "downloads", artifactID, filename)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return nil, output.NewError("CLI_ERROR", "创建输出目录失败", map[string]any{"details": err.Error()})
	}
	file, err := os.Create(outPath)
	if err != nil {
		return nil, output.NewError("CLI_ERROR", "创建输出文件失败", map[string]any{"details": err.Error()})
	}
	defer file.Close()

	n, err := io.Copy(file, res.Body)
	if err != nil {
		return nil, output.NewError("NETWORK_ERROR", "写入工件失败", map[string]any{"details": err.Error()})
	}

	return map[string]any{
		"artifact_id":  artifactID,
		"saved_to":     outPath,
		"bytes":        n,
		"content_type": meta.ContentType,
		"filename":     meta.Filename,
	}, nil
}

func whoamiTool(ctx context.Context, args map[string]any) (any, error) {
	var me types.AuthSession
	if err := currentClient().GetJSON(ctx, "/auth/me", nil, &me); err != nil {
		return nil, err
	}
	if me.User == nil && me.ID != "" {
		return types.User{
			ID:     me.ID,
			Email:  me.Email,
			Name:   me.Name,
			Scopes: me.Scopes,
		}, nil
	}
	return me, nil
}

func currentProjectTool(ctx context.Context, args map[string]any) (any, error) {
	cfg := config.Load()
	return map[string]any{
		"project":                 cfg.Project,
		"endpoint":                cfg.Endpoint,
		"config_path":             config.Path(),
		"config_dir":              config.Dir(),
		"has_key":                 cfg.Token != "",
		"server_name":             popiartMCPServerName,
		"server_id":               popiartMCPServerID,
		"official_runtime_skills": officialRuntimeSkills(),
	}, nil
}

func buildMCPSuccessToolResult(data any) mcpToolResult {
	structured := map[string]any{"data": data}
	return mcpToolResult{
		Content: []mcpToolContent{
			{Type: "text", Text: mustMarshalJSON(structured)},
		},
		StructuredContent: structured,
	}
}

func buildMCPErrorToolResult(err error) mcpToolResult {
	cliErr, ok := err.(*output.CLIError)
	payload := map[string]any{
		"code":    "FATAL",
		"message": err.Error(),
	}
	if ok {
		payload["code"] = cliErr.Code
		payload["message"] = cliErr.Message
		if len(cliErr.Details) > 0 {
			payload["details"] = cliErr.Details
		}
	}
	structured := map[string]any{"error": payload}
	return mcpToolResult{
		Content: []mcpToolContent{
			{Type: "text", Text: mustMarshalJSON(structured)},
		},
		StructuredContent: structured,
		IsError:           true,
	}
}

func (s *mcpServer) readFrame() ([]byte, error) {
	switch s.frameMode {
	case "header":
		return s.readHeaderFrame("")
	case "newline":
		return s.readNewlineFrame()
	default:
		return s.readAutoFrame()
	}
}

func (s *mcpServer) readAutoFrame() ([]byte, error) {
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil && len(line) == 0 {
			return nil, err
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if err != nil {
				return nil, err
			}
			continue
		}
		if strings.HasPrefix(strings.ToLower(trimmed), "content-length:") {
			s.frameMode = "header"
			return s.readHeaderFrame(trimmed)
		}
		s.frameMode = "newline"
		return []byte(trimmed), nil
	}
}

func (s *mcpServer) readNewlineFrame() ([]byte, error) {
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil && len(line) == 0 {
			return nil, err
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if err != nil {
				return nil, err
			}
			continue
		}
		return []byte(trimmed), nil
	}
}

func (s *mcpServer) readHeaderFrame(firstHeader string) ([]byte, error) {
	contentLength := -1
	headerLine := firstHeader
	for {
		if headerLine == "" {
			line, err := s.reader.ReadString('\n')
			if err != nil {
				return nil, err
			}
			headerLine = strings.TrimRight(line, "\r\n")
		}
		if strings.TrimSpace(headerLine) == "" {
			break
		}
		lower := strings.ToLower(headerLine)
		if strings.HasPrefix(lower, "content-length:") {
			value := strings.TrimSpace(headerLine[len("Content-Length:"):])
			length, err := strconv.Atoi(value)
			if err != nil {
				return nil, err
			}
			contentLength = length
		}
		headerLine = ""
	}
	if contentLength < 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}
	payload := make([]byte, contentLength)
	if _, err := io.ReadFull(s.reader, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (s *mcpServer) writeMessage(message any) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	switch s.frameMode {
	case "header":
		if _, err := fmt.Fprintf(s.writer, "Content-Length: %d\r\n\r\n", len(payload)); err != nil {
			return err
		}
		if _, err := s.writer.Write(payload); err != nil {
			return err
		}
	default:
		if _, err := s.writer.Write(payload); err != nil {
			return err
		}
		if err := s.writer.WriteByte('\n'); err != nil {
			return err
		}
	}
	return s.writer.Flush()
}

func parseMCPID(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil
	}
	return decoded
}

func newMCPProtocolError(code int, message string, data map[string]any) *mcpJSONRPCError {
	return &mcpJSONRPCError{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

func objectSchemaWithRequired(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func readOnlyToolAnnotations() map[string]any {
	return map[string]any{
		"readOnlyHint":   true,
		"idempotentHint": true,
	}
}

func requiredStringArg(args map[string]any, name string) (string, error) {
	value := strings.TrimSpace(optionalStringArg(args, name))
	if value == "" {
		return "", output.NewError("VALIDATION_ERROR", "缺少必填参数", map[string]any{
			"field": name,
		})
	}
	return value, nil
}

func optionalStringArg(args map[string]any, name string) string {
	value, ok := args[name]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return stringValue(value)
	}
}

func requiredMapArg(args map[string]any, name string) (map[string]any, error) {
	value, ok := args[name]
	if !ok || value == nil {
		return nil, output.NewError("VALIDATION_ERROR", "缺少必填参数", map[string]any{
			"field": name,
		})
	}
	if typed, ok := value.(map[string]any); ok {
		return typed, nil
	}
	return nil, output.NewError("VALIDATION_ERROR", "参数必须是对象", map[string]any{
		"field": name,
	})
}

func optionalIntArg(args map[string]any, name string, fallback int) (int, error) {
	value, ok := args[name]
	if !ok || value == nil {
		return fallback, nil
	}
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return 0, output.NewError("VALIDATION_ERROR", "参数必须是整数", map[string]any{"field": name})
		}
		return int(i), nil
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, output.NewError("VALIDATION_ERROR", "参数必须是整数", map[string]any{"field": name})
		}
		return i, nil
	default:
		return 0, output.NewError("VALIDATION_ERROR", "参数必须是整数", map[string]any{"field": name})
	}
}

func mustMarshalJSON(value any) string {
	payload, err := json.Marshal(value)
	if err != nil {
		return `{"error":{"code":"FATAL","message":"failed to encode result"}}`
	}
	return string(payload)
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func waitForJobQuiet(ctx context.Context, client interface {
	GetJSON(context.Context, string, map[string]string, any) error
}, jobID string, interval time.Duration, maxPolls int) (*types.Job, error) {
	for pollIndex := 0; pollIndex < maxPolls; pollIndex++ {
		var job types.Job
		if err := client.GetJSON(ctx, "/jobs/"+jobID, nil, &job); err != nil {
			return nil, err
		}
		if pollTerminalState(job.Status) {
			if job.Status == "failed" {
				return nil, output.NewError("JOB_FAILED", pollMessageFromJob(&job), map[string]any{
					"job_id": jobID,
					"status": job.Status,
					"error":  job.Error,
				})
			}
			return &job, nil
		}
		time.Sleep(interval)
	}
	return nil, output.NewError("POLL_TIMEOUT", fmt.Sprintf("Job %s did not complete within the timeout", jobID), map[string]any{
		"job_id":          jobID,
		"timeout_seconds": int(interval.Seconds()) * maxPolls,
	})
}

func pollTerminalState(status string) bool {
	return status == "done" || status == "failed" || status == "cancelled"
}

func pollMessageFromJob(job *types.Job) string {
	if job == nil || job.Error == nil {
		return "Job failed"
	}
	if job.Error.Message != "" {
		return job.Error.Message
	}
	return "Job failed"
}
