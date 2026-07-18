package mcp

import (
	"encoding/json"
	"fmt"
	"os"
)

type ServerConfig struct {
	Name      string            `json:"name"`
	Transport string            `json:"transport,omitempty"`
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	URL       string            `json:"url,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	Enabled   bool              `json:"enabled"`
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
		if s.Transport == "" {
			s.Transport = "stdio"
		}
		if s.Name == "" {
			return Config{}, fmt.Errorf("servers[%d]: name is required", i)
		}
		switch s.Transport {
		case "stdio":
			if s.Command == "" {
				return Config{}, fmt.Errorf("servers[%d]: command is required for stdio", i)
			}
		case "streamable-http":
			if s.URL == "" {
				return Config{}, fmt.Errorf("servers[%d]: url is required for streamable-http", i)
			}
		default:
			return Config{}, fmt.Errorf("servers[%d]: unsupported transport %q", i, s.Transport)
		}
		if seen[s.Name] {
			return Config{}, fmt.Errorf("duplicate MCP server name: %s", s.Name)
		}
		seen[s.Name] = true
	}
	return cfg, nil
}
