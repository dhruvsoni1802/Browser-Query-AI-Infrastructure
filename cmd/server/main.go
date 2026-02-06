package main

import (
	"log/slog"  // library for structured logging
	"os"        // library for os related operations
	"os/signal" // library for signal handling such as Ctrl+C and kill signals
	"syscall"   // library for system call constants
	"time"      // library for time formatting
)

//Function to initialize the logger
func setupLogger() *slog.Logger {
	var handler slog.Handler

	if os.Getenv("ENV") == "production" {

		// Initialize JSON handler for production environment
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{ Level: slog.LevelInfo })
	} else {

		// Initialize Text handler for development environment with better formatting
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{ 
			Level: slog.LevelDebug,
			AddSource: false,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				// Format timestamp to be more readable
				if a.Key == slog.TimeKey {
					t := a.Value.Time()
					return slog.String("time", t.Format(time.DateTime))
				}
				return a
			},
		})
	}

	// Create a new logger with the initialized handler
	return slog.New(handler)
}

// Main entry point of the program
func main() {

	// Setup the logger
	logger := setupLogger()
	slog.SetDefault(logger)

	var serverPort string = "8080"
	var debugPort int = 9222 // Default debug port for Chrome DevTools

	slog.Info("Browser Query AI Server starting", "server_port", serverPort, "debug_port", debugPort)

	// Create a channel to receive shutdown signals
	quit := make(chan os.Signal, 1)

	// Notify the channel for SIGINT and SIGTERM signals
	// Ctrl+C is SIGINT, kill signal is SIGTERM
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    
	// Log the service is ready and awaiting shutdown signal
	slog.Info("Service ready", "status", "awaiting shutdown signal")
    
	// Wait for a shutdown signal
  sig := <-quit

	// Log the shutdown initiated with the signal
  slog.Info("shutdown initiated", "signal", sig.String())
    
    
	// Log the shutdown complete
  slog.Info("shutdown complete")
}

