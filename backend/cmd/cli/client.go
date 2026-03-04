package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type APIClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewAPIClient(baseURL, token string) *APIClient {
	return &APIClient{
		baseURL: strings.TrimRight(baseURL, "/") + "/api/v1",
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type APIResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error,omitempty"`
}

func (c *APIClient) doRequest(method, path string, body io.Reader) (*APIResponse, error) {
	url := c.baseURL + path

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("request erstellen: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("X-API-Key", c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request ausfuehren: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("response lesen: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API Fehler %d: %s", resp.StatusCode, string(data))
	}

	var apiResp APIResponse
	if err := json.Unmarshal(data, &apiResp); err != nil {
		// If not wrapped in success/data, return raw
		apiResp.Success = true
		apiResp.Data = data
	}

	if !apiResp.Success && apiResp.Error != "" {
		return nil, fmt.Errorf("API Fehler: %s", apiResp.Error)
	}

	return &apiResp, nil
}

func (c *APIClient) Get(path string) (*APIResponse, error) {
	return c.doRequest(http.MethodGet, path, nil)
}

func (c *APIClient) Post(path string, body io.Reader) (*APIResponse, error) {
	return c.doRequest(http.MethodPost, path, body)
}

func (c *APIClient) Delete(path string) (*APIResponse, error) {
	return c.doRequest(http.MethodDelete, path, nil)
}
