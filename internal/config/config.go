package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const defaultEndpoint = "https://api.creatoragentos.io/v1"

type Config struct {
	Endpoint string `json:"endpoint,omitempty"`
	Token    string `json:"token,omitempty"`
	Project  string `json:"project,omitempty"`
}

type Patch struct {
	Endpoint *string
	Token    *string
	Project  *string
}

func Load() Config {
	cfg := loadStored()
	if cfg.Endpoint == "" {
		cfg.Endpoint = defaultEndpoint
	}

	if value := os.Getenv("POPIART_ENDPOINT"); value != "" {
		cfg.Endpoint = value
	}
	if value := os.Getenv("POPIART_KEY"); value != "" {
		cfg.Token = value
	}
	if value := os.Getenv("POPIART_TOKEN"); value != "" {
		cfg.Token = value
	}
	if value := os.Getenv("POPIART_PROJECT"); value != "" {
		cfg.Project = value
	}

	return cfg
}

func SavePatch(p Patch) (Config, error) {
	cfg := loadStored()
	if cfg.Endpoint == "" {
		cfg.Endpoint = defaultEndpoint
	}

	if p.Endpoint != nil {
		cfg.Endpoint = *p.Endpoint
	}
	if p.Token != nil {
		cfg.Token = *p.Token
	}
	if p.Project != nil {
		cfg.Project = *p.Project
	}

	dir := Dir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return Config{}, err
	}

	file, err := os.OpenFile(Path(), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cfg); err != nil {
		return Config{}, err
	}

	return Load(), nil
}

func Path() string {
	return filepath.Join(Dir(), "config.json")
}

func SkillsDir() string {
	return filepath.Join(Dir(), "skills")
}

func SkillDownloadsDir() string {
	return filepath.Join(SkillsDir(), "downloads")
}

func InstalledSkillsDir() string {
	return filepath.Join(SkillsDir(), "installed")
}

func SkillStatePath() string {
	return filepath.Join(SkillsDir(), "state.json")
}

func AgentDir(agent string) string {
	return filepath.Join(Dir(), "agents", agent)
}

func Dir() string {
	if dir := os.Getenv("POPIART_CONFIG_DIR"); dir != "" {
		return dir
	}
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, ".popiart")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".popiart"
	}
	return filepath.Join(home, ".popiart")
}

func loadStored() Config {
	path := Path()
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}
	}
	return cfg
}

func RequireToken() (string, error) {
	token := Load().Token
	if token == "" {
		return "", errors.New("missing token")
	}
	return token, nil
}
