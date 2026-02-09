package session

import (
	"testing"
	"time"

	"github.com/dhruvsoni1802/browser-query-ai/internal/browser"
	"github.com/dhruvsoni1802/browser-query-ai/internal/config"
)

// Test helper: Setup browser process for tests
func setupTestBrowser(t *testing.T) (*browser.Process, func()) {
	t.Helper() // Marks this as a helper function

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

	// Return cleanup function
	cleanup := func() {
		if err := proc.Stop(); err != nil {
			t.Errorf("failed to stop browser: %v", err)
		}
	}

	return proc, cleanup
}

// TestNewManager tests manager creation
func TestNewManager(t *testing.T) {
	manager := NewManager(nil)

	// Check that manager is properly initialized
	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if manager.sessions == nil {
		t.Error("sessions map is nil")
	}

	if manager.cdpClients == nil {
		t.Error("cdpClients map is nil")
	}

	// Check initial state
	if len(manager.sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(manager.sessions))
	}

	if len(manager.cdpClients) != 0 {
		t.Errorf("expected 0 CDP clients, got %d", len(manager.cdpClients))
	}
}

// TestGenerateSessionID tests session ID generation
func TestGenerateSessionID(t *testing.T) {
	// Generate multiple IDs
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id, err := generateSessionID()
		if err != nil {
			t.Fatalf("generateSessionID failed: %v", err)
		}

		// Check format
		if len(id) < 10 {
			t.Errorf("session ID too short: %s", id)
		}

		if id[:5] != "sess_" {
			t.Errorf("session ID missing prefix: %s", id)
		}

		// Check uniqueness
		if ids[id] {
			t.Errorf("duplicate session ID generated: %s", id)
		}
		ids[id] = true
	}

	t.Logf("generated %d unique session IDs", len(ids))
}

// TestCreateSession tests session creation
func TestCreateSession(t *testing.T) {
	// Setup browser
	proc, cleanup := setupTestBrowser(t)
	defer cleanup()

	// Create manager
	manager := NewManager(nil)
	defer manager.Close()

	// Create session
	session, err := manager.CreateSession(proc.DebugPort)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Verify session properties
	if session.ID == "" {
		t.Error("session ID is empty")
	}

	if session.ProcessPort != proc.DebugPort {
		t.Errorf("expected port %d, got %d", proc.DebugPort, session.ProcessPort)
	}

	if session.ContextID == "" {
		t.Error("context ID is empty")
	}

	if session.CDPClient == nil {
		t.Error("CDP client is nil")
	}

	if session.Status != SessionActive {
		t.Errorf("expected status %s, got %s", SessionActive, session.Status)
	}

	if len(session.PageIDs) != 0 {
		t.Errorf("expected 0 pages, got %d", len(session.PageIDs))
	}

	// Verify session is in manager
	if manager.GetSessionCount() != 1 {
		t.Errorf("expected 1 session in manager, got %d", manager.GetSessionCount())
	}

	t.Logf("created session: %s with context: %s", session.ID, session.ContextID)
}

// TestGetSession tests session retrieval
func TestGetSession(t *testing.T) {
	proc, cleanup := setupTestBrowser(t)
	defer cleanup()

	manager := NewManager(nil)
	defer manager.Close()

	// Create session
	created, err := manager.CreateSession(proc.DebugPort)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Get session
	retrieved, err := manager.GetSession(created.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	// Verify it's the same session
	if retrieved.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, retrieved.ID)
	}

	if retrieved.ContextID != created.ContextID {
		t.Errorf("expected context %s, got %s", created.ContextID, retrieved.ContextID)
	}

	// Test getting non-existent session
	_, err = manager.GetSession("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent session, got nil")
	}
}

// TestDestroySession tests session cleanup
func TestDestroySession(t *testing.T) {
	proc, cleanup := setupTestBrowser(t)
	defer cleanup()

	manager := NewManager(nil)
	defer manager.Close()

	// Create session
	session, err := manager.CreateSession(proc.DebugPort)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	sessionID := session.ID

	// Verify session exists
	if manager.GetSessionCount() != 1 {
		t.Fatalf("expected 1 session, got %d", manager.GetSessionCount())
	}

	// Destroy session
	if err := manager.DestroySession(sessionID); err != nil {
		t.Fatalf("DestroySession failed: %v", err)
	}

	// Verify session is gone
	if manager.GetSessionCount() != 0 {
		t.Errorf("expected 0 sessions after destroy, got %d", manager.GetSessionCount())
	}

	// Verify getting destroyed session fails
	_, err = manager.GetSession(sessionID)
	if err == nil {
		t.Error("expected error when getting destroyed session, got nil")
	}

	// Test destroying non-existent session
	err = manager.DestroySession("nonexistent")
	if err == nil {
		t.Error("expected error when destroying non-existent session, got nil")
	}
}

