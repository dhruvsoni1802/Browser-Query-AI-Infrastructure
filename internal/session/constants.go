// In internal/session/constants.go (create new file)

package session

import "fmt"

const (
	// MaxSessionsPerAgent is the maximum number of active sessions per agent
	MaxSessionsPerAgent = 10

	// MaxTotalSessions is the global limit across all agents
	MaxTotalSessions = 100

	// DefaultSessionNamePrefix for auto-generated names
	DefaultSessionNamePrefix = "session"
)

// Error definitions
var (
	ErrSessionLimitReached   = fmt.Errorf("agent session limit reached")
	ErrSessionNameConflict   = fmt.Errorf("session name already exists")
	ErrInvalidSessionName    = fmt.Errorf("invalid session name")
	ErrSessionNotFound       = fmt.Errorf("session not found")
)