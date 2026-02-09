package api

import (
	"time"

	"github.com/dhruvsoni1802/browser-query-ai/internal/session"
)

// Request Types

// CreateSessionRequest for POST /sessions
type CreateSessionRequest struct {
	AgentID     string `json:"agent_id" validate:"required"`
	SessionName string `json:"session_name,omitempty"`
	// Optional: Allow client to specify port
	// If not provided, server/load balancer decides
	BrowserPort int `json:"browser_port,omitempty"`
}

// NavigateRequest for POST /sessions/{id}/navigate
type NavigateRequest struct {
	URL string `json:"url" validate:"required"`
}

// ExecuteJSRequest for POST /sessions/{id}/execute
type ExecuteJSRequest struct {
	PageID string `json:"page_id" validate:"required"`
	Script string `json:"script" validate:"required"`
}

// ScreenshotRequest for POST /sessions/{id}/screenshot
type ScreenshotRequest struct {
	PageID string `json:"page_id" validate:"required"`
	Format string `json:"format,omitempty"` // "png" or "jpeg", default "png"
}


// Response Types

// CreateSessionResponse returned when session is created
type CreateSessionResponse struct {
	SessionID string `json:"session_id"`
	SessionName string    `json:"session_name"`
	AgentID     string    `json:"agent_id"`
	ContextID string `json:"context_id"`
	CreatedAt time.Time `json:"created_at"`
}

// NavigateResponse returned after navigation
type NavigateResponse struct {
	SessionID string `json:"session_id"`
	PageID    string `json:"page_id"`
	URL       string `json:"url"`
}

// ExecuteJSResponse returned after JavaScript execution
type ExecuteJSResponse struct {
	SessionID string      `json:"session_id"`
	PageID    string      `json:"page_id"`
	Result    interface{} `json:"result"`
}

// ScreenshotResponse returned after screenshot capture
type ScreenshotResponse struct {
	SessionID  string `json:"session_id"`
	PageID     string `json:"page_id"`
	Screenshot string `json:"screenshot"` // base64 encoded PNG/JPEG
	Format     string `json:"format"`
	Size       int    `json:"size"` // Size in bytes (before encoding)
}

// GetPageContentResponse returned with page HTML
type GetPageContentResponse struct {
	SessionID string `json:"session_id"`
	PageID    string `json:"page_id"`
	Content   string `json:"content"`
	Length    int    `json:"length"` // Content length in bytes
}

// GetSessionResponse returned with session details
type GetSessionResponse struct {
	SessionID    string                `json:"session_id"`
	SessionName  string                `json:"session_name"`
	AgentID      string                `json:"agent_id"`
	ContextID    string                `json:"context_id"`
	PageIDs      []string              `json:"page_ids"`
	PageCount    int                   `json:"page_count"`
	CreatedAt    time.Time             `json:"created_at"`
	LastActivity time.Time             `json:"last_activity"`
	Status       session.SessionStatus `json:"status"`
}

// ListSessionsResponse returned with all sessions
type ListSessionsResponse struct {
	Sessions []SessionInfo `json:"sessions"`
	Count    int           `json:"count"`
}

// SessionInfo contains summary information about a session
type SessionInfo struct {
	SessionID    string                `json:"session_id"`
	SessionName  string                `json:"session_name"`
	AgentID      string                `json:"agent_id"`
	ContextID    string                `json:"context_id"`
	PageCount    int                   `json:"page_count"`
	CreatedAt    time.Time             `json:"created_at"`
	LastActivity time.Time             `json:"last_activity"`
	Status       session.SessionStatus `json:"status"`
}

// SuccessResponse for operations that just need success confirmation
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// Error Types

// ErrorResponse for all error cases
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error information
type ErrorDetail struct {
	Code    string `json:"code"`    // Machine-readable error code
	Message string `json:"message"` // Human-readable message
}

// ListAgentSessionsResponse
type ListAgentSessionsResponse struct {
	AgentID  string               `json:"agent_id"`
	Sessions []SessionSummary     `json:"sessions"`
	Count    int                  `json:"count"`
}

// SessionSummary contains summary information about a session
type SessionSummary struct {
	SessionID    string                `json:"session_id"`
	SessionName  string                `json:"session_name"`
	Status       session.SessionStatus `json:"status"`
	PageCount    int                   `json:"page_count"`
	CreatedAt    time.Time             `json:"created_at"`
	LastActivity time.Time             `json:"last_activity"`
}

// ResumeSessionRequest for POST /sessions/resume
type ResumeSessionRequest struct {
	AgentID     string `json:"agent_id" validate:"required"`
	SessionName string `json:"session_name" validate:"required"`
}

// ResumeSessionResponse for resuming a session
type ResumeSessionResponse struct {
	SessionID   string    `json:"session_id"`
	SessionName string    `json:"session_name"`
	Resumed     bool      `json:"resumed"`  // true if existed, false if created new
	CreatedAt   time.Time `json:"created_at"`
}

// RenameSessionRequest for PUT /sessions/{id}/rename
type RenameSessionRequest struct {
	SessionName string `json:"session_name" validate:"required"`
}


// Common error codes
const (
	ErrCodeSessionNotFound     = "SESSION_NOT_FOUND"
	ErrCodePageNotFound        = "PAGE_NOT_FOUND"
	ErrCodeInvalidRequest      = "INVALID_REQUEST"
	ErrCodeSessionCreateFailed = "SESSION_CREATE_FAILED"
	ErrCodeNavigationFailed    = "NAVIGATION_FAILED"
	ErrCodeExecutionFailed     = "EXECUTION_FAILED"
	ErrCodeScreenshotFailed    = "SCREENSHOT_FAILED"
	ErrCodeInternalError       = "INTERNAL_ERROR"
)