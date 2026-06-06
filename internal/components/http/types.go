package http

import "time"

// ClientConfig configures an HTTP client.
type ClientConfig struct {
	BaseURL    string        // Base URL.
	AuthToken  string        // Optional authentication token.
	Timeout    time.Duration // Request timeout.
	MaxRetries int           // Maximum retry count (default 3).
}

// RequestOptions describes an HTTP request.
type RequestOptions struct {
	Method      string            // HTTP method.
	Path        string            // Request path.
	QueryParams map[string]string // Query parameters.
	Headers     map[string]string // Custom headers.
	Body        interface{}       // Request body, automatically marshaled as JSON.
	Response    interface{}       // Response body, automatically unmarshaled.
}
