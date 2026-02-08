package api

import (
	"encoding/base64"
	"encoding/json"
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
	// Use load balancer to select best process
	process, err := h.loadBalancer.SelectProcess()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, ErrCodeInternalError, "No available browser processes: "+err.Error())
		return
	}

	port := process.GetPort()

	// Create session via session manager
	sess, err := h.sessionManager.CreateSession(port)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeSessionCreateFailed, err.Error())
		return
	}

	// Increment session count on the selected process
	process.IncrementSessionCount()

	// Build response
	response := CreateSessionResponse{
		SessionID: sess.ID,
		ContextID: sess.ContextID,
		CreatedAt: sess.CreatedAt,
	}

	// Return 201 Created
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