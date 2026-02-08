package pool

import (
	"fmt"
	"log/slog"
)

//The load Balancer struct is responsible for balancing the load between the browser processes
type LoadBalancer struct {
	pool *ProcessPool
}

// This function creates a new load balancer
func NewLoadBalancer(pool *ProcessPool) *LoadBalancer {
	return &LoadBalancer{
		pool: pool,
	}
}

// This function balances the load between the browser processes by selecting the browser process with the least number of sessions
func (lb *LoadBalancer) SelectProcess() (*ManagedProcess, error) {
	// 1. Get all the processes from the pool
	processes := lb.pool.GetProcesses()

	//2. Edge case to check if the pool is empty
	if len(processes) == 0 {
		return nil, fmt.Errorf("no processes in the pool")
	}

	// 3. Select the process with the least load
	var selected *ManagedProcess
	var minSessions int64 = -1

	// 3a. Iterate through the processes and find the one with the least sessions
	for _, process := range processes {

		//We first check if the process is healthy
		if !process.IsHealthy() {
			slog.Warn("skipping unhealthy process", "port", process.GetPort())
			continue
		}

		//Then we check if the process has the least number of sessions
		sessionCount := process.GetSessionCount()
		if minSessions == -1 || sessionCount < minSessions {
			minSessions = sessionCount
			selected = process
		}
	}

	//If we didn't find any healthy process, we return an error
	if selected == nil {
		return nil, fmt.Errorf("no healthy processes in the pool")
	}

	//Logging the selected process
	slog.Debug("selected process", 
		"port", selected.GetPort(),
		"current_sessions", selected.GetSessionCount())

	// 3b. Return the selected process
	return selected, nil
	
}

// This function retuns the port of the selected process
func (lb *LoadBalancer) GetPort() (int, error) {
	process, err := lb.SelectProcess()
	if err != nil {
		return 0, err
	}
	return process.GetPort(), nil
}