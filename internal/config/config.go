package config

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"
)

// Config holds all service configuration
type Config struct {
	//Browser configuration
	ChromiumPath string
	ServerPort   string
	MaxBrowsers  int

	//Redis configuration
	RedisAddr    string
	RedisPassword string
	RedisDB      int
	SessionTTL   time.Duration
}

func Load() (*Config, error) {
	chromiumPath, err := findChromium()
	if err != nil {
		return nil, err
	}

	return &Config{
		ChromiumPath:  chromiumPath,
		ServerPort:    getEnv("SERVER_PORT", "8080"),
		MaxBrowsers:   getEnvAsInt("MAX_BROWSERS", 5),
		
		// Redis defaults
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 0),
		SessionTTL:    getEnvAsDuration("SESSION_TTL", 1*time.Hour),
	}, nil
}

func getEnv(key string, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

func getEnvAsInt(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return intVal
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	
	duration, err := time.ParseDuration(val)
	if err != nil {
		return defaultVal
	}
	
	return duration
}


// Function to find the Chromium binary path
func findChromium() (string, error) {
	
	// Check if CHROMIUM_PATH environment variable is set
	customPath := os.Getenv("CHROMIUM_PATH")
	if customPath != "" {
		
		// Validate the custom path exists
		if !fileExists(customPath) {
			return "", fmt.Errorf("chromium binary not found at path: %s", customPath)
		}

		// Validate the custom path is executable
		if !isExecutable(customPath) {
			return "", fmt.Errorf("chromium binary found but not executable: %s", customPath)
		}
		return customPath, nil
	}

	// Get current operating system
	currentOS := runtime.GOOS

	// Get common paths for this OS
	paths := getChromiumPaths(currentOS)

	// Search through common paths
	for _, path := range paths {
		if fileExists(path) && isExecutable(path) {
			return path, nil
		}
	}

	// If we get here, chromium wasn't found anywhere
	return "", fmt.Errorf("chromium not found in common paths for %s, set CHROMIUM_PATH environment variable", currentOS)
}

// getChromiumPaths returns common Chromium installation paths based on OS.
func getChromiumPaths(operatingSystem string) []string {
	// macOS paths
	if operatingSystem == "darwin" {
		return []string{
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		}
	}

	// Linux paths
	if operatingSystem == "linux" {
		return []string{
			"/usr/bin/chromium-browser",
			"/usr/bin/chromium",
			"/snap/bin/chromium",
		}
	}

	// TODO: Add Windows paths later

	// Unsupported OS
	return []string{}
}