package web

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseContentAcceptsResponsesTextBlocks(t *testing.T) {
	content := []any{
		map[string]any{"type": "input_text", "text": "input"},
		map[string]any{"type": "output_text", "text": " output"},
	}
	text, files := parseContent(content)
	if text != "input output" || len(files) != 0 {
		t.Fatalf("text=%q files=%#v", text, files)
	}
}

func TestResponsesStreamEmitsFailedForInnerRequestError(t *testing.T) {
	s := &Server{}
	r := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(`{"model":"gpt-5.5","input":[],"stream":true}`))
	w := httptest.NewRecorder()
	s.responses(w, r)
	body := w.Body.String()
	for _, want := range []string{"event: response.created", "event: response.failed", `"status":"failed"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
	if strings.Contains(body, "event: response.completed") {
		t.Fatalf("unexpected completion event in %s", body)
	}
}
