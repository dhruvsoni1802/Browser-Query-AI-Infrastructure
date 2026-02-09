package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dhruvsoni1802/browser-query-ai/internal/pool"
	"github.com/dhruvsoni1802/browser-query-ai/internal/session"
	"github.com/go-chi/chi/v5"
)

// Handlers contains HTTP handlers for the API
type Handlers struct {
	sessionManager *session.Manager
	loadBalancer   *pool.LoadBalancer
}

// NewHandlers creates a new Handlers instance
func NewHandlers(manager *session.Manager, loadBalancer *pool.LoadBalancer) *Handlers {
	return &Handlers{
		sessionManager: manager,
		loadBalancer:   loadBalancer,
	}
}

// CreateSession handles POST /sessions
func (h *Handlers) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Empty body is acceptable
		req = CreateSessionRequest{}
	}
	
	// Validate agent ID
	if req.AgentID == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "agent_id is required")
		return
	}
	
	// Select port (use provided or load balance)
	port := req.BrowserPort
	if port == 0 {
		process, err := h.loadBalancer.SelectProcess()
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, 
				ErrCodeInternalError, "No available browsers")
			return
		}
		port = process.GetPort()
	}
	
	// Create session with name
	sess, err := h.sessionManager.CreateSessionWithName(req.AgentID, req.SessionName, port)
	if err != nil {
		// Check for specific errors
		if err == session.ErrSessionNameConflict {
			writeError(w, http.StatusConflict, "SESSION_NAME_CONFLICT", 
				fmt.Sprintf("Session name '%s' already exists", req.SessionName))
			return
		}
		if err == session.ErrSessionLimitReached {
			writeError(w, http.StatusTooManyRequests, "SESSION_LIMIT_REACHED", err.Error())
			return
		}
		
		writeError(w, http.StatusInternalServerError, 
			ErrCodeSessionCreateFailed, err.Error())
		return
	}
	
	// Increment session count on process
	processes := h.loadBalancer.GetProcesses()
	for _, process := range processes {
		if process.GetPort() == port {
			process.IncrementSessionCount()
			break
		}
	}
	
	response := CreateSessionResponse{
		SessionID:   sess.ID,
		SessionName: sess.Name,
		AgentID:     sess.AgentID,
		ContextID:   sess.ContextID,
		CreatedAt:   sess.CreatedAt,
	}
	
	writeJSON(w, http.StatusCreated, response)
}

// DestroySession handles DELETE /sessions/{id}
func (h *Handlers) DestroySession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")

	// Get session first to know which process it's on
	sess, err := h.sessionManager.GetSession(sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, ErrCodeSessionNotFound, err.Error())
		return
	}

	// Destroy session
	if err := h.sessionManager.DestroySession(sessionID); err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	// Decrement session count on the process
	// Find the process by port
	processes := h.loadBalancer.GetProcesses()
	for _, process := range processes {
		if process.GetPort() == sess.ProcessPort {
			process.DecrementSessionCount()
			break
		}
	}

	// Return 204 No Content
	w.WriteHeader(http.StatusNoContent)
}

// GetSession handles GET /sessions/{id}
func (h *Handlers) GetSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")

	sess, err := h.sessionManager.GetSession(sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, ErrCodeSessionNotFound, err.Error())
		return
	}

	response := GetSessionResponse{
		SessionID:    sess.ID,
		SessionName:  sess.Name,
		AgentID:      sess.AgentID,
		ContextID:    sess.ContextID,
		PageIDs:      sess.PageIDs,
		PageCount:    len(sess.PageIDs),
		CreatedAt:    sess.CreatedAt,
		LastActivity: sess.LastActivity,
		Status:       sess.Status,
	}

	writeJSON(w, http.StatusOK, response)
}

