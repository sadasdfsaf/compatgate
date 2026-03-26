package upload

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/compatgate/compatgate/internal/findings"
)

type GitMetadata struct {
	Repository string `json:"repository,omitempty"`
	SHA        string `json:"sha,omitempty"`
	Ref        string `json:"ref,omitempty"`
}

type UploadRequest struct {
	ProjectID string            `json:"projectId"`
	Status    string            `json:"status"`
	Protocol  findings.Protocol `json:"protocol"`
	Git       GitMetadata       `json:"git,omitempty"`
	Report    findings.Report   `json:"report"`
}

type UploadResponse struct {
	Data struct {
		RunID  string `json:"runId"`
		RunURL string `json:"runUrl"`
	} `json:"data"`
}

type Project struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Repository      string `json:"repository,omitempty"`
	DefaultProtocol string `json:"defaultProtocol,omitempty"`
	Owner           string `json:"owner,omitempty"`
	ProjectToken    string `json:"projectToken,omitempty"`
	CreatedAt       string `json:"createdAt,omitempty"`
}

type CreateProjectRequest struct {
	Name            string `json:"name"`
	Repository      string `json:"repository,omitempty"`
	DefaultProtocol string `json:"defaultProtocol,omitempty"`
}

type CreateProjectResponse struct {
	Data Project `json:"data"`
}

type ListProjectsResponse struct {
	Data []Project `json:"data"`
}

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (c *Client) Upload(ctx context.Context, token string, payload UploadRequest) (UploadResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return UploadResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/v1/ingest/runs", bytes.NewReader(body))
	if err != nil {
		return UploadResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return UploadResponse{}, err
	}
	defer res.Body.Close()
	responseBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return UploadResponse{}, err
	}
	if res.StatusCode >= 400 {
		return UploadResponse{}, fmt.Errorf("upload failed: %s", strings.TrimSpace(string(responseBytes)))
	}
	var response UploadResponse
	if err := json.Unmarshal(responseBytes, &response); err != nil {
		return UploadResponse{}, err
	}
	return response, nil
}

func (c *Client) CreateProject(ctx context.Context, user string, payload CreateProjectRequest) (CreateProjectResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return CreateProjectResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/v1/projects", bytes.NewReader(body))
	if err != nil {
		return CreateProjectResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CompatGate-User", user)
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return CreateProjectResponse{}, err
	}
	defer res.Body.Close()
	responseBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return CreateProjectResponse{}, err
	}
	if res.StatusCode >= 400 {
		return CreateProjectResponse{}, fmt.Errorf("create project failed: %s", strings.TrimSpace(string(responseBytes)))
	}
	var response CreateProjectResponse
	if err := json.Unmarshal(responseBytes, &response); err != nil {
		return CreateProjectResponse{}, err
	}
	return response, nil
}

func (c *Client) ListProjects(ctx context.Context, user string) (ListProjectsResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/v1/projects", nil)
	if err != nil {
		return ListProjectsResponse{}, err
	}
	req.Header.Set("X-CompatGate-User", user)
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return ListProjectsResponse{}, err
	}
	defer res.Body.Close()
	responseBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return ListProjectsResponse{}, err
	}
	if res.StatusCode >= 400 {
		return ListProjectsResponse{}, fmt.Errorf("list projects failed: %s", strings.TrimSpace(string(responseBytes)))
	}
	var response ListProjectsResponse
	if err := json.Unmarshal(responseBytes, &response); err != nil {
		return ListProjectsResponse{}, err
	}
	return response, nil
}
