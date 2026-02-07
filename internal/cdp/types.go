package cdp

import "encoding/json"

// Command represents a CDP command sent to the browser
type Command struct {
	ID     int                    `json:"id"`
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params,omitempty"`
	SessionID string               `json:"sessionId,omitempty"`
}

// Response represents a CDP response from the browser
// RawMessage is a raw JSON message from the browser. This is because the response can be of any type which we don't know about in advance. We will unmarshal it into the appropriate type later.
type Response struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *ResponseError  `json:"error,omitempty"`
}

// ResponseError represents an error in a CDP response
type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Event represents an unsolicited CDP event from the browser
type Event struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}