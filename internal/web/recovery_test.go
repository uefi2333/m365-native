package web

import (
	"errors"
	"testing"
)

func TestRecoverableChatHubErrors(t *testing.T) {
	for _, tc := range []struct {
		msg  string
		want bool
	}{
		{"ws read before completion: unexpected EOF", true},
		{"ws dial: connection reset by peer", true},
		{"chathub completion error: timeout", true},
		{"missing access token / oid / tid", false},
		{"empty prompt and no attachments", false},
		{"tool protocol error", false},
	} {
		if got := isRecoverableChatHubError(errors.New(tc.msg)); got != tc.want {
			t.Fatalf("%q: got %v want %v", tc.msg, got, tc.want)
		}
	}
}
