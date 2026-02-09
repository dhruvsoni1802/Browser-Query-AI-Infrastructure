package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/dhruvsoni1802/browser-query-ai/internal/cdp"
	"github.com/dhruvsoni1802/browser-query-ai/internal/storage"
)

// Manager manages all active sessions and CDP connections
type Manager struct {
	sessions   map[string]*Session
	cdpClients map[int]*cdp.Client
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	repo       *storage.SessionRepository

	// Session limits
	maxSessionsPerAgent int 
	maxTotalSessions    int
}

// NewManager creates a new session manager
func NewManager(repo *storage.SessionRepository) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Manager{
		sessions:   make(map[string]*Session),
		cdpClients: make(map[int]*cdp.Client),
		ctx:        ctx,
		cancel:     cancel,
		repo:        repo,
		maxSessionsPerAgent: MaxSessionsPerAgent,
		maxTotalSessions: MaxTotalSessions,
	}
}

// generateSessionID creates a unique session identifier
func generateSessionID() (string, error) {
	// Generate 16 random bytes
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)

	// If there is an error, return an error
	if err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}

	// Encode the random bytes to base64
	sessionID := base64.URLEncoding.EncodeToString(randomBytes)

	// Return the session ID with prefix
	return "sess_" + sessionID, nil
}

// GetOrCreateCDPClient gets existing client or creates new one for a port
func (m *Manager) GetOrCreateCDPClient(port int) (*cdp.Client, error) {
	// Check if the client already exists for this port
	client, exists := m.cdpClients[port]
	if exists {
		return client, nil
	}

	// If the client does not exist, discover the WebSocket URL
	// TODO: Change this later so that we can use something other than localhost such as actual IP address of the machine
	wsURL, err := cdp.GetWebSocketURL("localhost", strconv.Itoa(port))
	if err != nil {
		return nil, fmt.Errorf("failed to discover WebSocket URL: %w", err)
	}

	// Create a new CDP client and connect to it
	client = cdp.NewClient(wsURL)
	if err := client.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to CDP client: %w", err)
	}

	// Add the client to the manager
	m.cdpClients[port] = client
	return client, nil
}

// CreateSession creates a new isolated browsing session
func (m *Manager) CreateSession(port int) (*Session, error) {
	// Acquire write lock to prevent concurrent access
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate a unique session ID
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	// Get or create a CDP client for the given port
	client, err := m.GetOrCreateCDPClient(port)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create CDP client: %w", err)
	}

	// Create a browser context on the browser process
	contextID, err := client.CreateBrowserContext()
	if err != nil {
		return nil, fmt.Errorf("failed to create browser context: %w", err)
	}

	// Create a new session struct
	session := &Session{
		ID:           sessionID,
		ProcessPort:  port,
		ContextID:    contextID,
		PageIDs:      []string{},
		CDPClient:    client,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		Status:       SessionActive,
	}

	// Add the session to the manager
	m.sessions[sessionID] = session

	// Return the session
	return session, nil
}

// GetSession retrieves a session by ID
func (m *Manager) GetSession(sessionID string) (*Session, error) {
	// Acquire read lock (allows multiple concurrent reads)
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Look up session in map
	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return session, nil
}

// DestroySession cleans up all resources for a session
func (m *Manager) DestroySession(sessionID string) error {
	// Acquire write lock (exclusive access)
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get session from map
	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Close all pages in this session
	for _, pageID := range session.PageIDs {
		if err := session.CDPClient.CloseTarget(pageID); err != nil {
			// Log error but continue cleanup
			fmt.Printf("warning: failed to close page %s: %v\n", pageID, err)
		}
	}

	// Dispose the browser context
	if err := session.CDPClient.DisposeBrowserContext(session.ContextID); err != nil {
		return fmt.Errorf("failed to dispose browser context: %w", err)
	}

	// Delete from Redis (this now handles name cleanup too)
	if m.repo != nil {
		if err := m.repo.DeleteSession(sessionID); err != nil {
			slog.Warn("failed to delete session from Redis", "error", err)
		}
	}

	// Mark session as closed
	session.Status = SessionClosed

	// Remove from map
	delete(m.sessions, sessionID)

	slog.Info("session destroyed", 
		"session_id", sessionID,
		"session_name", session.Name,
		"agent_id", session.AgentID)

	return nil
}

