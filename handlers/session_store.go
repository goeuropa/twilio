package handlers

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"oba-twilio/models"
)

const (
	sessionTimeoutMinutes  = 10
	cleanupIntervalMinutes = 5
)

var phoneRegex = regexp.MustCompile(`^\+1\d{10}$`)

type SessionStore struct {
	sessions  map[string]*models.DisambiguationSession
	mutex     sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	closeOnce sync.Once
}

func NewSessionStore() *SessionStore {
	ctx, cancel := context.WithCancel(context.Background())
	store := &SessionStore{
		sessions: make(map[string]*models.DisambiguationSession),
		ctx:      ctx,
		cancel:   cancel,
	}

	go store.cleanupExpiredSessions()

	return store
}

func (s *SessionStore) SetDisambiguationSession(phoneNumber string, session *models.DisambiguationSession) error {
	if !phoneRegex.MatchString(phoneNumber) {
		return fmt.Errorf("invalid phone number format: %s", phoneNumber)
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	session.CreatedAt = time.Now().Unix()
	s.sessions[phoneNumber] = session
	return nil
}

func (s *SessionStore) GetDisambiguationSession(phoneNumber string) *models.DisambiguationSession {
	if !phoneRegex.MatchString(phoneNumber) {
		return nil // Invalid phone number format
	}

	s.mutex.RLock()
	session, exists := s.sessions[phoneNumber]
	if !exists {
		s.mutex.RUnlock()
		return nil
	}

	// Check expiry while holding read lock
	if time.Now().Unix()-session.CreatedAt > sessionTimeoutMinutes*60 {
		s.mutex.RUnlock()
		// Upgrade to write lock to delete expired session
		s.mutex.Lock()
		// Double-check in case another goroutine already deleted it
		if existingSession, stillExists := s.sessions[phoneNumber]; stillExists &&
			time.Now().Unix()-existingSession.CreatedAt > sessionTimeoutMinutes*60 {
			delete(s.sessions, phoneNumber)
		}
		s.mutex.Unlock()
		return nil
	}
	s.mutex.RUnlock()

	return session
}

func (s *SessionStore) ClearDisambiguationSession(phoneNumber string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.sessions, phoneNumber)
}

func (s *SessionStore) Close() {
	s.closeOnce.Do(func() {
		s.cancel()
	})
}

func (s *SessionStore) cleanupExpiredSessions() {
	ticker := time.NewTicker(cleanupIntervalMinutes * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.mutex.Lock()
			now := time.Now().Unix()

			for phoneNumber, session := range s.sessions {
				if now-session.CreatedAt > sessionTimeoutMinutes*60 {
					delete(s.sessions, phoneNumber)
				}
			}
			s.mutex.Unlock()
		}
	}
}
