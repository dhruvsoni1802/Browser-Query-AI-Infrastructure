package session

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dhruvsoni1802/browser-query-ai/internal/browser"
	"github.com/dhruvsoni1802/browser-query-ai/internal/config"
)

// Test helper: Setup browser and manager for operations tests
func setupTestManager(t *testing.T) (*browser.Process, *Manager, func()) {
	t.Helper()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)	
	}

	// Create and start browser
	proc, err := browser.NewProcess(cfg.ChromiumPath)
	if err != nil {
		t.Fatalf("failed to create browser process: %v", err)
	}

	if err := proc.Start(); err != nil {
		t.Fatalf("failed to start browser: %v", err)
	}

	// Wait for browser to be ready
	time.Sleep(2 * time.Second)

	// Create manager
	manager := NewManager(nil)

	// Cleanup function
	cleanup := func() {
		manager.Close()
		if err := proc.Stop(); err != nil {
			t.Errorf("failed to stop browser: %v", err)
		}
	}

	return proc, manager, cleanup
}

// TestNavigate tests navigation to a URL
func TestNavigate(t *testing.T) {
	proc, manager, cleanup := setupTestManager(t)
	defer cleanup()

	// Create session
	session, err := manager.CreateSession(proc.DebugPort)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Navigate to a URL
	pageID, err := manager.Navigate(session.ID, "https://example.com")
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}

	// Verify pageID is not empty
	if pageID == "" {
		t.Error("pageID is empty")
	}

	// Verify page is tracked in session
	if len(session.PageIDs) != 1 {
		t.Errorf("expected 1 page in session, got %d", len(session.PageIDs))
	}

	if session.PageIDs[0] != pageID {
		t.Errorf("expected pageID %s in session, got %s", pageID, session.PageIDs[0])
	}

	t.Logf("navigated to example.com, pageID: %s", pageID)
}

// TestNavigateMultiplePages tests opening multiple pages
func TestNavigateMultiplePages(t *testing.T) {
	proc, manager, cleanup := setupTestManager(t)
	defer cleanup()

	session, err := manager.CreateSession(proc.DebugPort)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Open multiple pages
	urls := []string{
		"https://example.com",
		"https://example.org",
		"https://example.net",
	}

	pageIDs := make([]string, len(urls))
	for i, url := range urls {
		pageID, err := manager.Navigate(session.ID, url)
		if err != nil {
			t.Fatalf("Navigate to %s failed: %v", url, err)
		}
		pageIDs[i] = pageID
		t.Logf("opened page %d: %s → %s", i+1, url, pageID)
	}

	// Verify all pages are tracked
	if len(session.PageIDs) != len(urls) {
		t.Errorf("expected %d pages, got %d", len(urls), len(session.PageIDs))
	}

	// Verify all pageIDs are unique
	uniqueIDs := make(map[string]bool)
	for _, pageID := range pageIDs {
		if uniqueIDs[pageID] {
			t.Errorf("duplicate pageID found: %s", pageID)
		}
		uniqueIDs[pageID] = true
	}
}

// TestNavigateInvalidSession tests navigation with invalid session
func TestNavigateInvalidSession(t *testing.T) {
	_, manager, cleanup := setupTestManager(t)
	defer cleanup()

	// Try to navigate with non-existent session
	_, err := manager.Navigate("invalid-session-id", "https://example.com")
	if err == nil {
		t.Error("expected error for invalid session, got nil")
	}

	if !strings.Contains(err.Error(), "session not found") {
		t.Errorf("expected 'session not found' error, got: %v", err)
	}
}

// TestCaptureScreenshot tests taking a screenshot
func TestCaptureScreenshot(t *testing.T) {
	proc, manager, cleanup := setupTestManager(t)
	defer cleanup()

	session, err := manager.CreateSession(proc.DebugPort)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Navigate to a page
	pageID, err := manager.Navigate(session.ID, "https://example.com")
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}

	// Wait for page to load
	time.Sleep(2 * time.Second)

	// Capture screenshot
	screenshot, err := manager.CaptureScreenshot(session.ID, pageID)
	if err != nil {
		t.Fatalf("CaptureScreenshot failed: %v", err)
	}

	// Verify screenshot is not empty
	if len(screenshot) == 0 {
		t.Error("screenshot is empty")
	}

	// Verify it's a valid PNG (starts with PNG magic bytes)
	if len(screenshot) < 8 {
		t.Error("screenshot too small to be valid PNG")
	}

	// PNG magic bytes: 137 80 78 71 13 10 26 10
	if screenshot[0] != 137 || screenshot[1] != 80 || screenshot[2] != 78 || screenshot[3] != 71 {
		t.Error("screenshot is not a valid PNG file")
	}

	t.Logf("captured screenshot: %d bytes", len(screenshot))

	// Optionally save for manual inspection
	// os.WriteFile("test_screenshot.png", screenshot, 0644)
}

