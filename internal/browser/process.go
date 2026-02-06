package browser

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

type ProcessStatus string

const (
	StatusStarting ProcessStatus = "starting"
	StatusRunning  ProcessStatus = "running"
	StatusStopped  ProcessStatus = "stopped"
	StatusFailed   ProcessStatus = "failed"
)

type Process struct {
	BinaryPath  string        // Path to the chromium binary
	DebugPort   int           // Port for debugging
	UserDataDir string        // Directory for user data
	Cmd         *exec.Cmd     // Command to execute the chromium browser
	StartedAt   time.Time     // Time when the process started
	Status      ProcessStatus // Status of the process
}

// NewProcess creates a new browser process configuration.
// It allocates a free port from the pool and creates a temp directory.
func NewProcess(binaryPath string) (*Process, error) {
	// Get a free port from the pool
	debugPort, err := GetFreePort()
	if err != nil {
		return nil, fmt.Errorf("failed to get free port: %w", err)
	}

	// Convert port string to int
	debugPortInt, err := strconv.Atoi(debugPort)
	if err != nil {
		// Return port since we're failing
		ReturnPort(debugPort)
		return nil, fmt.Errorf("failed to convert port to int: %w", err)
	}

	// Create temporary directory for browser profile
	userDataDir, err := os.MkdirTemp("", "chromium-*")
	if err != nil {
		// Return port since we're failing
		ReturnPort(debugPort)
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return &Process{
		BinaryPath:  binaryPath,
		DebugPort:   debugPortInt,
		UserDataDir: userDataDir,
		Status:      StatusStarting,
	}, nil
}

// buildFlags constructs the command-line flags for Chrome
func (p *Process) buildFlags() []string {
	return []string{
		"--headless=new",                                       // Run in headless mode (no GUI)
		fmt.Sprintf("--remote-debugging-port=%d", p.DebugPort), // Enable DevTools Protocol on this port
		"--no-sandbox",                                         // Disable sandbox (needed in containers)
		"--disable-gpu",                                        // Disable GPU acceleration
		"--disable-dev-shm-usage",                              // Overcome limited resource problems
		fmt.Sprintf("--user-data-dir=%s", p.UserDataDir),       // Where browser stores its data
	}
}

// Start launches the browser process with appropriate flags
func (p *Process) Start() error {
	// Build command with all flags
	p.Cmd = exec.Command(p.BinaryPath, p.buildFlags()...)

	// Start the process
	if err := p.Cmd.Start(); err != nil {
		p.Status = StatusFailed
		return fmt.Errorf("failed to start browser process: %w", err)
	}

	// Update process status and timestamp
	p.Status = StatusRunning
	p.StartedAt = time.Now()

	return nil
}

// Stop gracefully terminates the browser process
func (p *Process) Stop() error {
	// Check if process was ever started
	if p.Cmd == nil || p.Cmd.Process == nil {
		return fmt.Errorf("process was never started")
	}

	// Send SIGTERM for graceful shutdown
	if err := p.Cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send termination signal: %w", err)
	}

	// Wait for process to exit with timeout
	done := make(chan error, 1)
	go func() {
		done <- p.Cmd.Wait()
	}()

	select {
	case err := <-done:
		// Process exited gracefully
		if err != nil && err.Error() != "signal: terminated" {
			return fmt.Errorf("process exit error: %w", err)
		}
	case <-time.After(5 * time.Second):
		// Timeout exceeded - force kill
		if err := p.Cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to force kill process: %w", err)
		}
	}

	// Clean up the user data directory
	if err := os.RemoveAll(p.UserDataDir); err != nil {
		return fmt.Errorf("failed to remove user data directory: %w", err)
	}

	// Update the process status
	p.Status = StatusStopped

	// Return the port to the pool
	ReturnPort(strconv.Itoa(p.DebugPort))

	return nil
}

// IsAlive checks if the process is still running
func (p *Process) IsAlive() bool {
	// Check if cmd or process is nil
	if p.Cmd == nil || p.Cmd.Process == nil {
		return false
	}

	// Send signal 0 - checks existence without affecting the process
	err := p.Cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

// GetPID returns the process ID if the process is running
func (p *Process) GetPID() int {
	if p.Cmd != nil && p.Cmd.Process != nil {
		return p.Cmd.Process.Pid
	}
	return 0
}

// GetDebugURL returns the Chrome DevTools Protocol URL
func (p *Process) GetDebugURL() string {
	return fmt.Sprintf("http://localhost:%d", p.DebugPort)
}