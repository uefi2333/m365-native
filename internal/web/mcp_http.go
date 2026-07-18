package web

import (
	"context"
	"encoding/json"
	"m365-native/internal/mcp"
	"net/http"
	"time"
)

func mcpToolsOrEmpty(tools []mcp.Tool) []mcp.Tool {
	if tools == nil {
		return []mcp.Tool{}
	}
	return tools
}

func (s *Server) mcpStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "invalid_request_error", "method not allowed")
		return
	}
	jsonOut(w, map[string]any{
		"servers": s.mcpManager.Statuses(),
		"tools":   mcpToolsOrEmpty(s.mcpManager.Tools()),
	})
}

func (s *Server) mcpRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "invalid_request_error", "method not allowed")
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body); err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "bad json")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if body.Name != "" {
		if err := s.mcpManager.Refresh(ctx, body.Name); err != nil {
			writeOpenAIError(w, http.StatusBadRequest, "mcp_error", err.Error())
			return
		}
	} else {
		for _, status := range s.mcpManager.Statuses() {
			if err := s.mcpManager.Refresh(ctx, status.Name); err != nil {
				writeOpenAIError(w, http.StatusBadRequest, "mcp_error", err.Error())
				return
			}
		}
	}
	jsonOut(w, map[string]any{
		"status":  "refreshed",
		"servers": s.mcpManager.Statuses(),
		"tools":   mcpToolsOrEmpty(s.mcpManager.Tools()),
	})
}
