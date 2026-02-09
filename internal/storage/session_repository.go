package storage

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"
)

// This struct handles session persistence in Redis
type SessionRepository struct {
	redis *RedisClient  // The Redis client to use for persistence
	ttl   time.Duration // Default TTL for sessions
}


// NewSessionRepository creates a new session repository
func NewSessionRepository(redisClient *RedisClient, ttl time.Duration) *SessionRepository {
	return &SessionRepository{
		redis: redisClient,
		ttl:   ttl,
	}
}

// SaveSession persists session state to Redis using Hash
func (r *SessionRepository) SaveSession(state *SessionState) error {
	key := fmt.Sprintf("session:%s", state.SessionID)

	// Build hash fields (basic metadata)
	fields := map[string]interface{}{
		"session_id":    state.SessionID,
		"agent_id":      state.AgentID,
		"process_port":  state.ProcessPort,
		"context_id":    state.ContextID,
		"created_at":    state.CreatedAt.Format(time.RFC3339),
		"last_activity": state.LastActivity.Format(time.RFC3339),
		"status":        state.Status,
	}

	// Store hash in Redis
	if err := r.redis.client.HSet(r.redis.ctx, key, fields).Err(); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	// Set expiration (TTL)
	if err := r.redis.client.Expire(r.redis.ctx, key, r.ttl).Err(); err != nil {
		return fmt.Errorf("failed to set TTL: %w", err)
	}

	// Add to active sessions set
	if err := r.redis.client.SAdd(r.redis.ctx, "active:sessions", state.SessionID).Err(); err != nil {
		slog.Warn("failed to add to active sessions set", "error", err)
	}

	// If agent ID provided, track this session for the agent
	if state.AgentID != "" {
		agentKey := fmt.Sprintf("agent:%s:sessions", state.AgentID)
		if err := r.redis.client.SAdd(r.redis.ctx, agentKey, state.SessionID).Err(); err != nil {
			slog.Warn("failed to add session to agent set", "error", err)
		}

		// Reserve the session name
		if err := r.ReserveSessionName(state.AgentID, state.SessionName, state.SessionID); err != nil {
			slog.Warn("failed to reserve session name", "error", err)
		}
	}

	// Save cookies, localStorage, pages separately
	if len(state.Cookies) > 0 {
		if err := r.SaveCookies(state.SessionID, state.Cookies); err != nil {
			slog.Warn("failed to save cookies", "error", err)
		}
	}

	if len(state.LocalStorage) > 0 {
		if err := r.SaveLocalStorage(state.SessionID, state.LocalStorage); err != nil {
			slog.Warn("failed to save localStorage", "error", err)
		}
	}

	if len(state.Pages) > 0 {
		if err := r.SavePages(state.SessionID, state.Pages); err != nil {
			slog.Warn("failed to save pages", "error", err)
		}
	}

	slog.Debug("session saved to Redis", "session_id", state.SessionID)
	return nil
}

// GetSession retrieves session state from Redis
func (r *SessionRepository) GetSession(sessionID string) (*SessionState, error) {
	key := fmt.Sprintf("session:%s", sessionID)

	// Get all hash fields
	data, err := r.redis.client.HGetAll(r.redis.ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Check if session exists (empty map means not found)
	if len(data) == 0 {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Parse fields
	state := &SessionState{
		SessionID:    data["session_id"],
		AgentID:      data["agent_id"],
		ContextID:    data["context_id"],
		Status:       data["status"],
	}

	// Parse port
	if port, err := strconv.Atoi(data["process_port"]); err == nil {
		state.ProcessPort = port
	}

	// Parse timestamps
	if createdAt, err := time.Parse(time.RFC3339, data["created_at"]); err == nil {
		state.CreatedAt = createdAt
	}
	if lastActivity, err := time.Parse(time.RFC3339, data["last_activity"]); err == nil {
		state.LastActivity = lastActivity
	}

	// Load cookies, localStorage, pages
	if cookies, err := r.GetCookies(sessionID); err == nil {
		state.Cookies = cookies
	}

	if localStorage, err := r.GetLocalStorage(sessionID); err == nil {
		state.LocalStorage = localStorage
	}

	if pages, err := r.GetPages(sessionID); err == nil {
		state.Pages = pages
	}

	return state, nil
}

// ListActiveSessions returns all active session IDs
func (r *SessionRepository) ListActiveSessions() ([]string, error) {
	sessions, err := r.redis.client.SMembers(r.redis.ctx, "active:sessions").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list active sessions: %w", err)
	}
	return sessions, nil
}

// DeleteSession removes session from Redis
func (r *SessionRepository) DeleteSession(sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)

	// First, get session to know agent_id and session_name
	data, err := r.redis.client.HGetAll(r.redis.ctx, key).Result()
	if err == nil && len(data) > 0 {
		agentID := data["agent_id"]
		sessionName := data["session_name"]
		
		// Release the session name
		if agentID != "" && sessionName != "" {
			if err := r.ReleaseSessionName(agentID, sessionName); err != nil {
				slog.Warn("failed to release session name", "error", err)
			}
		}
		
		// Remove from agent's sessions set
		if agentID != "" {
			agentKey := fmt.Sprintf("agent:%s:sessions", agentID)
			r.redis.client.SRem(r.redis.ctx, agentKey, sessionID)
		}
	}

	// Delete main session hash
	if err := r.redis.client.Del(r.redis.ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Delete associated data
	r.redis.client.Del(r.redis.ctx, fmt.Sprintf("session:%s:cookies", sessionID))
	r.redis.client.Del(r.redis.ctx, fmt.Sprintf("session:%s:localStorage", sessionID))
	r.redis.client.Del(r.redis.ctx, fmt.Sprintf("session:%s:pages", sessionID))

	// Remove from active sessions set
	r.redis.client.SRem(r.redis.ctx, "active:sessions", sessionID)

	slog.Debug("session deleted from Redis", "session_id", sessionID)
	return nil
}
