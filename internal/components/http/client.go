package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Client is a general-purpose HTTP client.
type Client struct {
	httpClient *http.Client
	baseURL    string
	authToken  string
	maxRetries int
}

// NewClient creates an HTTP client.
func NewClient(cfg ClientConfig) *Client {
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 1 // Default retry count.
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		baseURL:    cfg.BaseURL,
		authToken:  cfg.AuthToken,
		maxRetries: cfg.MaxRetries,
	}
}

// DoRequest executes an HTTP request.
func (c *Client) DoRequest(ctx context.Context, opts RequestOptions) error {
	return c.doRequestOnce(ctx, opts)
}

// doRequestOnce executes one HTTP request attempt.
func (c *Client) doRequestOnce(ctx context.Context, opts RequestOptions) error {
	// Build the URL.
	reqURL := c.baseURL + opts.Path

	// Add query parameters.
	if len(opts.QueryParams) > 0 {
		params := url.Values{}
		for k, v := range opts.QueryParams {
			params.Set(k, v)
		}
		reqURL += "?" + params.Encode()
	}

	// Build the request body.
	var bodyReader io.Reader
	if opts.Body != nil {
		data, err := json.Marshal(opts.Body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	// Create the HTTP request.
	req, err := http.NewRequestWithContext(ctx, opts.Method, reqURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers.
	req.Header.Set("Content-Type", "application/json")

	// Set the authentication token.
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	// Set custom headers.
	for k, v := range opts.Headers {
		req.Header.Set(k, v)
	}

	// Send the request.
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check the HTTP status code.
	/*if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}*/

	// Parse the response body.
	if opts.Response != nil {
		if err := json.Unmarshal(body, opts.Response); err != nil {
			return fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
		}
	}

	return nil
}

// DoRequestRaw executes an HTTP request and returns the raw response without parsing JSON.
func (c *Client) DoRequestRaw(ctx context.Context, opts RequestOptions) ([]byte, error) {
	var responseBody []byte
	var err error

	operation := func() error {
		// Build the URL.
		reqURL := c.baseURL + opts.Path

		// Add query parameters.
		if len(opts.QueryParams) > 0 {
			params := url.Values{}
			for k, v := range opts.QueryParams {
				params.Set(k, v)
			}
			reqURL += "?" + params.Encode()
		}

		// Build the request body.
		var bodyReader io.Reader
		if opts.Body != nil {
			data, marshalErr := json.Marshal(opts.Body)
			if marshalErr != nil {
				return fmt.Errorf("failed to marshal request body: %w", marshalErr)
			}
			bodyReader = bytes.NewReader(data)
		}

		// Create the HTTP request.
		req, createErr := http.NewRequestWithContext(ctx, opts.Method, reqURL, bodyReader)
		if createErr != nil {
			return fmt.Errorf("failed to create request: %w", createErr)
		}

		// Set default headers.
		req.Header.Set("Content-Type", "application/json")

		// Set the authentication token.
		if c.authToken != "" {
			req.Header.Set("Authorization", "Bearer "+c.authToken)
		}

		// Set custom headers.
		for k, v := range opts.Headers {
			req.Header.Set(k, v)
		}

		// Send the request.
		resp, doErr := c.httpClient.Do(req)
		if doErr != nil {
			return fmt.Errorf("request failed: %w", doErr)
		}
		defer resp.Body.Close()

		// Read the response body.
		responseBody, err = io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		// Check the HTTP status code.
		if resp.StatusCode >= 400 {
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(responseBody))
		}

		return nil
	}

	if err := operation(); err != nil {
		return nil, err
	}

	return responseBody, nil
}
