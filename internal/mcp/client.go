package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
}

type CallResult struct {
	Content        []map[string]any `json:"content,omitempty"`
	StructuredData any              `json:"structuredContent,omitempty"`
	IsError        bool             `json:"isError,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Client struct {
	cmd       *exec.Cmd
	in        io.WriteCloser
	out       *bufio.Reader
	mu        sync.Mutex
	nextID    int64
	closed    bool
	toolCache ToolCache
}

func StartStdio(ctx context.Context, command string, args []string, env map[string]string) (*Client, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		_ = in.Close()
		return nil, err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &Client{cmd: cmd, in: in, out: bufio.NewReader(out), nextID: 1}, nil
}

func (c *Client) request(ctx context.Context, method string, params any, dst any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return errors.New("mcp client is closed")
	}
	id := c.nextID
	c.nextID++
	req := map[string]any{"jsonrpc": "2.0", "id": id, "method": method}
	if params != nil {
		req["params"] = params
	}
	b, _ := json.Marshal(req)
	b = append(b, '\n')
	if _, err := c.in.Write(b); err != nil {
		return err
	}
	result := make(chan rpcResponse, 1)
	errch := make(chan error, 1)
	go func() {
		for {
			line, err := c.out.ReadBytes('\n')
			if err != nil {
				errch <- err
				return
			}
			var r rpcResponse
			if err := json.Unmarshal(line, &r); err != nil {
				errch <- err
				return
			}
			if r.ID == nil {
				// Ignore server notifications and continue until our response arrives.
				continue
			}
			result <- r
			return
		}
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errch:
		return err
	case r := <-result:
		if r.Error != nil {
			return fmt.Errorf("MCP error %d: %s", r.Error.Code, r.Error.Message)
		}
		if dst != nil {
			return json.Unmarshal(r.Result, dst)
		}
		return nil
	}
}

func (c *Client) notify(method string, params any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return errors.New("mcp client is closed")
	}
	req := map[string]any{"jsonrpc": "2.0", "method": method}
	if params != nil {
		req["params"] = params
	}
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = c.in.Write(b)
	return err
}

func (c *Client) Initialize(ctx context.Context) error {
	var result map[string]any
	if err := c.request(ctx, "initialize", map[string]any{"protocolVersion": "2024-11-05", "capabilities": map[string]any{}, "clientInfo": map[string]any{"name": "m365-native", "version": "0.1.0"}}, &result); err != nil {
		return err
	}
	return c.notify("notifications/initialized", nil)
}
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	var r struct {
		Tools []Tool `json:"tools"`
	}
	err := c.request(ctx, "tools/list", nil, &r)
	return r.Tools, err
}
func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (CallResult, error) {
	var r CallResult
	err := c.request(ctx, "tools/call", map[string]any{"name": name, "arguments": args}, &r)
	return r, err
}
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	_ = c.in.Close()
	return c.cmd.Process.Kill()
}

var defaultTimeout = 30 * time.Second

func WithDefaultTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, defaultTimeout)
}
