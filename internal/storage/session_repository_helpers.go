package storage

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// UpdateLastActivity updates just the last activity timestamp
func (r *SessionRepository) UpdateLastActivity(sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)

	// Update single field
	err := r.redis.client.HSet(r.redis.ctx, key, "last_activity", time.Now().Format(time.RFC3339)).Err()
	if err != nil {
		return fmt.Errorf("failed to update last activity: %w", err)
	}

	// Refresh TTL
	if err := r.redis.client.Expire(r.redis.ctx, key, r.ttl).Err(); err != nil {
		slog.Warn("failed to refresh TTL", "error", err)
	}

	return nil
}

// SaveCookies stores cookies as JSON string
func (r *SessionRepository) SaveCookies(sessionID string, cookies []Cookie) error {
	if len(cookies) == 0 {
		return nil
	}

	key := fmt.Sprintf("session:%s:cookies", sessionID)

	// Marshal to JSON
	data, err := json.Marshal(cookies)
	if err != nil {
		return fmt.Errorf("failed to marshal cookies: %w", err)
	}

	// Store as string with TTL
	if err := r.redis.client.Set(r.redis.ctx, key, data, r.ttl).Err(); err != nil {
		return fmt.Errorf("failed to save cookies: %w", err)
	}

	return nil
}

// GetCookies retrieves cookies
func (r *SessionRepository) GetCookies(sessionID string) ([]Cookie, error) {
	key := fmt.Sprintf("session:%s:cookies", sessionID)

	data, err := r.redis.client.Get(r.redis.ctx, key).Result()
	if err != nil {
		// Not an error if cookies don't exist
		return nil, nil
	}

	var cookies []Cookie
	if err := json.Unmarshal([]byte(data), &cookies); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cookies: %w", err)
	}

	return cookies, nil
}

// SaveLocalStorage stores localStorage as hash
func (r *SessionRepository) SaveLocalStorage(sessionID string, localStorage map[string]string) error {
	if len(localStorage) == 0 {
		return nil
	}

	key := fmt.Sprintf("session:%s:localStorage", sessionID)

	// Convert map[string]string to map[string]interface{} for HSET
	fields := make(map[string]interface{})
	for k, v := range localStorage {
		fields[k] = v
	}

	if err := r.redis.client.HSet(r.redis.ctx, key, fields).Err(); err != nil {
		return fmt.Errorf("failed to save localStorage: %w", err)
	}

	// Set TTL
	if err := r.redis.client.Expire(r.redis.ctx, key, r.ttl).Err(); err != nil {
		slog.Warn("failed to set TTL on localStorage", "error", err)
	}

	return nil
}

// GetLocalStorage retrieves localStorage
func (r *SessionRepository) GetLocalStorage(sessionID string) (map[string]string, error) {
	key := fmt.Sprintf("session:%s:localStorage", sessionID)

	data, err := r.redis.client.HGetAll(r.redis.ctx, key).Result()
	if err != nil {
		return nil, err
	}

	// Empty map means no localStorage
	if len(data) == 0 {
		return nil, nil
	}

	return data, nil
}

// SavePages stores pages as JSON string
func (r *SessionRepository) SavePages(sessionID string, pages []PageState) error {
	if len(pages) == 0 {
		return nil
	}

	key := fmt.Sprintf("session:%s:pages", sessionID)

	data, err := json.Marshal(pages)
	if err != nil {
		return fmt.Errorf("failed to marshal pages: %w", err)
	}

	if err := r.redis.client.Set(r.redis.ctx, key, data, r.ttl).Err(); err != nil {
		return fmt.Errorf("failed to save pages: %w", err)
	}

	return nil
}

// GetPages retrieves pages
func (r *SessionRepository) GetPages(sessionID string) ([]PageState, error) {
	key := fmt.Sprintf("session:%s:pages", sessionID)

	data, err := r.redis.client.Get(r.redis.ctx, key).Result()
	if err != nil {
		return nil, nil
	}

	var pages []PageState
	if err := json.Unmarshal([]byte(data), &pages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pages: %w", err)
	}

	return pages, nil
}

