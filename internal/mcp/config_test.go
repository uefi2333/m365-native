package mcp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigValidation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	if err := os.WriteFile(path, []byte(`{"servers":[{"name":"demo","command":"python3","enabled":true}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil || len(cfg.Servers) != 1 || cfg.Servers[0].Name != "demo" {
		t.Fatalf("cfg=%+v err=%v", cfg, err)
	}
	if err := os.WriteFile(path, []byte(`{"servers":[{"name":"demo","command":"x"},{"name":"demo","command":"y"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("expected duplicate server error")
	}
}
