package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dhruvsoni1802/browser-query-ai/internal/browser"
	"github.com/dhruvsoni1802/browser-query-ai/internal/config"
	"github.com/dhruvsoni1802/browser-query-ai/internal/session"
)

func main() {
	// Setup logger
	logger := InitializeLogger()
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	slog.Info("configuration loaded",
		"chromium_path", cfg.ChromiumPath,
		"server_port", cfg.ServerPort,
		"max_browsers", cfg.MaxBrowsers,
	)

	// Create and start browser process
	proc, err := browser.NewProcess(cfg.ChromiumPath)
	if err != nil {
		slog.Error("failed to create browser process", "error", err)
		os.Exit(1)
	}

	if err := proc.Start(); err != nil {
		slog.Error("failed to start browser", "error", err)
		os.Exit(1)
	}

	slog.Info("browser process started",
		"pid", proc.GetPID(),
		"debug_port", proc.DebugPort,
	)

	// Wait for browser to initialize
	time.Sleep(2 * time.Second)

	// Create session manager
	manager := session.NewManager()
	defer manager.Close()

	slog.Info("session manager initialized")

	// ========================================
	// DEMO: Complete Session Workflow
	// ========================================

	// 1. Create a session
	slog.Info("=== Step 1: Creating session ===")
	sess, err := manager.CreateSession(proc.DebugPort)
	if err != nil {
		slog.Error("failed to create session", "error", err)
		proc.Stop()
		os.Exit(1)
	}

	slog.Info("session created",
		"session_id", sess.ID,
		"context_id", sess.ContextID,
	)

	// 2. Navigate to a website
	slog.Info("=== Step 2: Navigating to example.com ===")
	pageID, err := manager.Navigate(sess.ID, "https://example.com")
	if err != nil {
		slog.Error("failed to navigate", "error", err)
		manager.DestroySession(sess.ID)
		proc.Stop()
		os.Exit(1)
	}

	slog.Info("navigation complete", "page_id", pageID)

	// Wait for page to load
	time.Sleep(2 * time.Second)

	// 3. Execute JavaScript to get page title
	slog.Info("=== Step 3: Executing JavaScript ===")
	title, err := manager.ExecuteJavascript(sess.ID, pageID, "document.title")
	if err != nil {
		slog.Error("failed to execute JavaScript", "error", err)
	} else {
		slog.Info("page title retrieved", "title", title)
	}

	// 4. Get page URL
	url, err := manager.ExecuteJavascript(sess.ID, pageID, "window.location.href")
	if err != nil {
		slog.Error("failed to get URL", "error", err)
	} else {
		slog.Info("page URL retrieved", "url", url)
	}

	// 5. Get page content (HTML)
	slog.Info("=== Step 4: Getting page content ===")
	content, err := manager.GetPageContent(sess.ID, pageID)
	if err != nil {
		slog.Error("failed to get page content", "error", err)
	} else {
		slog.Info("page content retrieved", "length", len(content))
	}

	// 6. Capture screenshot
	slog.Info("=== Step 5: Capturing screenshot ===")
	screenshot, err := manager.CaptureScreenshot(sess.ID, pageID)
	if err != nil {
		slog.Error("failed to capture screenshot", "error", err)
	} else {
		// Save screenshot to file
		filename := "example_screenshot.png"
		if err := os.WriteFile(filename, screenshot, 0644); err != nil {
			slog.Error("failed to save screenshot", "error", err)
		} else {
			slog.Info("screenshot saved", "filename", filename, "size", len(screenshot))
		}
	}

	// 7. Open another page in the same session
	slog.Info("=== Step 6: Opening second page ===")
	pageID2, err := manager.Navigate(sess.ID, "https://example.org")
	if err != nil {
		slog.Error("failed to navigate to second page", "error", err)
	} else {
		slog.Info("second page opened", "page_id", pageID2)
		
		// Wait and get title of second page
		time.Sleep(2 * time.Second)
		title2, err := manager.ExecuteJavascript(sess.ID, pageID2, "document.title")
		if err != nil {
			slog.Error("failed to get second page title", "error", err)
		} else {
			slog.Info("second page title", "title", title2)
		}
	}

	// 8. Show session stats
	slog.Info("=== Session Statistics ===",
		"session_id", sess.ID,
		"total_pages", len(sess.PageIDs),
		"pages", sess.PageIDs,
		"created_at", sess.CreatedAt,
		"last_activity", sess.LastActivity,
	)

	// 9. Close first page
	slog.Info("=== Step 7: Closing first page ===")
	if err := manager.ClosePage(sess.ID, pageID); err != nil {
		slog.Error("failed to close page", "error", err)
	} else {
		slog.Info("first page closed", "remaining_pages", len(sess.PageIDs))
	}

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	slog.Info("=== Demo Complete ===")
	slog.Info("service ready",
		"active_sessions", manager.GetSessionCount(),
		"status", "press Ctrl+C to shutdown",
	)

	// Wait for shutdown signal
	sig := <-quit
	slog.Info("shutdown initiated", "signal", sig.String())

	// Cleanup
	slog.Info("cleaning up sessions")
	if err := manager.DestroySession(sess.ID); err != nil {
		slog.Error("failed to destroy session", "error", err)
	}

	if err := proc.Stop(); err != nil {
		slog.Error("failed to stop browser", "error", err)
	}

	total, available := browser.GetPoolStats()
	slog.Info("shutdown complete",
		"ports_total", total,
		"ports_available", available,
	)
}