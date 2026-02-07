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

	// Demo: Create a session
	slog.Info("creating demo session")
	sess, err := manager.CreateSession(proc.DebugPort)
	if err != nil {
		slog.Error("failed to create session", "error", err)
		proc.Stop()
		os.Exit(1)
	}

	slog.Info("demo session created",
		"session_id", sess.ID,
		"context_id", sess.ContextID,
		"status", sess.Status,
	)

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	slog.Info("service ready",
		"active_sessions", manager.GetSessionCount(),
		"status", "press Ctrl+C to shutdown",
	)

	// Wait for shutdown signal
	sig := <-quit
	slog.Info("shutdown initiated", "signal", sig.String())

	// Cleanup demo session
	slog.Info("cleaning up sessions")
	if err := manager.DestroySession(sess.ID); err != nil {
		slog.Error("failed to destroy session", "error", err)
	}

	// Stop browser
	if err := proc.Stop(); err != nil {
		slog.Error("failed to stop browser", "error", err)
	}

	// Get port stats
	total, available := browser.GetPoolStats()
	slog.Info("shutdown complete",
		"ports_total", total,
		"ports_available", available,
	)
}