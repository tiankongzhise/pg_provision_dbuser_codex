package provision

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

type Draft struct {
	ID        string
	Owner     string
	Request   Request
	SQL       string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type DraftStore struct {
	mu      sync.Mutex
	now     func() time.Time
	ttl     time.Duration
	entries map[string]Draft
}

func NewDraftStore(ttl time.Duration) *DraftStore {
	return &DraftStore{
		now:     time.Now,
		ttl:     ttl,
		entries: make(map[string]Draft),
	}
}

func (s *DraftStore) Save(owner string, req Request) (Draft, error) {
	id, err := randomID()
	if err != nil {
		return Draft{}, err
	}

	now := s.now()
	req = req.Normalized()
	draft := Draft{
		ID:        id,
		Owner:     owner,
		Request:   req,
		SQL:       PreviewSQL(req),
		CreatedAt: now,
		ExpiresAt: now.Add(s.ttl),
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked(now)
	s.entries[id] = draft
	return draft, nil
}

func (s *DraftStore) Get(id, owner string) (Draft, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	s.cleanupLocked(now)
	draft, ok := s.entries[id]
	if !ok || draft.Owner != owner || !now.Before(draft.ExpiresAt) {
		return Draft{}, false
	}
	return draft, true
}

func (s *DraftStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, id)
}

func (s *DraftStore) cleanupLocked(now time.Time) {
	for id, draft := range s.entries {
		if !now.Before(draft.ExpiresAt) {
			delete(s.entries, id)
		}
	}
}

func randomID() (string, error) {
	raw := make([]byte, 18)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