// ListSessions handles GET /sessions
func (h *Handlers) ListSessions(w http.ResponseWriter, r *http.Request) {
	sessions := h.sessionManager.ListSessions()

	sessionInfos := make([]SessionInfo, 0, len(sessions))
	for _, sess := range sessions {
		sessionInfos = append(sessionInfos, SessionInfo{
			SessionID:    sess.ID,
			SessionName:  sess.Name,
			AgentID:      sess.AgentID,
			ContextID:    sess.ContextID,
			PageCount:    len(sess.PageIDs),
			CreatedAt:    sess.CreatedAt,
			LastActivity: sess.LastActivity,
			Status:       sess.Status,
		})
	}

	response := ListSessionsResponse{
		Sessions: sessionInfos,
		Count:    len(sessionInfos),
	}

	writeJSON(w, http.StatusOK, response)
}

// Navigate handles POST /sessions/{id}/navigate
func (h *Handlers) Navigate(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")

	var req NavigateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}

	if req.URL == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "URL is required")
		return
	}

	pageID, err := h.sessionManager.Navigate(sessionID, req.URL)
	if err != nil {
		if err.Error() == "failed to get session: session not found: "+sessionID {
			writeError(w, http.StatusNotFound, ErrCodeSessionNotFound, "Session not found")
		} else {
			writeError(w, http.StatusInternalServerError, ErrCodeNavigationFailed, err.Error())
		}
		return
	}

	response := NavigateResponse{
		SessionID: sessionID,
		PageID:    pageID,
		URL:       req.URL,
	}

	writeJSON(w, http.StatusOK, response)
}

// ExecuteJS handles POST /sessions/{id}/execute
func (h *Handlers) ExecuteJS(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")

	var req ExecuteJSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}

	if req.PageID == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "page_id is required")
		return
	}
	if req.Script == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "script is required")
		return
	}

	result, err := h.sessionManager.ExecuteJavascript(sessionID, req.PageID, req.Script)
	if err != nil {
		if err.Error() == "failed to get session: session not found: "+sessionID {
			writeError(w, http.StatusNotFound, ErrCodeSessionNotFound, "Session not found")
		} else if err.Error() == "page not found in session: "+req.PageID {
			writeError(w, http.StatusNotFound, ErrCodePageNotFound, "Page not found in session")
		} else {
			writeError(w, http.StatusInternalServerError, ErrCodeExecutionFailed, err.Error())
		}
		return
	}

	response := ExecuteJSResponse{
		SessionID: sessionID,
		PageID:    req.PageID,
		Result:    result,
	}

	writeJSON(w, http.StatusOK, response)
}

// CaptureScreenshot handles POST /sessions/{id}/screenshot
func (h *Handlers) CaptureScreenshot(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")

	var req ScreenshotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}

	if req.PageID == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "page_id is required")
		return
	}

	screenshotBytes, err := h.sessionManager.CaptureScreenshot(sessionID, req.PageID)
	if err != nil {
		if err.Error() == "failed to get session: session not found: "+sessionID {
			writeError(w, http.StatusNotFound, ErrCodeSessionNotFound, "Session not found")
		} else if err.Error() == "page not found in session: "+req.PageID {
			writeError(w, http.StatusNotFound, ErrCodePageNotFound, "Page not found in session")
		} else {
			writeError(w, http.StatusInternalServerError, ErrCodeScreenshotFailed, err.Error())
		}
		return
	}

	encoded := base64.StdEncoding.EncodeToString(screenshotBytes)

	format := req.Format
	if format == "" {
		format = "png"
	}

	response := ScreenshotResponse{
		SessionID:  sessionID,
		PageID:     req.PageID,
		Screenshot: encoded,
		Format:     format,
		Size:       len(screenshotBytes),
	}

	writeJSON(w, http.StatusOK, response)
}

// GetPageContent handles GET /sessions/{id}/pages/{pageId}/content
func (h *Handlers) GetPageContent(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	pageID := chi.URLParam(r, "pageId")

	content, err := h.sessionManager.GetPageContent(sessionID, pageID)
	if err != nil {
		if err.Error() == "failed to get session: session not found: "+sessionID {
			writeError(w, http.StatusNotFound, ErrCodeSessionNotFound, "Session not found")
		} else if err.Error() == "page not found in session: "+pageID {
			writeError(w, http.StatusNotFound, ErrCodePageNotFound, "Page not found in session")
		} else {
			writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		}
		return
	}

	response := GetPageContentResponse{
		SessionID: sessionID,
		PageID:    pageID,
		Content:   content,
		Length:    len(content),
	}

	writeJSON(w, http.StatusOK, response)
}

