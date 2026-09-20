package command

import (
	"hsbot/internal/core/domain"
	"sync"
	"time"
)

type Conversation struct {
	messages  []domain.Prompt
	expiresAt time.Time
}

type conversationStore struct {
	mu       sync.Mutex
	convos   map[int64]*Conversation
	duration time.Duration
}

func newConversationStore(duration time.Duration) *conversationStore {
	return &conversationStore{
		convos:   make(map[int64]*Conversation),
		duration: duration,
	}
}

func (s *conversationStore) getOrCreate(chatID int64) *Conversation {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.purgeExpired(now)

	if c, ok := s.convos[chatID]; ok {
		c.expiresAt = now.Add(s.duration)
		return c
	}

	c := &Conversation{expiresAt: now.Add(s.duration)}
	s.convos[chatID] = c
	return c
}

func (s *conversationStore) get(chatID int64) (*Conversation, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.purgeExpired(time.Now())
	c, ok := s.convos[chatID]
	return c, ok
}

func (s *conversationStore) delete(chatID int64) (*Conversation, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.convos[chatID]
	if !ok {
		return nil, false
	}

	delete(s.convos, chatID)
	return c, true
}

func (s *conversationStore) len() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.purgeExpired(time.Now())
	return len(s.convos)
}

func (s *conversationStore) purgeExpired(now time.Time) {
	for id, c := range s.convos {
		if !now.Before(c.expiresAt) {
			delete(s.convos, id)
		}
	}
}
