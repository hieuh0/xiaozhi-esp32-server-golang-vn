package http

import (
	"context"
	"time"
)

// ManagerClient is an HTTP client for the manager backend.
type ManagerClient struct {
	client *Client
}

// ManagerClientConfig configures a manager client.
type ManagerClientConfig struct {
	BaseURL    string        // Manager backend URL.
	AuthToken  string        // Optional authentication token.
	Timeout    time.Duration // Request timeout.
	MaxRetries int           // Maximum retry count.
}

// NewManagerClient creates an HTTP client for the manager backend.
func NewManagerClient(cfg ManagerClientConfig) *ManagerClient {
	client := NewClient(ClientConfig{
		BaseURL:    cfg.BaseURL,
		AuthToken:  cfg.AuthToken,
		Timeout:    cfg.Timeout,
		MaxRetries: cfg.MaxRetries,
	})

	return &ManagerClient{
		client: client,
	}
}

// DoRequest executes an HTTP request through the general client.
func (m *ManagerClient) DoRequest(ctx context.Context, opts RequestOptions) error {
	return m.client.DoRequest(ctx, opts)
}

// DoRequestRaw executes an HTTP request and returns the raw response.
func (m *ManagerClient) DoRequestRaw(ctx context.Context, opts RequestOptions) ([]byte, error) {
	return m.client.DoRequestRaw(ctx, opts)
}