// ClosePage handles DELETE /sessions/{id}/pages/{pageId}
func (h *Handlers) ClosePage(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	pageID := chi.URLParam(r, "pageId")

	if err := h.sessionManager.ClosePage(sessionID, pageID); err != nil {
		if err.Error() == "failed to get session: session not found: "+sessionID {
			writeError(w, http.StatusNotFound, ErrCodeSessionNotFound, "Session not found")
		} else if err.Error() == "page not found in session: "+pageID {
			writeError(w, http.StatusNotFound, ErrCodePageNotFound, "Page not found in session")
		} else {
			writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListAgentSessions handles GET /agents/{agentId}/sessions
func (h *Handlers) ListAgentSessions(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentId")
	
	if agentID == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "agent_id is required")
		return
	}
	
	sessions, err := h.sessionManager.ListAgentSessions(agentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}
	
	// Convert to summary format
	summaries := make([]SessionSummary, len(sessions))
	for i, sess := range sessions {
		summaries[i] = SessionSummary{
			SessionID:    sess.ID,
			SessionName:  sess.Name,
			Status:       sess.Status,
			PageCount:    len(sess.PageIDs),
			CreatedAt:    sess.CreatedAt,
			LastActivity: sess.LastActivity,
		}
	}
	
	response := ListAgentSessionsResponse{
		AgentID:  agentID,
		Sessions: summaries,
		Count:    len(summaries),
	}
	
	writeJSON(w, http.StatusOK, response)
}

// ResumeSession handles POST /sessions/resume
func (h *Handlers) ResumeSession(w http.ResponseWriter, r *http.Request) {
	var req ResumeSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}
	
	if req.AgentID == "" || req.SessionName == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, 
			"agent_id and session_name are required")
		return
	}
	
	// Resume session by name
	sess, err := h.sessionManager.ResumeSessionByName(req.AgentID, req.SessionName)
	if err != nil {
		writeError(w, http.StatusNotFound, ErrCodeSessionNotFound, err.Error())
		return
	}
	
	response := ResumeSessionResponse{
		SessionID:   sess.ID,
		SessionName: sess.Name,
		Resumed:     true,
		CreatedAt:   sess.CreatedAt,
	}
	
	writeJSON(w, http.StatusOK, response)
}

// ResumeSessionByID handles POST /sessions/{id}/resume
func (h *Handlers) ResumeSessionByID(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	
	// Get session (will resurrect if needed)
	sess, err := h.sessionManager.GetSession(sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, ErrCodeSessionNotFound, err.Error())
		return
	}
	
	// Update activity
	sess.UpdateActivity()
	
	response := ResumeSessionResponse{
		SessionID:   sess.ID,
		SessionName: sess.Name,
		Resumed:     true,
		CreatedAt:   sess.CreatedAt,
	}
	
	writeJSON(w, http.StatusOK, response)
}

// RenameSession handles PUT /sessions/{id}/rename
func (h *Handlers) RenameSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	
	var req RenameSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}
	
	if req.SessionName == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "session_name is required")
		return
	}
	
	// Rename the session
	if err := h.sessionManager.RenameSession(sessionID, req.SessionName); err != nil {
		if err.Error() == fmt.Sprintf("session name '%s' already exists", req.SessionName) {
			writeError(w, http.StatusConflict, "SESSION_NAME_CONFLICT", err.Error())
			return
		}
		
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}
	
	// Return updated session info
	sess, _ := h.sessionManager.GetSession(sessionID)
	
	response := map[string]interface{}{
		"session_id":   sess.ID,
		"session_name": sess.Name,
		"agent_id":     sess.AgentID,
	}
	
	writeJSON(w, http.StatusOK, response)
}