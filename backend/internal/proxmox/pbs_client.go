package proxmox

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type PBSClient struct {
	baseURL     string
	tokenID     string
	tokenSecret string
	httpClient  *http.Client
}

func NewPBSClient(hostname string, port int, tokenID, tokenSecret string, insecureSkipVerify ...bool) *PBSClient {
	skipVerify := true // default for self-signed Proxmox certs
	if len(insecureSkipVerify) > 0 {
		skipVerify = insecureSkipVerify[0]
	}
	return &PBSClient{
		baseURL:     fmt.Sprintf("https://%s:%d/api2/json", hostname, port),
		tokenID:     tokenID,
		tokenSecret: tokenSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: skipVerify},
			},
		},
	}
}

func (c *PBSClient) doRequest(ctx context.Context, method, path string) (json.RawMessage, error) {
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("PBSAPIToken=%s:%s", c.tokenID, c.tokenSecret))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("PBS API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return result.Data, nil
}

func (c *PBSClient) GetDatastores(ctx context.Context) ([]PBSDatastore, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/admin/datastore")
	if err != nil {
		return nil, err
	}
	var datastores []PBSDatastore
	if err := json.Unmarshal(data, &datastores); err != nil {
		return nil, fmt.Errorf("unmarshal datastores: %w", err)
	}
	return datastores, nil
}

func (c *PBSClient) GetDatastoreStatus(ctx context.Context) ([]PBSDatastoreStatus, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/status/datastore-usage")
	if err != nil {
		return nil, err
	}
	var statuses []PBSDatastoreStatus
	if err := json.Unmarshal(data, &statuses); err != nil {
		return nil, fmt.Errorf("unmarshal datastore status: %w", err)
	}
	for i := range statuses {
		if statuses[i].Total > 0 {
			statuses[i].UsagePercent = float64(statuses[i].Used) / float64(statuses[i].Total) * 100
		}
	}
	return statuses, nil
}

func (c *PBSClient) GetBackupJobs(ctx context.Context) ([]PBSBackupJob, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/config/sync")
	if err != nil {
		return nil, err
	}
	var jobs []PBSBackupJob
	if err := json.Unmarshal(data, &jobs); err != nil {
		return nil, fmt.Errorf("unmarshal backup jobs: %w", err)
	}
	return jobs, nil
}
