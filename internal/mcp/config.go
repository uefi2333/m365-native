package mcp

import (
	"encoding/json"
	"fmt"
	"os"
)

type ServerConfig struct {
	Name    string            `json:"name"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Enabled bool              `json:"enabled"`
}

type Config struct {
	Servers []ServerConfig `json:"servers"`
}

func LoadConfig(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Config{}, err
	}
	seen := map[string]bool{}
	for i := range cfg.Servers {
		s := &cfg.Servers[i]
		if s.Name == "" || s.Command == "" {
			return Config{}, fmt.Errorf("servers[%d]: name and command are required", i)
		}
		if seen[s.Name] {
			return Config{}, fmt.Errorf("duplicate MCP server name: %s", s.Name)
		}
		seen[s.Name] = true
	}
	return cfg, nil
}
