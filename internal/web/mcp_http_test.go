package web

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestMCPStatusEmptyManager(t *testing.T) {
	s, err := New()
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", "/api/admin/mcp/status", nil)
	w := httptest.NewRecorder()
	s.mcpStatus(w, r)
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Servers []any `json:"servers"`
		Tools   []any `json:"tools"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Servers == nil || body.Tools == nil {
		t.Fatalf("expected arrays, body=%s", w.Body.String())
	}
}
