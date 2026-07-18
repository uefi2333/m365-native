package mcp

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

type ManagedServer struct {
	Name   string
	Client *Client
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
		c, err := StartStdio(ctx, spec.Command, spec.Args, spec.Env)
		if err != nil {
			m.Close()
			return fmt.Errorf("start MCP server %q: %w", spec.Name, err)
		}
		if err := c.Initialize(ctx); err != nil {
			c.Close()
			m.Close()
			return fmt.Errorf("initialize MCP server %q: %w", spec.Name, err)
		}
		if _, err := c.RefreshTools(ctx); err != nil {
			c.Close()
			m.Close()
			return fmt.Errorf("list tools from MCP server %q: %w", spec.Name, err)
		}
		m.mu.Lock()
		m.servers[spec.Name] = &ManagedServer{Name: spec.Name, Client: c}
		m.mu.Unlock()
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
