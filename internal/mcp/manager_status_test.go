package mcp

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestManagerStatuses(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses python3")
	}
	server := `import sys,json
for line in sys.stdin:
 r=json.loads(line); m=r.get("method"); p=r.get("params",{})
 if m=="initialize": out={"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"serverInfo":{"name":"status-test"}}
 elif m=="tools/list": out={"tools":[{"name":"echo"}]}
 else: out={"content":[{"type":"text","text":"ok"}]}
 if "id" in r: print(json.dumps({"jsonrpc":"2.0","id":r["id"],"result":out}),flush=True)`
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	manager := NewManager()
	if err := manager.Start(ctx, Config{Servers: []ServerConfig{{Name: "demo", Command: "python3", Args: []string{"-c", server}, Enabled: true}}}); err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	statuses := manager.Statuses()
	if len(statuses) != 1 || statuses[0].Status != "running" || statuses[0].ToolCount != 1 || statuses[0].LastRefreshAt.IsZero() {
		t.Fatalf("statuses=%+v", statuses)
	}
	if err := manager.Refresh(ctx, "missing"); err == nil {
		t.Fatal("expected missing server error")
	}
}
