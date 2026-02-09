package storage

import (
	"fmt"
	"time"
)

// SessionState represents persisted session data
type SessionState struct {
	SessionID    string            `json:"session_id"`
	SessionName  string            `json:"session_name"`
	AgentID      string            `json:"agent_id,omitempty"`
	ProcessPort  int               `json:"process_port"`
	ContextID    string            `json:"context_id"`
	CreatedAt    time.Time         `json:"created_at"`
	LastActivity time.Time         `json:"last_activity"`
	Status       string            `json:"status"`
	
	// Browser state
	Cookies      []Cookie          `json:"cookies,omitempty"`
	LocalStorage map[string]string `json:"local_storage,omitempty"`
	Pages        []PageState       `json:"pages,omitempty"`
}

// Cookie represents a browser cookie
type Cookie struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires"`  // Unix timestamp
	Secure   bool    `json:"secure"`
	HttpOnly bool    `json:"httpOnly"`
	SameSite string  `json:"sameSite"`
}

// PageState represents an open page
type PageState struct {
	PageID   string `json:"page_id"`
	URL      string `json:"url"`
	Title    string `json:"title,omitempty"`
}

//validation helper for named sessions
func (s *SessionState) Validate() error {
	if s.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	if s.AgentID == "" {
		return fmt.Errorf("agent_id is required for named sessions")
	}
	// Session name is optional - will be auto-generated if not provided
	return nil
}

// auto-generate name if not provided
func (s *SessionState) EnsureSessionName() {
	if s.SessionName == "" {
		// Generate name like: "session-2026-02-08-001"
		timestamp := s.CreatedAt.Format("2006-01-02")
		s.SessionName = fmt.Sprintf("session-%s-%s", timestamp, s.SessionID[:8])
	}
}