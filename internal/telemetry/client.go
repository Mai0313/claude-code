package telemetry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"claude_analysis/internal/config"
)

// Client handles telemetry data submission to the API
type Client struct {
	httpClient *http.Client
	config     *config.Config
}

// New creates a new telemetry client
func New(cfg *config.Config) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: cfg.API.Timeout,
		},
		config: cfg,
	}
}

// Submit sends telemetry data to the API and returns the response
func (c *Client) Submit(data interface{}) (map[string]interface{}, error) {
	// Check if data is empty
	// 支援傳入 array 或 map
	var jsonData []byte
	var err error
	if data == nil {
		return make(map[string]interface{}), nil
	}
	jsonData, err = json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", c.config.API.Endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response JSON
	var responseDict map[string]interface{}
	if err := json.Unmarshal(responseBody, &responseDict); err != nil {
		return nil, fmt.Errorf("failed to parse response JSON: %w", err)
	}

	return responseDict, nil
}
