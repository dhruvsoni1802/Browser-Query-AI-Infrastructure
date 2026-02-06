package browser

import (
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"sync"
)

const (
    MinPortRange = 9222 // Chrome's default debug port
    MaxPortRange = 9272 // 50 ports for browser processes
)

var (
    freePortsStack []string
    freePortsSet   map[string]bool // Tracks which ports are available
    portStackMutex sync.Mutex
)

// Initialize the port pool at startup
func init() {
    freePortsSet = make(map[string]bool)
    for i := MinPortRange; i < MaxPortRange; i++ {
        port := strconv.Itoa(i)
        freePortsStack = append(freePortsStack, port)
        freePortsSet[port] = true
    }
    slog.Info("port pool initialized", "size", len(freePortsStack))
}

// IsPortAvailable checks if a port is available by attempting to listen on it
func IsPortAvailable(port string) bool {
    listener, err := net.Listen("tcp", ":"+port)
    if err != nil {
        return false
    }
    listener.Close()
    return true
}

// GetFreePort retrieves an available port from the pool
func GetFreePort() (string, error) {
    portStackMutex.Lock()
    defer portStackMutex.Unlock()

    // Try ports from the stack until we find an available one
    for len(freePortsStack) > 0 {
        // Pop from stack
        port := freePortsStack[len(freePortsStack)-1]
        freePortsStack = freePortsStack[:len(freePortsStack)-1]
        delete(freePortsSet, port)

        // Verify port is actually available
        if IsPortAvailable(port) {
            slog.Debug("allocated port from pool", "port", port, "remaining", len(freePortsStack))
            return port, nil
        }

        // Port was in use by another process, try next one
        slog.Debug("port in use by external process", "port", port)
    }

    return "", fmt.Errorf("no free ports available in pool")
}

// ReturnPort returns a port back to the pool for reuse
func ReturnPort(port string) {
    portStackMutex.Lock()
    defer portStackMutex.Unlock()

    // Validate port is in valid range
    portInt, err := strconv.Atoi(port)
    if err != nil || portInt < MinPortRange || portInt >= MaxPortRange {
        slog.Warn("attempted to return invalid port", "port", port)
        return
    }

    // Check if port already in pool
    if freePortsSet[port] {
        slog.Warn("port already in pool, ignoring duplicate return", "port", port)
        return
    }

    // Add back to stack and set
    freePortsStack = append(freePortsStack, port)
    freePortsSet[port] = true
    slog.Info("returned port to pool", "port", port, "available", len(freePortsStack))
}

// GetPoolStats returns current pool statistics (useful for monitoring)
func GetPoolStats() (total, available int) {
    portStackMutex.Lock()
    defer portStackMutex.Unlock()
    
    return MaxPortRange - MinPortRange, len(freePortsStack)
}