// GetSessionByName retrieves session ID by agent + name
func (r *SessionRepository) GetSessionByName(agentID, sessionName string) (string, error) {
	key := fmt.Sprintf("agent:%s:session_names", agentID)
	
	sessionID, err := r.redis.client.HGet(r.redis.ctx, key, sessionName).Result()
	if err != nil {
		return "", fmt.Errorf("session not found with name '%s': %w", sessionName, err)
	}
	
	return sessionID, nil
}

// CheckSessionNameExists checks if a session name is already taken by an agent
func (r *SessionRepository) CheckSessionNameExists(agentID, sessionName string) (bool, error) {
	key := fmt.Sprintf("agent:%s:session_names", agentID)
	
	exists, err := r.redis.client.HExists(r.redis.ctx, key, sessionName).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check session name: %w", err)
	}
	
	return exists, nil
}

// ReserveSessionName atomically reserves a session name for an agent
func (r *SessionRepository) ReserveSessionName(agentID, sessionName, sessionID string) error {
	key := fmt.Sprintf("agent:%s:session_names", agentID)
	
	// Check if name already exists
	exists, err := r.CheckSessionNameExists(agentID, sessionName)
	if err != nil {
		return err
	}
	
	if exists {
		return fmt.Errorf("session name '%s' already exists for agent '%s'", sessionName, agentID)
	}
	
	// Reserve the name
	if err := r.redis.client.HSet(r.redis.ctx, key, sessionName, sessionID).Err(); err != nil {
		return fmt.Errorf("failed to reserve session name: %w", err)
	}
	
	// Set TTL on the hash
	if err := r.redis.client.Expire(r.redis.ctx, key, r.ttl).Err(); err != nil {
		slog.Warn("failed to set TTL on session names", "error", err)
	}
	
	return nil
}

// ReleaseSessionName removes the name mapping when session is deleted
func (r *SessionRepository) ReleaseSessionName(agentID, sessionName string) error {
	if sessionName == "" || agentID == "" {
		return nil
	}
	
	key := fmt.Sprintf("agent:%s:session_names", agentID)
	
	return r.redis.client.HDel(r.redis.ctx, key, sessionName).Err()
}

// RenameSession updates the session name
func (r *SessionRepository) RenameSession(sessionID, agentID, oldName, newName string) error {
	// Check if new name is already taken
	exists, err := r.CheckSessionNameExists(agentID, newName)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("session name '%s' already exists", newName)
	}
	
	// Remove old name mapping
	if err := r.ReleaseSessionName(agentID, oldName); err != nil {
		slog.Warn("failed to release old session name", "error", err)
	}
	
	// Add new name mapping
	if err := r.ReserveSessionName(agentID, newName, sessionID); err != nil {
		return err
	}
	
	// Update session hash
	sessionKey := fmt.Sprintf("session:%s", sessionID)
	if err := r.redis.client.HSet(r.redis.ctx, sessionKey, "session_name", newName).Err(); err != nil {
		return fmt.Errorf("failed to update session name: %w", err)
	}
	
	slog.Info("session renamed", 
		"session_id", sessionID, 
		"old_name", oldName, 
		"new_name", newName)
	
	return nil
}

// CountAgentSessions returns the number of active sessions for an agent
func (r *SessionRepository) CountAgentSessions(agentID string) (int, error) {
	key := fmt.Sprintf("agent:%s:sessions", agentID)
	
	count, err := r.redis.client.SCard(r.redis.ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to count agent sessions: %w", err)
	}
	
	return int(count), nil
}

// ListAgentSessions returns all sessions for an agent with details
func (r *SessionRepository) ListAgentSessions(agentID string) ([]*SessionState, error) {
	// Get all session IDs for this agent
	key := fmt.Sprintf("agent:%s:sessions", agentID)
	sessionIDs, err := r.redis.client.SMembers(r.redis.ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list agent sessions: %w", err)
	}
	
	// Fetch each session
	sessions := make([]*SessionState, 0, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		state, err := r.GetSession(sessionID)
		if err != nil {
			slog.Warn("failed to load session", 
				"session_id", sessionID, 
				"error", err)
			continue
		}
		sessions = append(sessions, state)
	}
	
	return sessions, nil
}