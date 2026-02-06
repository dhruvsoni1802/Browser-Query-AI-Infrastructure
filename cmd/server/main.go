package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/dhruvsoni1802/browser-query-ai/internal/browser"
	"github.com/dhruvsoni1802/browser-query-ai/internal/config"
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

	// Create browser process
	proc, err := browser.NewProcess(cfg.ChromiumPath)
	if err != nil {
		slog.Error("failed to create browser process", "error", err)
		os.Exit(1)
	}

	// Start the browser
	if err := proc.Start(); err != nil {
		slog.Error("failed to start browser", "error", err)
		os.Exit(1)
	}

	slog.Info("browser process started",
		"pid", proc.GetPID(),
		"debug_port", proc.DebugPort,
		"debug_url", proc.GetDebugURL(),
		"status", proc.Status,
	)

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	slog.Info("service ready", "status", "awaiting shutdown signal")

	// Wait for shutdown signal
	sig := <-quit
	slog.Info("shutdown initiated", "signal", sig.String())

	// Stop the browser
	if err := proc.Stop(); err != nil {
		slog.Error("failed to stop browser", "error", err)
		os.Exit(1)
	}

	// Get port pool stats
	total, available := browser.GetPoolStats()
	slog.Info("shutdown complete", 
		"ports_total", total, 
		"ports_available", available,
	)
}