// TestCaptureScreenshotInvalidPage tests screenshot with invalid page
func TestCaptureScreenshotInvalidPage(t *testing.T) {
	proc, manager, cleanup := setupTestManager(t)
	defer cleanup()

	session, err := manager.CreateSession(proc.DebugPort)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Try to screenshot non-existent page
	_, err = manager.CaptureScreenshot(session.ID, "invalid-page-id")
	if err == nil {
		t.Error("expected error for invalid page, got nil")
	}

	if !strings.Contains(err.Error(), "page not found") {
		t.Errorf("expected 'page not found' error, got: %v", err)
	}
}

// TestExecuteJavascript tests JavaScript execution
func TestExecuteJavascript(t *testing.T) {
	proc, manager, cleanup := setupTestManager(t)
	defer cleanup()

	session, err := manager.CreateSession(proc.DebugPort)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	pageID, err := manager.Navigate(session.ID, "https://example.com")
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}

	// Wait for page to load
	time.Sleep(2 * time.Second)

	// Test 1: Get page title
	result, err := manager.ExecuteJavascript(session.ID, pageID, "document.title")
	if err != nil {
		t.Fatalf("ExecuteJavascript failed: %v", err)
	}

	title, ok := result.(string)
	if !ok {
		t.Errorf("expected string result, got %T", result)
	}

	if title == "" {
		t.Error("page title is empty")
	}

	t.Logf("page title: %s", title)

	// Test 2: Simple arithmetic
	result, err = manager.ExecuteJavascript(session.ID, pageID, "2 + 2")
	if err != nil {
		t.Fatalf("ExecuteJavascript failed: %v", err)
	}

	// JavaScript numbers are float64 in Go
	sum, ok := result.(float64)
	if !ok {
		t.Errorf("expected float64 result, got %T", result)
	}

	if sum != 4.0 {
		t.Errorf("expected 4.0, got %f", sum)
	}

	// Test 3: Return object
	result, err = manager.ExecuteJavascript(session.ID, pageID, "({name: 'test', value: 42})")
	if err != nil {
		t.Fatalf("ExecuteJavascript failed: %v", err)
	}

	obj, ok := result.(map[string]interface{})
	if !ok {
		t.Errorf("expected object result, got %T", result)
	}

	if obj["name"] != "test" || obj["value"].(float64) != 42.0 {
		t.Errorf("unexpected object values: %v", obj)
	}

	t.Logf("executed JavaScript successfully")
}

// TestExecuteJavascriptInvalidPage tests JS execution with invalid page
func TestExecuteJavascriptInvalidPage(t *testing.T) {
	proc, manager, cleanup := setupTestManager(t)
	defer cleanup()

	session, err := manager.CreateSession(proc.DebugPort)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Try to execute on non-existent page
	_, err = manager.ExecuteJavascript(session.ID, "invalid-page-id", "2 + 2")
	if err == nil {
		t.Error("expected error for invalid page, got nil")
	}

	if !strings.Contains(err.Error(), "page not found") {
		t.Errorf("expected 'page not found' error, got: %v", err)
	}
}

// TestGetPageContent tests getting HTML content
func TestGetPageContent(t *testing.T) {
	proc, manager, cleanup := setupTestManager(t)
	defer cleanup()

	session, err := manager.CreateSession(proc.DebugPort)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	pageID, err := manager.Navigate(session.ID, "https://example.com")
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}

	// Wait for page to load
	time.Sleep(2 * time.Second)

	// Get page content
	content, err := manager.GetPageContent(session.ID, pageID)
	if err != nil {
		t.Fatalf("GetPageContent failed: %v", err)
	}

	// Verify content is not empty
	if len(content) == 0 {
		t.Error("page content is empty")
	}

	// Verify it's HTML (contains html tags)
	if !strings.Contains(strings.ToLower(content), "<html") {
		t.Error("content doesn't appear to be HTML")
	}

	if !strings.Contains(strings.ToLower(content), "example") {
		t.Error("content doesn't contain 'example' (expected from example.com)")
	}

	t.Logf("page content length: %d bytes", len(content))
}

// TestGetPageContentInvalidPage tests getting content from invalid page
func TestGetPageContentInvalidPage(t *testing.T) {
	proc, manager, cleanup := setupTestManager(t)
	defer cleanup()

	session, err := manager.CreateSession(proc.DebugPort)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Try to get content from non-existent page
	_, err = manager.GetPageContent(session.ID, "invalid-page-id")
	if err == nil {
		t.Error("expected error for invalid page, got nil")
	}

	if !strings.Contains(err.Error(), "page not found") {
		t.Errorf("expected 'page not found' error, got: %v", err)
	}
}