// ListSessions returns all active sessions
func (m *Manager) ListSessions() []*Session {
	// Acquire read lock
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create slice to hold sessions
	sessions := make([]*Session, 0, len(m.sessions))

	// Loop through sessions and append to slice
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}

// GetSessionCount returns the number of active sessions
func (m *Manager) GetSessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// Close closes all CDP connections and stops background workers
func (m *Manager) Close() error {
	// Signal cleanup worker to stop
	m.cancel()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Close all CDP clients
	for port, client := range m.cdpClients {
		if err := client.Close(); err != nil {
			slog.Warn("failed to close CDP client", "port", port, "error", err)
		}
	}

	// Clear maps
	m.sessions = make(map[string]*Session)
	m.cdpClients = make(map[int]*cdp.Client)

	return nil
}

// StartCleanupWorker starts a background worker to clean up expired sessions
func (m *Manager) StartCleanupWorker(interval, timeout time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		slog.Info("cleanup worker started", 
			"check_interval", interval, 
			"session_timeout", timeout)

		for {
			select {
			case <-m.ctx.Done():
				slog.Info("cleanup worker stopping")
				return

			case <-ticker.C:
				m.cleanupExpiredSessions(timeout)
			}
		}
	}()
}

// cleanupExpiredSessions removes sessions inactive for longer than timeout
func (m *Manager) cleanupExpiredSessions(timeout time.Duration) {
	// Phase 1: Collect expired session IDs (read lock)
	m.mu.RLock()
	expiredIDs := make([]string, 0)
	
	for sessionID, session := range m.sessions {
		if session.IsExpired(timeout) {
			expiredIDs = append(expiredIDs, sessionID)
		}
	}
	m.mu.RUnlock()

	// Phase 2: Destroy expired sessions (each acquires its own lock)
	if len(expiredIDs) > 0 {
		slog.Info("cleaning up expired sessions", 
			"count", len(expiredIDs),
			"timeout", timeout)
		
		for _, sessionID := range expiredIDs {
			if err := m.DestroySession(sessionID); err != nil {
				slog.Warn("failed to destroy expired session", 
					"session_id", sessionID, 
					"error", err)
			} else {
				slog.Debug("destroyed expired session", 
					"session_id", sessionID)
			}
		}
	}
}

// CreateSessionWithName creates a new session with optional name and agent ID
func (m *Manager) CreateSessionWithName(agentID, sessionName string, port int) (*Session, error) {
	// Validate agent ID is provided
	if agentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}
	
	// Check session limits
	if err := m.checkSessionLimits(agentID); err != nil {
		return nil, err
	}
	
	// If name provided, check for conflicts
	if sessionName != "" && m.repo != nil {
		exists, err := m.repo.CheckSessionNameExists(agentID, sessionName)
		if err != nil {
			return nil, fmt.Errorf("failed to check session name: %w", err)
		}
		if exists {
			return nil, ErrSessionNameConflict
		}
	}
	
	// Create the session (existing logic)
	m.mu.Lock()
	defer m.mu.Unlock()

	sessionID, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	client, err := m.GetOrCreateCDPClient(port)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create CDP client: %w", err)
	}

	contextID, err := client.CreateBrowserContext()
	if err != nil {
		return nil, fmt.Errorf("failed to create browser context: %w", err)
	}

	// Create session with name
	session := &Session{
		ID:           sessionID,
		Name:         sessionName,  // ← ADD (will be auto-generated if empty)
		AgentID:      agentID,      // ← ADD
		ProcessPort:  port,
		ContextID:    contextID,
		PageIDs:      []string{},
		CDPClient:    client,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		Status:       SessionActive,
	}

	// Auto-generate name if not provided
	if session.Name == "" {
		session.Name = m.generateSessionName(session)
	}

	// Add to manager
	m.sessions[sessionID] = session

	// Persist to Redis
	if m.repo != nil {
		state := m.sessionToState(session)
		if err := m.repo.SaveSession(state); err != nil {
			slog.Warn("failed to persist session to Redis", "error", err)
		}
	}

	slog.Info("session created", 
		"session_id", session.ID,
		"session_name", session.Name,
		"agent_id", agentID,
		"port", port)

	return session, nil
}

