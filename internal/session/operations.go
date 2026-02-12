package session

import (
	"fmt"
	"slices"
)

// Navigate navigates to a URL and creates a new page in the session
func (m *Manager) Navigate(sessionID string, url string) (string, error) {
	// Get the session from the manager
	session, err := m.GetSession(sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to get session: %w", err)
	}

	// Create a new target/page in this session's context
	pageID, err := session.CDPClient.CreateTarget(url, session.ContextID)
	if err != nil {
		return "", fmt.Errorf("failed to create target: %w", err)
	}

	// Add the page ID to the session
	session.AddPage(pageID)

	// Return the page ID
	return pageID, nil
}

// CaptureScreenshot captures a screenshot of a given page
func (m *Manager) CaptureScreenshot(sessionID string, pageID string) ([]byte, error) {
	// Get the session from the manager
	session, err := m.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Verify that the page ID is in the session
	if !slices.Contains(session.PageIDs, pageID) {
		return nil, fmt.Errorf("page not found in session: %s", pageID)
	}

	// Capture screenshot of the page
	screenshot, err := session.CaptureScreenshot(pageID)
	if err != nil {
		return nil, fmt.Errorf("failed to capture screenshot: %w", err)
	}

	// Update the last activity time of the session
	session.UpdateActivity()

	// Return the screenshot
	return screenshot, nil
}

// ExecuteJavascript executes JavaScript code on a page
func (m *Manager) ExecuteJavascript(sessionID string, pageID string, code string) (interface{}, error) {
	// Get the session from the manager
	session, err := m.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Verify that the page ID is in the session
	if !slices.Contains(session.PageIDs, pageID) {
		return nil, fmt.Errorf("page not found in session: %s", pageID)
	}

	// Execute the JavaScript code on the page
	result, err := session.ExecuteJavascript(pageID, code)
	if err != nil {
		return nil, fmt.Errorf("failed to execute javascript: %w", err)
	}

	// Update the last activity time of the session
	session.UpdateActivity()

	// Return the result
	return result, nil
}

// GetPageContent gets the HTML content of a page
func (m *Manager) GetPageContent(sessionID string, pageID string) (string, error) {
	// Get the session from the manager
	session, err := m.GetSession(sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to get session: %w", err)
	}

	// Verify that the page ID is in the session
	if !slices.Contains(session.PageIDs, pageID) {
		return "", fmt.Errorf("page not found in session: %s", pageID)
	}

	// Get the HTML content of the page
	content, err := session.GetPageContent(pageID)
	if err != nil {
		return "", fmt.Errorf("failed to get page content: %w", err)
	}

	// Update the last activity time of the session
	session.UpdateActivity()

	// Return the content
	return content, nil
}

// AnalyzePage extracts the structural overview of a page
func (m *Manager) AnalyzePage(sessionID string, pageID string) (*PageStructure, error) {
	// Get the session from the manager
	session, err := m.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Verify that the page ID is in the session
	if !slices.Contains(session.PageIDs, pageID) {
		return nil, fmt.Errorf("page not found in session: %s", pageID)
	}

	// Analyze the page structure
	structure, err := session.AnalyzePage(pageID)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze page: %w", err)
	}

	// Update the last activity time of the session
	session.UpdateActivity()

	// Return the structure
	return structure, nil
}

// InvalidatePageAnalysis clears the cached analysis for a specific page in a session
func (m *Manager) InvalidatePageAnalysis(sessionID string, pageID string) error {
	// Get the session from the manager
	session, err := m.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Clear the cache for this page
	session.InvalidatePageAnalysis(pageID)

	return nil
}

// GetAccessibilityTree retrieves the accessibility tree for a page
func (m *Manager) GetAccessibilityTree(sessionID string, pageID string) (*AccessibilityTree, error) {
	// Get the session from the manager
	session, err := m.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Verify that the page ID is in the session
	if !slices.Contains(session.PageIDs, pageID) {
		return nil, fmt.Errorf("page not found in session: %s", pageID)
	}

	// Get the accessibility tree
	tree, err := session.GetAccessibilityTree(pageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get accessibility tree: %w", err)
	}

	// Update the last activity time of the session
	session.UpdateActivity()

	// Return the tree
	return tree, nil
}

// ClosePage closes a specific page in the session
func (m *Manager) ClosePage(sessionID string, pageID string) error {
	// Get the session from the manager
	session, err := m.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Verify that the page ID is in the session
	if !slices.Contains(session.PageIDs, pageID) {
		return fmt.Errorf("page not found in session: %s", pageID)
	}

	// Close the page via CDP
	if err := session.CDPClient.CloseTarget(pageID); err != nil {
		return fmt.Errorf("failed to close page: %w", err)
	}

	// Remove the page from the session tracking
	session.RemovePage(pageID)

	// Note: We DO update activity via RemovePage (it calls UpdateActivity)
	// Note: We do NOT dispose context - other pages might still be open

	return nil
}