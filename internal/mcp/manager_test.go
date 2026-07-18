package mcp

import "testing"

func TestToolCacheCopyAndFind(t *testing.T) {
	var cache ToolCache
	cache.Replace([]Tool{{Name: "echo"}, {Name: "sum"}})
	if got := cache.List(); len(got) != 2 {
		t.Fatalf("got %d tools", len(got))
	}
	if _, ok := cache.Find("echo"); !ok {
		t.Fatal("echo not found")
	}
	got := cache.List()
	got[0].Name = "changed"
	if found, _ := cache.Find("echo"); found.Name != "echo" {
		t.Fatal("cache leaked internal slice")
	}
}
