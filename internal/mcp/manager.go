package mcp

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

type ManagedServer struct {
	Name          string
	Config        ServerConfig
	Client        *Client
	Status        string
	ToolCount     int
	StartedAt     time.Time
	LastRefreshAt time.Time
	LastError     string
}

type ServerStatus struct {
	Name          string    `json:"name"`
	Enabled       bool      `json:"enabled"`
	Status        string    `json:"status"`
	ToolCount     int       `json:"toolCount"`
	StartedAt     time.Time `json:"startedAt,omitempty"`
	LastRefreshAt time.Time `json:"lastRefreshAt,omitempty"`
	LastError     string    `json:"lastError,omitempty"`
}

type Manager struct {
	mu      sync.RWMutex
	servers map[string]*ManagedServer
}

func NewManager() *Manager { return &Manager{servers: map[string]*ManagedServer{}} }

func (m *Manager) Start(ctx context.Context, cfg Config) error {
	for _, spec := range cfg.Servers {
		if !spec.Enabled {
			continue
		}
		var c *Client
		var err error
		switch spec.Transport {
		case "", "stdio":
			c, err = StartStdio(ctx, spec.Command, spec.Args, spec.Env)
		case "streamable-http":
			c = NewStreamableHTTP(spec.URL, spec.Headers)
		default:
			err = fmt.Errorf("unsupported transport %q", spec.Transport)
		}
		if err != nil {
			m.Close()
			return fmt.Errorf("start MCP server %q: %w", spec.Name, err)
		}
		if err := c.Initialize(ctx); err != nil {
			c.Close()
			m.Close()
			return fmt.Errorf("initialize MCP server %q: %w", spec.Name, err)
		}
		server := &ManagedServer{Name: spec.Name, Config: spec, Client: c, Status: "running", StartedAt: time.Now()}
		m.mu.Lock()
		m.servers[spec.Name] = server
		m.mu.Unlock()
		if err := m.Refresh(ctx, spec.Name); err != nil {
			c.Close()
			m.Close()
			return fmt.Errorf("list tools from MCP server %q: %w", spec.Name, err)
		}
	}
	return nil
}

func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, server := range m.servers {
		_ = server.Client.Close()
		delete(m.servers, name)
	}
}

func (m *Manager) Refresh(ctx context.Context, name string) error {
	m.mu.RLock()
	server, ok := m.servers[name]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("MCP server not found: %s", name)
	}
	tools, err := server.Client.RefreshTools(ctx)
	m.mu.Lock()
	defer m.mu.Unlock()
	if err != nil {
		server.Status = "error"
		server.LastError = err.Error()
		return err
	}
	server.Status = "running"
	server.ToolCount = len(tools)
	server.LastRefreshAt = time.Now()
	server.LastError = ""
	return nil
}

func (m *Manager) Statuses() []ServerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]ServerStatus, 0, len(m.servers))
	for _, server := range m.servers {
		out = append(out, ServerStatus{Name: server.Name, Enabled: server.Config.Enabled, Status: server.Status, ToolCount: server.ToolCount, StartedAt: server.StartedAt, LastRefreshAt: server.LastRefreshAt, LastError: server.LastError})
	}
	return out
}

func (m *Manager) Tools() []Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Tool
	for name, server := range m.servers {
		for _, tool := range server.Client.CachedTools() {
			tool.Name = name + "/" + tool.Name
			out = append(out, tool)
		}
	}
	return out
}

func (m *Manager) Call(ctx context.Context, qualifiedName string, args map[string]any) (CallResult, error) {
	parts := strings.SplitN(qualifiedName, "/", 2)
	if len(parts) != 2 {
		return CallResult{}, fmt.Errorf("MCP tool name must be server/tool: %s", qualifiedName)
	}
	m.mu.RLock()
	server, ok := m.servers[parts[0]]
	m.mu.RUnlock()
	if !ok {
		return CallResult{}, fmt.Errorf("MCP server not found: %s", parts[0])
	}
	return server.Client.CallTool(ctx, parts[1], args)
}
