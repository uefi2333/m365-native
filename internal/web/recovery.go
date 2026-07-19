package web

import (
	"context"
	"fmt"
	"strings"

	"m365-native/internal/chathub"
)

// isRecoverableChatHubError deliberately matches transport/session failures,
// not authentication, validation, or tool-protocol errors.
func isRecoverableChatHubError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, marker := range []string{
		"ws read before completion", "ws dial", "handshake recv", "handshake send",
		"connection reset", "unexpected eof", "use of closed network connection",
		"chathub completion error", "i/o timeout", "timeout",
	} {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	for _, marker := range []string{"missing access token", "missing oid", "invalid", "unauthorized", "forbidden", "empty prompt", "upload attachment"} {
		if strings.Contains(msg, marker) {
			return false
		}
	}
	return false
}

func (s *Server) clearSession(accountID, sessionKey string) {
	if sessionKey == "" {
		return
	}
	if v, ok := s.sessions.getForAccount(sessionKey, accountID); ok {
		v.ConversationID = ""
		v.SessionID = ""
		s.sessions.upsert(v)
	}
}

// chatWithRecovery retries once with a fresh M365 conversation only for
// transport/session failures. It never retries arbitrary provider errors.
func (s *Server) chatWithRecovery(ctx context.Context, account chathub.Account, req chathub.Request, accountID, sessionKey string) (chathub.Result, error) {
	res, err := s.chat.Chat(ctx, account, req)
	if err == nil || !isRecoverableChatHubError(err) || sessionKey == "" {
		return res, err
	}
	s.clearSession(accountID, sessionKey)
	req.ConversationID = ""
	req.SessionID = ""
	res, retryErr := s.chat.Chat(ctx, account, req)
	if retryErr != nil {
		return chathub.Result{}, fmt.Errorf("chathub recovery failed: initial=%v retry=%w", err, retryErr)
	}
	return res, nil
}