// TestClosePage tests closing a page
func TestClosePage(t *testing.T) {
	proc, manager, cleanup := setupTestManager(t)
	defer cleanup()

	session, err := manager.CreateSession(proc.DebugPort)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Open two pages
	pageID1, err := manager.Navigate(session.ID, "https://example.com")
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}

	pageID2, err := manager.Navigate(session.ID, "https://example.org")
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}

	// Verify both pages are tracked
	if len(session.PageIDs) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(session.PageIDs))
	}

	// Close first page
	if err := manager.ClosePage(session.ID, pageID1); err != nil {
		t.Fatalf("ClosePage failed: %v", err)
	}

	// Verify only one page remains
	if len(session.PageIDs) != 1 {
		t.Errorf("expected 1 page after close, got %d", len(session.PageIDs))
	}

	// Verify the remaining page is pageID2
	if session.PageIDs[0] != pageID2 {
		t.Errorf("expected remaining page to be %s, got %s", pageID2, session.PageIDs[0])
	}

	t.Log("page closed successfully")
}

// TestClosePageInvalidPage tests closing invalid page
func TestClosePageInvalidPage(t *testing.T) {
	proc, manager, cleanup := setupTestManager(t)
	defer cleanup()

	session, err := manager.CreateSession(proc.DebugPort)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Try to close non-existent page
	err = manager.ClosePage(session.ID, "invalid-page-id")
	if err == nil {
		t.Error("expected error for invalid page, got nil")
	}

	if !strings.Contains(err.Error(), "page not found") {
		t.Errorf("expected 'page not found' error, got: %v", err)
	}
}

// TestCompleteWorkflow tests a complete workflow
func TestCompleteWorkflow(t *testing.T) {
	proc, manager, cleanup := setupTestManager(t)
	defer cleanup()

	// Create session
	session, err := manager.CreateSession(proc.DebugPort)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	t.Logf("created session: %s", session.ID)

	// Navigate to page
	pageID, err := manager.Navigate(session.ID, "https://example.com")
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}

	t.Logf("navigated to example.com, pageID: %s", pageID)

	// Wait for page load
	time.Sleep(2 * time.Second)

	// Get title via JavaScript
	title, err := manager.ExecuteJavascript(session.ID, pageID, "document.title")
	if err != nil {
		t.Fatalf("ExecuteJavascript failed: %v", err)
	}

	t.Logf("page title: %v", title)

	// Get page content
	content, err := manager.GetPageContent(session.ID, pageID)
	if err != nil {
		t.Fatalf("GetPageContent failed: %v", err)
	}

	t.Logf("page content: %d bytes", len(content))

	// Take screenshot
	screenshot, err := manager.CaptureScreenshot(session.ID, pageID)
	if err != nil {
		t.Fatalf("CaptureScreenshot failed: %v", err)
	}

	t.Logf("screenshot: %d bytes", len(screenshot))

	// Save screenshot (optional)
	os.WriteFile("test_complete_workflow.png", screenshot, 0644)

	// Close page
	if err := manager.ClosePage(session.ID, pageID); err != nil {
		t.Fatalf("ClosePage failed: %v", err)
	}

	// Verify page is closed
	if len(session.PageIDs) != 0 {
		t.Errorf("expected 0 pages after close, got %d", len(session.PageIDs))
	}

	// Destroy session
	if err := manager.DestroySession(session.ID); err != nil {
		t.Fatalf("DestroySession failed: %v", err)
	}

	t.Log("complete workflow successful")
}

// TestActivityTracking tests that operations update LastActivity
func TestActivityTracking(t *testing.T) {
	proc, manager, cleanup := setupTestManager(t)
	defer cleanup()

	session, err := manager.CreateSession(proc.DebugPort)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	initialActivity := session.LastActivity

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Navigate (should update activity via AddPage)
	pageID, err := manager.Navigate(session.ID, "https://example.com")
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}

	if !session.LastActivity.After(initialActivity) {
		t.Error("Navigate did not update LastActivity")
	}

	activity1 := session.LastActivity
	time.Sleep(100 * time.Millisecond)

	// Screenshot (should update activity)
	time.Sleep(2 * time.Second) // Let page load
	_, err = manager.CaptureScreenshot(session.ID, pageID)
	if err != nil {
		t.Fatalf("CaptureScreenshot failed: %v", err)
	}

	if !session.LastActivity.After(activity1) {
		t.Error("CaptureScreenshot did not update LastActivity")
	}

	activity2 := session.LastActivity
	time.Sleep(100 * time.Millisecond)

	// ExecuteJS (should update activity)
	_, err = manager.ExecuteJavascript(session.ID, pageID, "2 + 2")
	if err != nil {
		t.Fatalf("ExecuteJavascript failed: %v", err)
	}

	if !session.LastActivity.After(activity2) {
		t.Error("ExecuteJavascript did not update LastActivity")
	}

	t.Log("activity tracking works correctly")
}