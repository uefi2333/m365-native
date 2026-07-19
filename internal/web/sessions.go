package web

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

type conversation struct {
	ID             string    `json:"id"`
	AccountID      string    `json:"accountId"`
	ConversationID string    `json:"conversationId"`
	SessionID      string    `json:"sessionId"`
	Title          string    `json:"title,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type sessionStore struct {
	mu    sync.Mutex
	path  string
	data  map[string]conversation
	locks map[string]*sync.Mutex
}

func openSessionStore() *sessionStore {
	path := os.Getenv("M365_SESSION_CACHE")
	if path == "" {
		path = filepath.Join(os.TempDir(), "m365-native-sessions.json")
	}
	s := &sessionStore{path: path, data: map[string]conversation{}, locks: map[string]*sync.Mutex{}}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &s.data)
	}
	return s
}

func (s *sessionStore) saveLocked() {
	b, _ := json.MarshalIndent(s.data, "", "  ")
	_ = os.MkdirAll(filepath.Dir(s.path), 0o700)
	_ = os.WriteFile(s.path, b, 0o600)
}

func (s *sessionStore) list() []conversation {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]conversation, 0, len(s.data))
	for _, v := range s.data {
		out = append(out, v)
	}
	return out
}

func (s *sessionStore) get(id string) (conversation, bool) {
	return s.getForAccount(id, "")
}

// getForAccount prevents a caller that explicitly selected another account
// from silently inheriting the previous account's M365 ConversationId.
func (s *sessionStore) getForAccount(id, accountID string) (conversation, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data[id]
	if !ok || (accountID != "" && v.AccountID != "" && v.AccountID != accountID) {
		return conversation{}, false
	}
	return v, true
}

func (s *sessionStore) upsert(v conversation) conversation {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v.ID == "" {
		v.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	if v.CreatedAt.IsZero() {
		v.CreatedAt = now
	}
	v.UpdatedAt = now
	s.data[v.ID] = v
	s.saveLocked()
	return v
}

func (s *sessionStore) lockSession(accountID, sessionKey string) func() {
	if sessionKey == "" {
		return func() {}
	}
	key := accountID + "\x00" + sessionKey
	s.mu.Lock()
	if s.locks == nil {
		s.locks = map[string]*sync.Mutex{}
	}
	lock := s.locks[key]
	if lock == nil {
		lock = &sync.Mutex{}
		s.locks[key] = lock
	}
	s.mu.Unlock()
	lock.Lock()
	return lock.Unlock
}

func (s *sessionStore) delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[id]; !ok {
		return false
	}
	delete(s.data, id)
	s.saveLocked()
	return true
}
