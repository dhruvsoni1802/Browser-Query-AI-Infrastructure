package pool

import (
	"fmt"
	"log/slog"
	"sync"
)

// ProcessPool manages a pool of browser processes
type ProcessPool struct {
	processes    []*ManagedProcess // Pool of browser processes
	chromiumPath string            // Path to chromium binary
	maxProcesses int               // Maximum number of processes
	mu           sync.RWMutex      // Protects processes slice
}

// PoolMetrics contains metrics about the entire pool
type PoolMetrics struct {
	TotalProcesses int              `json:"total_processes"`
	TotalSessions  int64            `json:"total_sessions"`
	Processes      []ProcessMetrics `json:"processes"`
}

// NewProcessPool creates a new process pool
func NewProcessPool(chromiumPath string, poolSize int) (*ProcessPool, error) {
	// Validate pool size
	if poolSize < 1 || poolSize > 10 {
		return nil, fmt.Errorf("pool size must be between 1 and 10, got %d", poolSize)
	}

	// Create process pool
	pool := &ProcessPool{
		processes:    make([]*ManagedProcess, 0, poolSize),
		chromiumPath: chromiumPath,
		maxProcesses: poolSize,
	}

	// Start managed processes
	for i := 0; i < poolSize; i++ {
		process, err := NewManagedProcess(chromiumPath)
		if err != nil {
			// Cleanup on failure - stop all processes started so far
			slog.Error("failed to start process, cleaning up", "index", i, "error", err)
			pool.Shutdown()
			return nil, fmt.Errorf("failed to start process %d: %w", i, err)
		}
		pool.processes = append(pool.processes, process)
		slog.Info("started browser process", "index", i, "port", process.GetPort())
	}

	slog.Info("process pool initialized", "size", poolSize)
	return pool, nil
}

// GetProcesses returns a copy of all processes (for monitoring)
func (p *ProcessPool) GetProcesses() []*ManagedProcess {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Return a copy to prevent external modification
	processes := make([]*ManagedProcess, len(p.processes))
	copy(processes, p.processes)
	return processes
}

// GetProcessCount returns the number of processes in the pool
func (p *ProcessPool) GetProcessCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.processes)
}

// Shutdown stops all processes in the pool (best effort)
func (p *ProcessPool) Shutdown() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errors []error

	// Stop all processes, collecting errors but continuing
	for i, process := range p.processes {
		if err := process.Stop(); err != nil {
			slog.Warn("failed to stop process", "index", i, "port", process.GetPort(), "error", err)
			errors = append(errors, err)
		} else {
			slog.Info("process stopped", "index", i, "port", process.GetPort())
		}
	}

	// Clear the slice even if some processes failed to stop
	p.processes = p.processes[:0]

	// Return error if any processes failed to stop
	if len(errors) > 0 {
		slog.Warn("shutdown completed with errors", "failed_count", len(errors))
		return fmt.Errorf("failed to stop %d processes", len(errors))
	}

	slog.Info("all processes shut down successfully")
	return nil
}

// GetMetrics returns metrics for the entire pool
func (p *ProcessPool) GetMetrics() PoolMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Calculate total sessions and collect process metrics
	var totalSessions int64
	processMetrics := make([]ProcessMetrics, len(p.processes))

	for i, process := range p.processes {
		metrics := process.GetMetrics()
		processMetrics[i] = metrics
		totalSessions += metrics.SessionCount
	}

	return PoolMetrics{
		TotalProcesses: len(p.processes),
		TotalSessions:  totalSessions,
		Processes:      processMetrics,
	}
}