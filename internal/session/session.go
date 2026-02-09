package session

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dhruvsoni1802/browser-query-ai/internal/cdp"
)

// SessionStatus represents the current state of a session
type SessionStatus string

const (
	SessionActive  SessionStatus = "active"   // Session is running
	SessionClosed  SessionStatus = "closed"   // Session was explicitly closed
	SessionExpired SessionStatus = "expired"  // Session timed out
)

// Session represents an AI agent's isolated browsing session
type Session struct {
	ID           string          // Unique session identifier
	Name         string          // Session name
	AgentID      string          // Agent ID
	ProcessPort  int             // Which browser process (9222, 9223, etc.)
	ContextID    string          // CDP browser context ID
	PageIDs      []string        // List of page IDs in this context
	CDPClient    *cdp.Client     // WebSocket connection to browser
	CreatedAt    time.Time       // When session was created
	LastActivity time.Time       // Last time session was used
	Status       SessionStatus   // Current session status
}

// IsExpired checks if the session has been inactive too long
func (s *Session) IsExpired(timeout time.Duration) bool {
	return time.Since(s.LastActivity) > timeout
}

// UpdateActivity updates the last activity timestamp
func (s *Session) UpdateActivity() {
	s.LastActivity = time.Now()
}

// AddPage tracks a new page in this session
func (s *Session) AddPage(pageID string) {
	s.PageIDs = append(s.PageIDs, pageID)
	s.UpdateActivity()
}

// RemovePage removes a page from tracking
func (s *Session) RemovePage(pageID string) {
	for i, id := range s.PageIDs {
		if id == pageID {
			// Remove by swapping with last element and truncating
			s.PageIDs[i] = s.PageIDs[len(s.PageIDs)-1]
			s.PageIDs = s.PageIDs[:len(s.PageIDs)-1]
			break
		}
	}
	s.UpdateActivity()
}

// CaptureScreenshot takes a screenshot of the page
func (s *Session) CaptureScreenshot(targetID string) ([]byte, error) {
	params := map[string]interface{}{
		"format": "png",
	}

	result, err := s.CDPClient.SendCommandToTarget(targetID, "Page.captureScreenshot", params)
	if err != nil {
		return nil, fmt.Errorf("failed to capture screenshot: %w", err)
	}

	var response struct {
		Data string `json:"data"`
	}

	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("failed to parse screenshot response: %w", err)
	}

	imageBytes, err := base64.StdEncoding.DecodeString(response.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode screenshot: %w", err)
	}

	return imageBytes, nil
}

// ExecuteJavascript executes JavaScript code on the page
func (s *Session) ExecuteJavascript(targetID string, code string) (interface{}, error) {
	params := map[string]interface{}{
		"expression":    code,
		"returnByValue": true,
	}

	result, err := s.CDPClient.SendCommandToTarget(targetID, "Runtime.evaluate", params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute javascript: %w", err)
	}

	var response struct {
		Result struct {
			Type  string      `json:"type"`
			Value interface{} `json:"value"`
		} `json:"result"`
		ExceptionDetails interface{} `json:"exceptionDetails,omitempty"`
	}

	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("failed to parse execution result: %w", err)
	}

	if response.ExceptionDetails != nil {
		return nil, fmt.Errorf("javascript execution error: %v", response.ExceptionDetails)
	}

	return response.Result.Value, nil
}

// GetPageContent gets the HTML content of a page
func (s *Session) GetPageContent(targetID string) (string, error) {
	// Step 1: Get document
	result, err := s.CDPClient.SendCommandToTarget(targetID, "DOM.getDocument", nil)
	if err != nil {
		return "", fmt.Errorf("failed to get document: %w", err)
	}

	var docResponse struct {
		Root struct {
			NodeID int `json:"nodeId"`
		} `json:"root"`
	}

	if err := json.Unmarshal(result, &docResponse); err != nil {
		return "", fmt.Errorf("failed to parse document response: %w", err)
	}

	// Step 2: Get outer HTML
	params := map[string]interface{}{
		"nodeId": docResponse.Root.NodeID,
	}

	result, err = s.CDPClient.SendCommandToTarget(targetID, "DOM.getOuterHTML", params)
	if err != nil {
		return "", fmt.Errorf("failed to get outer HTML: %w", err)
	}

	var htmlResponse struct {
		OuterHTML string `json:"outerHTML"`
	}

	if err := json.Unmarshal(result, &htmlResponse); err != nil {
		return "", fmt.Errorf("failed to parse HTML response: %w", err)
	}

	return htmlResponse.OuterHTML, nil
}