// Helper: Check if agent is within session limits
func (m *Manager) checkSessionLimits(agentID string) error {
	// Check total sessions
	m.mu.RLock()
	totalSessions := len(m.sessions)
	m.mu.RUnlock()
	
	if totalSessions >= m.maxTotalSessions {
		return fmt.Errorf("global session limit reached (%d)", m.maxTotalSessions)
	}
	
	// Check per-agent limit (from Redis)
	if m.repo != nil {
		count, err := m.repo.CountAgentSessions(agentID)
		if err != nil {
			slog.Warn("failed to count agent sessions", "error", err)
			// Don't block on Redis error
			return nil
		}
		
		if count >= m.maxSessionsPerAgent {
			return fmt.Errorf("%w: agent has %d sessions (max %d)", 
				ErrSessionLimitReached, count, m.maxSessionsPerAgent)
		}
	}
	
	return nil
}

// Helper: Auto-generate session name
func (m *Manager) generateSessionName(session *Session) string {
	timestamp := session.CreatedAt.Format("2006-01-02")
	shortID := session.ID[5:13] // Take 8 chars after "sess_"
	return fmt.Sprintf("%s-%s-%s", DefaultSessionNamePrefix, timestamp, shortID)
}

// Helper: Convert Session to SessionState for Redis
func (m *Manager) sessionToState(s *Session) *storage.SessionState {
	// Collect page states
	pages := make([]storage.PageState, len(s.PageIDs))
	for i, pageID := range s.PageIDs {
		pages[i] = storage.PageState{
			PageID: pageID,
			// URL and Title could be fetched if needed
		}
	}
	
	return &storage.SessionState{
		SessionID:    s.ID,
		SessionName:  s.Name,
		AgentID:      s.AgentID,
		ProcessPort:  s.ProcessPort,
		ContextID:    s.ContextID,
		CreatedAt:    s.CreatedAt,
		LastActivity: s.LastActivity,
		Status:       string(s.Status),
		Pages:        pages,
	}
}

// ResumeSessionByName resumes a session by agent ID and session name
func (m *Manager) ResumeSessionByName(agentID, sessionName string) (*Session, error) {
	if agentID == "" || sessionName == "" {
		return nil, fmt.Errorf("agent_id and session_name are required")
	}
	
	// Look up session ID by name
	if m.repo == nil {
		return nil, fmt.Errorf("Redis not configured")
	}
	
	sessionID, err := m.repo.GetSessionByName(agentID, sessionName)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}
	
	// Try to get from memory first
	m.mu.RLock()
	session, exists := m.sessions[sessionID]
	m.mu.RUnlock()
	
	if exists {
		// Session already in memory
		session.UpdateActivity()
		if m.repo != nil {
			m.repo.UpdateLastActivity(sessionID)
		}
		
		slog.Info("resumed session from memory", 
			"session_id", sessionID,
			"session_name", sessionName,
			"agent_id", agentID)
		
		return session, nil
	}
	
	// Session not in memory - resurrect from Redis
	state, err := m.repo.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load session from Redis: %w", err)
	}
	
	// Resurrect the session
	session, err = m.resurrectSession(state)
	if err != nil {
		return nil, fmt.Errorf("failed to resurrect session: %w", err)
	}
	
	slog.Info("resurrected session from Redis", 
		"session_id", sessionID,
		"session_name", sessionName,
		"agent_id", agentID)
	
	return session, nil
}

