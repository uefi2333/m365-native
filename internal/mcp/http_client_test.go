package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStreamableHTTPRoundTrip(t *testing.T) {
	var sessionSeen bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("MCP-Session-Id") == "session-1" {
			sessionSeen = true
		}
		var req struct {
			ID     int64          `json:"id"`
			Method string         `json:"method"`
			Params map[string]any `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("MCP-Session-Id", "session-1")
		var result any = map[string]any{}
		switch req.Method {
		case "initialize":
			result = map[string]any{"protocolVersion": "2024-11-05"}
		case "tools/list":
			result = map[string]any{"tools": []map[string]any{{"name": "echo"}}}
		case "tools/call":
			result = map[string]any{"content": []map[string]any{{"type": "text", "text": req.Params["arguments"].(map[string]any)["text"]}}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	defer srv.Close()
	c := NewStreamableHTTP(srv.URL, map[string]string{"X-Test": "ok"})
	ctx := context.Background()
	if err := c.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	tools, err := c.ListTools(ctx)
	if err != nil || len(tools) != 1 {
		t.Fatalf("tools=%v err=%v", tools, err)
	}
	result, err := c.CallTool(ctx, "echo", map[string]any{"text": "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text() != "hello" || !sessionSeen {
		t.Fatalf("result=%q session=%v", result.Text(), sessionSeen)
	}
}
