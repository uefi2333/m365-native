package mcp

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestManagerStdioLifecycleAndNamespacedCall(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses python3")
	}
	server := `import sys,json
for line in sys.stdin:
 r=json.loads(line); m=r.get("method"); p=r.get("params",{})
 if m=="initialize": out={"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"serverInfo":{"name":"manager-test"}}
 elif m=="tools/list": out={"tools":[{"name":"echo","description":"Echo text","inputSchema":{"type":"object"}}]}
 elif m=="tools/call": out={"content":[{"type":"text","text":p.get("arguments",{}).get("text","")}]}
 else: out={}
 if "id" in r: print(json.dumps({"jsonrpc":"2.0","id":r["id"],"result":out}),flush=True)`
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	manager := NewManager()
	if err := manager.Start(ctx, Config{Servers: []ServerConfig{{Name: "demo", Command: "python3", Args: []string{"-c", server}, Enabled: true}}}); err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	tools := manager.Tools()
	if len(tools) != 1 || tools[0].Name != "demo/echo" {
		t.Fatalf("tools=%v", tools)
	}
	result, err := manager.Call(ctx, "demo/echo", map[string]any{"text": "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Text(); got != "hello" {
		t.Fatalf("result text=%q", got)
	}
}
