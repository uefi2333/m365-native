package web

import (
	"strings"

	"m365-native/internal/chathub"
)

// routeKind describes capabilities requested by one API call. It is kept
// provider-neutral so OpenAI, Anthropic, and the native endpoints can share
// the same dispatch decision.
type routeKind string

const (
	routeChat       routeKind = "chat"
	routeReasoning  routeKind = "reasoning"
	routeTools      routeKind = "tools"
	routeMultimodal routeKind = "multimodal"
	routeStreaming  routeKind = "streaming"
)

type requestRoute struct {
	Kinds       []routeKind
	Tone        string
	HasTools    bool
	HasFiles    bool
	Streaming   bool
	Reasoning   bool
	NeedsRouter bool
}

func classifyRoute(modelOrTone string, tools []chathub.Tool, attachments []chathub.Attachment, streaming bool, reasoning bool) requestRoute {
	r := requestRoute{Tone: routeTone(modelOrTone), Streaming: streaming, Reasoning: reasoning, HasTools: len(tools) > 0, HasFiles: len(attachments) > 0}
	if r.Reasoning || strings.Contains(strings.ToLower(modelOrTone), "reason") || strings.Contains(strings.ToLower(modelOrTone), "think") {
		r.Kinds = append(r.Kinds, routeReasoning)
	}
	if r.HasTools {
		r.Kinds = append(r.Kinds, routeTools)
		r.NeedsRouter = true
	}
	if r.HasFiles {
		r.Kinds = append(r.Kinds, routeMultimodal)
	}
	if r.Streaming {
		r.Kinds = append(r.Kinds, routeStreaming)
	}
	// Every request has a chat transport; other kinds are capability overlays.
	if !hasRouteKind(r.Kinds, routeChat) {
		r.Kinds = append([]routeKind{routeChat}, r.Kinds...)
	}
	return r
}

func routeTone(value string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return "magic"
	}
	known := map[string]bool{"magic": true, "Gpt_5_2_Chat": true, "Gpt_5_2_Reasoning": true, "Gpt_5_3_Chat": true, "Gpt_5_4_Chat": true, "Gpt_5_4_Reasoning": true, "Gpt_5_5_Chat": true, "Gpt_5_5_Reasoning": true, "Claude_Sonnet": true, "Claude_Sonnet_Reasoning": true, "Gpt_Quick": true, "Gpt_Reasoning": true}
	if known[v] {
		return v
	}
	return modelTone(v)
}

func hasRouteKind(kinds []routeKind, want routeKind) bool {
	for _, k := range kinds {
		if k == want {
			return true
		}
	}
	return false
}