// TestListSessions tests listing all sessions
func TestListSessions(t *testing.T) {
	proc, cleanup := setupTestBrowser(t)
	defer cleanup()

	manager := NewManager(nil)
	defer manager.Close()

	// Initially should be empty
	sessions := manager.ListSessions()
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}

	// Create multiple sessions
	created := make([]*Session, 3)
	for i := 0; i < 3; i++ {
		sess, err := manager.CreateSession(proc.DebugPort)
		if err != nil {
			t.Fatalf("CreateSession %d failed: %v", i, err)
		}
		created[i] = sess
	}

	// List sessions
	sessions = manager.ListSessions()
	if len(sessions) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(sessions))
	}

	// Verify all created sessions are in the list
	sessionMap := make(map[string]bool)
	for _, sess := range sessions {
		sessionMap[sess.ID] = true
	}

	for i, sess := range created {
		if !sessionMap[sess.ID] {
			t.Errorf("created session %d not in list: %s", i, sess.ID)
		}
	}
}

// TestCDPClientPooling tests that CDP clients are reused
func TestCDPClientPooling(t *testing.T) {
	proc, cleanup := setupTestBrowser(t)
	defer cleanup()

	manager := NewManager(nil)
	defer manager.Close()

	// Create first session
	sess1, err := manager.CreateSession(proc.DebugPort)
	if err != nil {
		t.Fatalf("CreateSession 1 failed: %v", err)
	}

	// Create second session on same port
	sess2, err := manager.CreateSession(proc.DebugPort)
	if err != nil {
		t.Fatalf("CreateSession 2 failed: %v", err)
	}

	// Both should use the same CDP client (connection pooling)
	if sess1.CDPClient != sess2.CDPClient {
		t.Error("expected sessions to share CDP client, but they don't")
	}

	// Verify only one CDP client exists
	manager.mu.RLock()
	clientCount := len(manager.cdpClients)
	manager.mu.RUnlock()

	if clientCount != 1 {
		t.Errorf("expected 1 CDP client, got %d", clientCount)
	}

	t.Log("connection pooling verified: both sessions share same CDP client")
}

// TestConcurrentSessionCreation tests thread safety
func TestConcurrentSessionCreation(t *testing.T) {
	proc, cleanup := setupTestBrowser(t)
	defer cleanup()

	manager := NewManager(nil)
	defer manager.Close()

	// Create sessions concurrently
	concurrency := 10
	done := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(n int) {
			sess, err := manager.CreateSession(proc.DebugPort)
			if err != nil {
				done <- err
				return
			}
			t.Logf("goroutine %d created session: %s", n, sess.ID)
			done <- nil
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < concurrency; i++ {
		if err := <-done; err != nil {
			t.Errorf("concurrent creation failed: %v", err)
		}
	}

	// Verify all sessions were created
	if manager.GetSessionCount() != concurrency {
		t.Errorf("expected %d sessions, got %d", concurrency, manager.GetSessionCount())
	}

	// Verify all session IDs are unique
	sessions := manager.ListSessions()
	sessionIDs := make(map[string]bool)
	for _, sess := range sessions {
		if sessionIDs[sess.ID] {
			t.Errorf("duplicate session ID found: %s", sess.ID)
		}
		sessionIDs[sess.ID] = true
	}

	t.Logf("successfully created %d unique sessions concurrently", concurrency)
}

// TestSessionActivityTracking tests LastActivity updates
func TestSessionActivityTracking(t *testing.T) {
	proc, cleanup := setupTestBrowser(t)
	defer cleanup()

	manager := NewManager(nil)
	defer manager.Close()

	// Create session
	session, err := manager.CreateSession(proc.DebugPort)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	initialActivity := session.LastActivity

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Update activity
	session.UpdateActivity()

	if !session.LastActivity.After(initialActivity) {
		t.Error("LastActivity was not updated")
	}

	// Test expiration
	session.LastActivity = time.Now().Add(-1 * time.Hour)
	if !session.IsExpired(30 * time.Minute) {
		t.Error("session should be expired but isn't")
	}

	if session.IsExpired(2 * time.Hour) {
		t.Error("session should not be expired but is")
	}
}