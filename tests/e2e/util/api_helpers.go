package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// APIResponse wraps the standard HTTP response for API calls
type APIResponse struct {
	StatusCode int
	Body       []byte
	Error      error
}

// APIClient provides methods for making API requests
type APIClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewAPIClient creates a new API client with the given base URL
func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// PostJSON sends a POST request with JSON body and returns the response
func (c *APIClient) PostJSON(endpoint string, body []byte) (*APIResponse, error) {
	url := fmt.Sprintf("%s%s", c.BaseURL, endpoint)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &APIResponse{
		StatusCode: resp.StatusCode,
		Body:       respBody,
		Error:      nil,
	}, nil
}

// GetWithHeaders sends a GET request and returns the response, allowing custom headers
func (c *APIClient) GetWithHeaders(endpoint string, headers map[string]string) (*APIResponse, error) {
	url := fmt.Sprintf("%s%s", c.BaseURL, endpoint)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &APIResponse{
		StatusCode: resp.StatusCode,
		Body:       respBody,
		Error:      nil,
	}, nil
}

// WaitForAPI waits for the API to become available
func (c *APIClient) WaitForAPI(maxAttempts int, interval time.Duration) error {
	for i := 0; i < maxAttempts; i++ {
		_, err := c.PostJSON("/cosmos/base/tendermint/v1beta1/blocks/latest", nil)
		if err == nil {
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("API not available after %d attempts", maxAttempts)
}

// UnmarshalResponse unmarshals the response body into the given interface
func UnmarshalResponse(resp *APIResponse, v interface{}) error {
	if resp.Error != nil {
		return resp.Error
	}

	if err := json.Unmarshal(resp.Body, v); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}