// resurrectSession rebuilds a session from Redis state
func (m *Manager) resurrectSession(state *storage.SessionState) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Get or create CDP client for the port
	client, err := m.GetOrCreateCDPClient(state.ProcessPort)
	if err != nil {
		return nil, fmt.Errorf("failed to reconnect to browser: %w", err)
	}
	
	// Recreate session object
	session := &Session{
		ID:           state.SessionID,
		Name:         state.SessionName,
		AgentID:      state.AgentID,
		ProcessPort:  state.ProcessPort,
		ContextID:    state.ContextID,
		PageIDs:      []string{},
		CDPClient:    client,
		CreatedAt:    state.CreatedAt,
		LastActivity: time.Now(),
		Status:       SessionStatus(state.Status),
	}
	
	// Restore pages
	for _, pageState := range state.Pages {
		session.PageIDs = append(session.PageIDs, pageState.PageID)
	}
	
	// Add to manager
	m.sessions[session.ID] = session
	
	// Update last activity in Redis
	if m.repo != nil {
		m.repo.UpdateLastActivity(session.ID)
	}
	
	return session, nil
}

// ListAgentSessions returns all sessions for an agent
func (m *Manager) ListAgentSessions(agentID string) ([]*Session, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}
	
	if m.repo == nil {
		// No Redis - return only in-memory sessions for this agent
		m.mu.RLock()
		defer m.mu.RUnlock()
		
		sessions := make([]*Session, 0)
		for _, session := range m.sessions {
			if session.AgentID == agentID {
				sessions = append(sessions, session)
			}
		}
		return sessions, nil
	}
	
	// Get from Redis
	states, err := m.repo.ListAgentSessions(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent sessions: %w", err)
	}
	
	// Convert to Session objects
	sessions := make([]*Session, 0, len(states))
	for _, state := range states {
		// Check if already in memory
		m.mu.RLock()
		session, exists := m.sessions[state.SessionID]
		m.mu.RUnlock()
		
		if exists {
			sessions = append(sessions, session)
		} else {
			// Create lightweight session object for listing
			// (don't fully resurrect unless explicitly resumed)
			session := &Session{
				ID:           state.SessionID,
				Name:         state.SessionName,
				AgentID:      state.AgentID,
				ProcessPort:  state.ProcessPort,
				ContextID:    state.ContextID,
				CreatedAt:    state.CreatedAt,
				LastActivity: state.LastActivity,
				Status:       SessionStatus(state.Status),
				PageIDs:      make([]string, len(state.Pages)),
			}
			for i, page := range state.Pages {
				session.PageIDs[i] = page.PageID
			}
			sessions = append(sessions, session)
		}
	}
	
	return sessions, nil
}

// RenameSession updates a session's name
func (m *Manager) RenameSession(sessionID, newName string) error {
	if sessionID == "" || newName == "" {
		return fmt.Errorf("session_id and new_name are required")
	}
	
	// Get session
	session, err := m.GetSession(sessionID)
	if err != nil {
		return err
	}
	
	if session.AgentID == "" {
		return fmt.Errorf("cannot rename session without agent_id")
	}
	
	oldName := session.Name
	
	// Update in Redis
	if m.repo != nil {
		if err := m.repo.RenameSession(sessionID, session.AgentID, oldName, newName); err != nil {
			return fmt.Errorf("failed to rename session in Redis: %w", err)
		}
	}
	
	// Update in memory
	session.Name = newName
	
	slog.Info("session renamed", 
		"session_id", sessionID,
		"old_name", oldName,
		"new_name", newName)
	
	return nil
}

// GetSessionByName is a convenience wrapper
func (m *Manager) GetSessionByName(agentID, sessionName string) (*Session, error) {
	return m.ResumeSessionByName(agentID, sessionName)
}