package web

import (
	"encoding/json"
	"fmt"
	"strings"
)

func compactRouterRequest(prompt string) string {
	const maxBytes = 16000
	if len(prompt) <= maxBytes {
		return prompt
	}
	// Preserve system/developer constraints at the front and the current turn
	// at the end. Tool history is represented separately by EVIDENCE_LEDGER.
	front := 6000
	back := maxBytes - front
	return prompt[:front] + fmt.Sprintf("\n... [router context truncated %d bytes] ...\n", len(prompt)-maxBytes) + prompt[len(prompt)-back:]
}

func modelToolRouterPrompt(prompt string, tools []map[string]any, choice any) string {
	defs, _ := json.Marshal(tools)
	mode := normalizedToolChoiceMode(choice)
	prompt = compactRouterRequest(prompt)
	// Keep the tool schemas lossless; only remove redundant prose around the
	// router contract. This reduces tokens without changing call semantics.
	return fmt.Sprintf(`Return JSON only for the next tool action.
Schema: {"calls":[{"name":"function_name","arguments":{}}]}
Rules: names must come from FUNCTION_DEFINITIONS; arguments must satisfy schemas; use the multi-turn evidence; Completed evidence must not be repeated; if unfinished work remains, select the next applicable action; use [] when no external action is needed; MODE required must return a call; no markdown or commentary. Prefer an existing purpose-built function tool whenever it matches the requested operation (for example search, browser, file, edit, write, database, or API tools). Use a shell/exec/command tool only when no matching purpose-built tool exists, or when the request explicitly asks for a command. Never replace a matching specialized tool with shell merely because shell can do the same thing.
MODE: %s
FUNCTION_DEFINITIONS: %s
REQUEST_AND_EVIDENCE: %s`, mode, defs, prompt)
}

func parseModelToolDecision(text string, tools []map[string]any, choice any) ([]detectedToolCall, bool) {
	text = strings.TrimSpace(text)
	if i := strings.Index(text, "```"); i >= 0 {
		text = strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(text[i+3:], "```"), "json"))
	}
	start, end := strings.Index(text, "{"), strings.LastIndex(text, "}")
	if start < 0 || end <= start {
		return nil, false
	}
	var envelope struct {
		Calls []struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		} `json:"calls"`
	}
	if json.Unmarshal([]byte(text[start:end+1]), &envelope) != nil {
		return nil, false
	}
	out := make([]detectedToolCall, 0, len(envelope.Calls))
	for i, c := range envelope.Calls {
		fn := toolFunction(c.Name, tools)
		if fn == nil || c.Arguments == nil || !toolChoiceAllows(choice, c.Name) || schemaValid(c.Arguments, fn) != nil {
			continue
		}
		b, _ := json.Marshal(c.Arguments)
		out = append(out, detectedToolCall{ID: callID(c.Name, string(b), i), Type: toolType(c.Name, tools), Name: c.Name, Arguments: b})
	}
	return out, true
}
