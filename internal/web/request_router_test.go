package web

import (
	"testing"

	"m365-native/internal/chathub"
)

func hasRoute(r requestRoute, want routeKind) bool {
	for _, k := range r.Kinds {
		if k == want {
			return true
		}
	}
	return false
}

func TestClassifyRouteChat(t *testing.T) {
	r := classifyRoute("gpt-5.4", nil, nil, false, false)
	if !hasRoute(r, routeChat) || r.NeedsRouter || r.Tone != "Gpt_5_4_Chat" {
		t.Fatalf("route=%+v", r)
	}
}

func TestClassifyRouteToolsAndReasoning(t *testing.T) {
	r := classifyRoute("gpt-5.4-reasoning", []chathub.Tool{{}}, nil, false, false)
	if !hasRoute(r, routeReasoning) || !hasRoute(r, routeTools) || !r.NeedsRouter {
		t.Fatalf("route=%+v", r)
	}
}

func TestClassifyRouteMultimodalStreaming(t *testing.T) {
	r := classifyRoute("claude", nil, []chathub.Attachment{{}}, true, false)
	if !hasRoute(r, routeMultimodal) || !hasRoute(r, routeStreaming) || !hasRoute(r, routeChat) {
		t.Fatalf("route=%+v", r)
	}
}
