package types

type User struct {
	ID     string   `json:"id"`
	Email  string   `json:"email"`
	Name   string   `json:"name"`
	Scopes []string `json:"scopes,omitempty"`
}

type AuthSession struct {
	User              *User    `json:"user,omitempty"`
	ID                string   `json:"id,omitempty"`
	Email             string   `json:"email,omitempty"`
	Name              string   `json:"name,omitempty"`
	Scopes            []string `json:"scopes,omitempty"`
	SessionKey        string   `json:"session_key,omitempty"`
	UpstreamKeyMasked string   `json:"upstream_key_masked,omitempty"`
}

type LoginResponse struct {
	Key   string `json:"key,omitempty"`
	Token string `json:"token,omitempty"`
	User  User   `json:"user"`
}

type SkillSummary struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	Tags               []string `json:"tags"`
	Version            string   `json:"version"`
	ModelType          string   `json:"model_type"`
	EstimatedDurationS int      `json:"estimated_duration_s"`
}

type SkillListResponse struct {
	Items  []SkillSummary `json:"items"`
	Total  int            `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

type Skill struct {
	ID                 string         `json:"id"`
	Name               string         `json:"name"`
	Description        string         `json:"description"`
	Tags               []string       `json:"tags"`
	Version            string         `json:"version"`
	InputSchema        map[string]any `json:"input_schema"`
	OutputSchema       map[string]any `json:"output_schema"`
	ModelType          string         `json:"model_type"`
	EstimatedDurationS int            `json:"estimated_duration_s"`
}

type SkillSchemaResponse struct {
	InputSchema  map[string]any `json:"input_schema"`
	OutputSchema map[string]any `json:"output_schema"`
}

type Job struct {
	JobID       string         `json:"job_id"`
	Status      string         `json:"status"`
	SkillID     string         `json:"skill_id,omitempty"`
	Input       map[string]any `json:"input,omitempty"`
	CreatedAt   string         `json:"created_at,omitempty"`
	StartedAt   string         `json:"started_at,omitempty"`
	FinishedAt  string         `json:"finished_at,omitempty"`
	ArtifactIDs []string       `json:"artifact_ids,omitempty"`
	Error       *JobError      `json:"error,omitempty"`
}

type JobError struct {
	Code    string         `json:"code,omitempty"`
	Message string         `json:"message,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

type JobListResponse struct {
	Items  []Job `json:"items"`
	Total  int   `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
}

type LogEntry struct {
	TS      string `json:"ts"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

type ArtifactSummary struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	CreatedAt   string `json:"created_at"`
}

type ArtifactListResponse struct {
	Items []ArtifactSummary `json:"items"`
}

type Artifact struct {
	ID          string `json:"id"`
	JobID       string `json:"job_id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	CreatedAt   string `json:"created_at"`
	ExpiresAt   string `json:"expires_at"`
}
