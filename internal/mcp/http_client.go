package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
)

type httpTransport struct {
	client  *http.Client
	url     string
	headers map[string]string
	mu      sync.Mutex
	nextID  int64
	session string
	closed  bool
}

func NewStreamableHTTP(url string, headers map[string]string) *Client {
	return &Client{http: &httpTransport{client: &http.Client{}, url: url, headers: headers, nextID: 1}}
}

func (t *httpTransport) request(ctx context.Context, method string, params any, dst any) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return errors.New("mcp client is closed")
	}
	id := t.nextID
	t.nextID++
	body := map[string]any{"jsonrpc": "2.0", "id": id, "method": method}
	if params != nil {
		body["params"] = params
	}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}
	if t.session != "" {
		req.Header.Set("MCP-Session-Id", t.session)
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if sid := resp.Header.Get("MCP-Session-Id"); sid != "" {
		t.session = sid
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("MCP HTTP %s: %s", resp.Status, string(data))
	}
	var r rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return err
	}
	if r.Error != nil {
		return fmt.Errorf("MCP error %d: %s", r.Error.Code, r.Error.Message)
	}
	if dst != nil {
		return json.Unmarshal(r.Result, dst)
	}
	return nil
}

func (t *httpTransport) notify(ctx context.Context, method string, params any) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	body := map[string]any{"jsonrpc": "2.0", "method": method}
	if params != nil {
		body["params"] = params
	}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}
	if t.session != "" {
		req.Header.Set("MCP-Session-Id", t.session)
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if sid := resp.Header.Get("MCP-Session-Id"); sid != "" {
		t.session = sid
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("MCP HTTP %s: %s", resp.Status, string(data))
	}
	return nil
}
func (t *httpTransport) close() error { t.mu.Lock(); t.closed = true; t.mu.Unlock(); return nil }
