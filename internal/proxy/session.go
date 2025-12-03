// Package proxy provides core proxy session management
package proxy

import (
	"sync"
	"time"
)

// SessionState represents the current state of a proxy session
type SessionState string

const (
	// SessionStateInitializing indicates the session is starting up
	SessionStateInitializing SessionState = "initializing"
	// SessionStateActive indicates the session is running
	SessionStateActive SessionState = "active"
	// SessionStateShuttingDown indicates the session is shutting down
	SessionStateShuttingDown SessionState = "shutting_down"
	// SessionStateClosed indicates the session has been closed
	SessionStateClosed SessionState = "closed"
	// SessionStateError indicates the session encountered an error
	SessionStateError SessionState = "error"
)

// ProxySession represents a single proxy session
type ProxySession struct {
	// ID is a unique identifier for this session
	ID string

	// State is the current session state
	State SessionState

	// StartedAt is when the session was created
	StartedAt time.Time

	// ClosedAt is when the session was closed (if applicable)
	ClosedAt *time.Time

	// ServerBinary is the path to the local server binary (local mode)
	ServerBinary string

	// ServerArgs are the arguments passed to the server (local mode)
	ServerArgs []string

	// RemoteURL is the URL of the remote server (remote mode)
	RemoteURL string

	// MessageCount tracks the number of messages proxied
	MessageCount int64

	// ErrorCount tracks the number of errors encountered
	ErrorCount int64

	// LastActivity is the timestamp of the last message
	LastActivity time.Time

	// mu protects concurrent access to session fields
	mu sync.RWMutex
}

// NewProxySession creates a new proxy session
func NewProxySession(id string) *ProxySession {
	now := time.Now()
	return &ProxySession{
		ID:           id,
		State:        SessionStateInitializing,
		StartedAt:    now,
		LastActivity: now,
	}
}

// SetState updates the session state
func (s *ProxySession) SetState(state SessionState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.State = state
	if state == SessionStateClosed || state == SessionStateError {
		now := time.Now()
		s.ClosedAt = &now
	}
}

// GetState returns the current session state
func (s *ProxySession) GetState() SessionState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State
}

// IncrementMessageCount increments the message counter
func (s *ProxySession) IncrementMessageCount() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.MessageCount++
	s.LastActivity = time.Now()
}

// IncrementErrorCount increments the error counter
func (s *ProxySession) IncrementErrorCount() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ErrorCount++
}

// GetStats returns session statistics
func (s *ProxySession) GetStats() SessionStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := SessionStats{
		ID:           s.ID,
		State:        s.State,
		StartedAt:    s.StartedAt,
		MessageCount: s.MessageCount,
		ErrorCount:   s.ErrorCount,
		LastActivity: s.LastActivity,
	}

	if s.ClosedAt != nil {
		stats.ClosedAt = s.ClosedAt
		stats.Duration = s.ClosedAt.Sub(s.StartedAt)
	} else {
		now := time.Now()
		stats.Duration = now.Sub(s.StartedAt)
	}

	return stats
}

// SessionStats represents session statistics
type SessionStats struct {
	ID           string
	State        SessionState
	StartedAt    time.Time
	ClosedAt     *time.Time
	Duration     time.Duration
	MessageCount int64
	ErrorCount   int64
	LastActivity time.Time
}
