package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Client manages communication with the AlertOps v2 API.
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewClient creates a new AlertOps API client.
func NewClient(apiKey string) *Client {
	return &Client{
		BaseURL: "https://api.alertops.com/api/v2",
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// doRequest performs an HTTP request with proper headers.
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	url := c.BaseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("error marshaling request body: %w", err)
		}
		tflog.Debug(ctx, "API Request", map[string]interface{}{
			"method": method,
			"url":    url,
			"body":   string(jsonData),
		})
		reqBody = bytes.NewBuffer(jsonData)
	} else {
		tflog.Debug(ctx, "API Request", map[string]interface{}{
			"method": method,
			"url":    url,
		})
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("api-key", c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error performing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	tflog.Debug(ctx, "API Response", map[string]interface{}{
		"status": resp.StatusCode,
		"body":   string(respBody),
	})

	if resp.StatusCode == 204 {
		return []byte("{}"), nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// Get performs a GET request.
func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	return c.doRequest(ctx, http.MethodGet, path, nil)
}

// Post performs a POST request.
func (c *Client) Post(ctx context.Context, path string, body interface{}) ([]byte, error) {
	return c.doRequest(ctx, http.MethodPost, path, body)
}

// Put performs a PUT request.
func (c *Client) Put(ctx context.Context, path string, body interface{}) ([]byte, error) {
	return c.doRequest(ctx, http.MethodPut, path, body)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) ([]byte, error) {
	return c.doRequest(ctx, http.MethodDelete, path, nil)
}
