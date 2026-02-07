package cdp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// GetWebSocketURL discovers the browser-level WebSocket URL
func GetWebSocketURL(host string, debugPort string) (string, error) {

	// If host is not provided, use localhost
	if host == "" {
		host = "localhost"
	}

	// Query the /json/version endpoint for browser-level WebSocket
	// Use %s for string port, not %d for integer
	url := fmt.Sprintf("http://%s:%s/json/version", host, debugPort)

	response, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to connect to debug port: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse browser version info
	var versionInfo struct {
		Browser              string `json:"Browser"`
		ProtocolVersion      string `json:"Protocol-Version"`
		UserAgent            string `json:"User-Agent"`
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}

	if err := json.Unmarshal(body, &versionInfo); err != nil {
		return "", fmt.Errorf("failed to parse JSON response: %w", err)
	}

	if versionInfo.WebSocketDebuggerURL == "" {
		return "", fmt.Errorf("no browser WebSocket URL found")
	}

	return versionInfo.WebSocketDebuggerURL, nil
}