package cdp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Target represents a CDP target (page/tab)
// We do a simple mapping from JSON response to a Go Struct
type Target struct {
    ID                    string `json:"id"`
    Type                  string `json:"type"`
    Title                 string `json:"title"`
    URL                   string `json:"url"`
    WebSocketDebuggerURL  string `json:"webSocketDebuggerUrl"`
}

// GetWebSocketURL discovers the WebSocket URL from the debug port.
// It queries http://localhost:PORT/json and returns the first page target's WebSocket URL.
func GetWebSocketURL(ipAddress string, debugPort string) (string, error) {

	// Default to localhost if ipAddress is not provided
	if ipAddress == "" {
		ipAddress = "localhost"
	}

	// Construct the URL using the base URL and the debug port
	url := fmt.Sprintf("http://%s:%s/json", ipAddress, debugPort)

	//We make an HTTP GET request on the URL
	response, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to get WebSocket URL: %w", err)
	}
	defer response.Body.Close()

	// Check if response status is OK
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	//We read the response body using io.ReadAll
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	//We unmarshal the JSON response into a list of Targets
	var targets []Target
	err = json.Unmarshal(body, &targets)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	//Edge case to check if the target list is empty
	if len(targets) == 0 {
    return "", fmt.Errorf("no targets available - browser may still be starting")
	}


	//Now we find the first target that is a page
	for _, target := range targets {
		if target.Type == "page" {
			return target.WebSocketDebuggerURL, nil
		}
	}

	//If we don't find a page target, we return an error
	return "", fmt.Errorf("no page target found")
}