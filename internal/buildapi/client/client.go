// Copyright 2026 Red Hat Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/centos-automotive-suite/bob/internal/buildapi"
)

type Client struct {
	BaseURL    string
	Token      string
	Namespace  string
	HTTPClient *http.Client
}

func New(baseURL, token, namespace string) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Token:      token,
		Namespace:  namespace,
		HTTPClient: http.DefaultClient,
	}
}

func (c *Client) List(ctx context.Context) ([]buildapi.BuildJobSummary, error) {
	url := fmt.Sprintf("%s/v1/namespaces/%s/buildjobs", c.BaseURL, c.Namespace)
	body, err := c.doGet(ctx, url)
	if err != nil {
		return nil, err
	}
	var resp buildapi.BuildJobListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return resp.Items, nil
}

func (c *Client) Get(ctx context.Context, name string) (*buildapi.BuildJobSummary, error) {
	url := fmt.Sprintf("%s/v1/namespaces/%s/buildjobs/%s", c.BaseURL, c.Namespace, name)
	body, err := c.doGet(ctx, url)
	if err != nil {
		return nil, err
	}
	var resp buildapi.BuildJobSummary
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &resp, nil
}

func (c *Client) Run(ctx context.Context, name string) (*buildapi.BuildJobSummary, error) {
	url := fmt.Sprintf("%s/v1/namespaces/%s/buildjobs/%s/run", c.BaseURL, c.Namespace, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	c.setAuth(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
	}
	var result buildapi.BuildJobSummary
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &result, nil
}

func (c *Client) Delete(ctx context.Context, name string) error {
	url := fmt.Sprintf("%s/v1/namespaces/%s/buildjobs/%s", c.BaseURL, c.Namespace, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	c.setAuth(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (c *Client) Logs(ctx context.Context, name string) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/v1/namespaces/%s/buildjobs/%s/logs", c.BaseURL, c.Namespace, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	c.setAuth(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
	}
	return resp.Body, nil
}

func (c *Client) ListArtifacts(ctx context.Context, name string) (*buildapi.ArtifactListResponse, error) {
	url := fmt.Sprintf("%s/v1/namespaces/%s/buildjobs/%s/artifacts", c.BaseURL, c.Namespace, name)
	body, err := c.doGet(ctx, url)
	if err != nil {
		return nil, err
	}
	var resp buildapi.ArtifactListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &resp, nil
}

func (c *Client) DownloadArtifact(ctx context.Context, name, filename string) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/v1/namespaces/%s/buildjobs/%s/artifacts/%s", c.BaseURL, c.Namespace, name, filename)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	c.setAuth(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
	}
	return resp.Body, nil
}

func (c *Client) doGet(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	c.setAuth(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func (c *Client) setAuth(req *http.Request) {
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
}
