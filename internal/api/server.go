package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/dhruvsoni1802/browser-query-ai/internal/pool"
	"github.com/dhruvsoni1802/browser-query-ai/internal/session"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// Server represents the HTTP API server
type Server struct {
	router  *chi.Mux
	server  *http.Server
	manager *session.Manager
}

// NewServer creates a new HTTP server
func NewServer(port string, manager *session.Manager, loadBalancer *pool.LoadBalancer) *Server {
	router := chi.NewRouter()

	// Middleware
	router.Use(RecoveryMiddleware)
	router.Use(LoggingMiddleware)
	router.Use(middleware.RequestID)
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Create handlers with load balancer
	handlers := NewHandlers(manager, loadBalancer)

	// Register routes (same as before)
	router.Route("/sessions", func(r chi.Router) {
		r.Post("/", handlers.CreateSession)
		r.Get("/", handlers.ListSessions)
		r.Post("/resume", handlers.ResumeSession)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", handlers.GetSession)
			r.Delete("/", handlers.DestroySession)
			r.Post("/navigate", handlers.Navigate)
			r.Post("/execute", handlers.ExecuteJS)
			r.Post("/screenshot", handlers.CaptureScreenshot)
			r.Post("/resume", handlers.ResumeSessionByID)
			r.Put("/rename", handlers.RenameSession)

			r.Route("/pages/{pageId}", func(r chi.Router) {
				r.Get("/content", handlers.GetPageContent)
				r.Delete("/", handlers.ClosePage)
			})
		})
	})

	// Agent routes
	router.Route("/agents/{agentId}", func(r chi.Router) {
		r.Get("/sessions", handlers.ListAgentSessions)
	})

	// Add metrics endpoint
	router.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics := loadBalancer.GetMetrics()
		writeJSON(w, http.StatusOK, metrics)
	})

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		router:  router,
		server:  server,
		manager: manager,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	slog.Info("starting HTTP server", "addr", s.server.Addr)

	err := s.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server error: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the HTTP server
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("shutting down HTTP server")

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("HTTP server shutdown error: %w", err)
	}

	slog.Info("HTTP server stopped")
	return nil
}