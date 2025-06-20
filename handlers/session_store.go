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
	sessionTimeoutMinutes    = 10
	smsSessionTimeoutMinutes = 15
	cleanupIntervalMinutes   = 5
	maxSessions              = 10000
)

var phoneRegex = regexp.MustCompile(`^\+1\d{10}$`)

type SessionStore struct {
	sessions      map[string]*models.DisambiguationSession
	voiceSessions map[string]*models.VoiceSession
	smsSessions   map[string]*models.SMSSession
	accessTimes   map[string]int64 // Track last access time for LRU
	mutex         sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	closeOnce     sync.Once
}

func NewSessionStore() *SessionStore {
	ctx, cancel := context.WithCancel(context.Background())
	store := &SessionStore{
		sessions:      make(map[string]*models.DisambiguationSession),
		voiceSessions: make(map[string]*models.VoiceSession),
		smsSessions:   make(map[string]*models.SMSSession),
		accessTimes:   make(map[string]int64),
		ctx:           ctx,
		cancel:        cancel,
	}

	go store.cleanupExpiredSessions()

	return store
}

// evictOldestSession removes the least recently used session
func (s *SessionStore) evictOldestSession() {
	var oldestPhone string
	oldestTime := time.Now().Unix()

	for phone, accessTime := range s.accessTimes {
		if accessTime < oldestTime {
			oldestTime = accessTime
			oldestPhone = phone
		}
	}

	if oldestPhone != "" {
		delete(s.sessions, oldestPhone)
		delete(s.voiceSessions, oldestPhone)
		delete(s.smsSessions, oldestPhone)
		delete(s.accessTimes, oldestPhone)
	}
}

// GetSessionCount returns the current number of active sessions
func (s *SessionStore) GetSessionCount() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.sessions) + len(s.voiceSessions) + len(s.smsSessions)
}

func (s *SessionStore) SetDisambiguationSession(phoneNumber string, session *models.DisambiguationSession) error {
	if !phoneRegex.MatchString(phoneNumber) {
		return fmt.Errorf("invalid phone number format: %s", phoneNumber)
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check session limit and evict oldest if necessary
	if len(s.sessions) >= maxSessions {
		s.evictOldestSession()
	}

	now := time.Now().Unix()
	session.CreatedAt = now
	s.sessions[phoneNumber] = session
	s.accessTimes[phoneNumber] = now
	return nil
}

func (s *SessionStore) GetDisambiguationSession(phoneNumber string) *models.DisambiguationSession {
	if !phoneRegex.MatchString(phoneNumber) {
		return nil // Invalid phone number format
	}

	s.mutex.Lock() // Use write lock for simplicity and to update access time
	defer s.mutex.Unlock()

	session, exists := s.sessions[phoneNumber]
	if !exists {
		return nil
	}

	// Check expiry and clean up in single critical section
	if time.Now().Unix()-session.CreatedAt > sessionTimeoutMinutes*60 {
		delete(s.sessions, phoneNumber)
		delete(s.accessTimes, phoneNumber)
		return nil
	}

	// Update access time for LRU
	s.accessTimes[phoneNumber] = time.Now().Unix()

	return session
}

func (s *SessionStore) ClearDisambiguationSession(phoneNumber string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.sessions, phoneNumber)
	delete(s.accessTimes, phoneNumber)
}

func (s *SessionStore) SetVoiceSession(phoneNumber string, session *models.VoiceSession) error {
	if !phoneRegex.MatchString(phoneNumber) {
		return fmt.Errorf("invalid phone number format: %s", phoneNumber)
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check session limit and evict oldest if necessary
	if len(s.sessions)+len(s.voiceSessions)+len(s.smsSessions) >= maxSessions {
		s.evictOldestSession()
	}

	now := time.Now().Unix()
	session.CreatedAt = now
	s.voiceSessions[phoneNumber] = session
	s.accessTimes[phoneNumber] = now
	return nil
}

func (s *SessionStore) GetVoiceSession(phoneNumber string) *models.VoiceSession {
	if !phoneRegex.MatchString(phoneNumber) {
		return nil // Invalid phone number format
	}

	s.mutex.Lock() // Use write lock for simplicity and to update access time
	defer s.mutex.Unlock()

	session, exists := s.voiceSessions[phoneNumber]
	if !exists {
		return nil
	}

	// Check expiry and clean up in single critical section
	if time.Now().Unix()-session.CreatedAt > sessionTimeoutMinutes*60 {
		delete(s.voiceSessions, phoneNumber)
		delete(s.accessTimes, phoneNumber)
		return nil
	}

	// Update access time for LRU
	s.accessTimes[phoneNumber] = time.Now().Unix()

	return session
}

func (s *SessionStore) ClearVoiceSession(phoneNumber string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.voiceSessions, phoneNumber)
	delete(s.accessTimes, phoneNumber)
}

func (s *SessionStore) SetSMSSession(phoneNumber string, session *models.SMSSession) error {
	if !phoneRegex.MatchString(phoneNumber) {
		return fmt.Errorf("invalid phone number format: %s", phoneNumber)
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check session limit and evict oldest if necessary
	if len(s.sessions)+len(s.voiceSessions)+len(s.smsSessions) >= maxSessions {
		s.evictOldestSession()
	}

	now := time.Now().Unix()
	session.CreatedAt = now
	session.LastQueryTime = now
	s.smsSessions[phoneNumber] = session
	s.accessTimes[phoneNumber] = now
	return nil
}

func (s *SessionStore) GetSMSSession(phoneNumber string) *models.SMSSession {
	if !phoneRegex.MatchString(phoneNumber) {
		return nil // Invalid phone number format
	}

	s.mutex.Lock() // Use write lock for simplicity and to update access time
	defer s.mutex.Unlock()

	session, exists := s.smsSessions[phoneNumber]
	if !exists {
		return nil
	}

	// Check expiry and clean up in single critical section
	if time.Now().Unix()-session.CreatedAt > smsSessionTimeoutMinutes*60 {
		delete(s.smsSessions, phoneNumber)
		delete(s.accessTimes, phoneNumber)
		return nil
	}

	// Update access time for LRU
	s.accessTimes[phoneNumber] = time.Now().Unix()

	return session
}

func (s *SessionStore) ClearSMSSession(phoneNumber string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.smsSessions, phoneNumber)
	delete(s.accessTimes, phoneNumber)
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

			// Clean up disambiguation sessions
			for phoneNumber, session := range s.sessions {
				if now-session.CreatedAt > sessionTimeoutMinutes*60 {
					delete(s.sessions, phoneNumber)
					delete(s.accessTimes, phoneNumber)
				}
			}

			// Clean up voice sessions
			for phoneNumber, session := range s.voiceSessions {
				if now-session.CreatedAt > sessionTimeoutMinutes*60 {
					delete(s.voiceSessions, phoneNumber)
					delete(s.accessTimes, phoneNumber)
				}
			}

			// Clean up SMS sessions
			for phoneNumber, session := range s.smsSessions {
				if now-session.CreatedAt > smsSessionTimeoutMinutes*60 {
					delete(s.smsSessions, phoneNumber)
					delete(s.accessTimes, phoneNumber)
				}
			}
			s.mutex.Unlock()
		}
	}
}
