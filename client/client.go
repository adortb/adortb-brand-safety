// Package client 提供 adortb-brand-safety 服务的 HTTP 客户端，供 adx 等服务调用。
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ClassifyResult 是 /v1/classify 接口的响应。
type ClassifyResult struct {
	Categories  []string `json:"categories"`
	SafetyScore float64  `json:"safety_score"`
	Blocked     bool     `json:"blocked"`
	Warnings    []string `json:"warnings,omitempty"`
}

// CheckResult 是 /v1/check 接口的响应。
type CheckResult struct {
	Allowed     bool     `json:"allowed"`
	Categories  []string `json:"categories"`
	SafetyScore float64  `json:"safety_score"`
}

// Client 是 brand-safety 服务的 HTTP 客户端。
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New 创建客户端。baseURL 如 "http://brand-safety:8092"。
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 50 * time.Millisecond, // adx 竞价链路要求低延迟
		},
	}
}

// Classify 调用 /v1/classify，返回内容类别和安全评分。
func (c *Client) Classify(ctx context.Context, pageURL, title string) (*ClassifyResult, error) {
	body, err := json.Marshal(map[string]string{"url": pageURL, "title": title})
	if err != nil {
		return nil, fmt.Errorf("brand-safety client: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/classify", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("brand-safety client: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("brand-safety client: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("brand-safety client: unexpected status %d", resp.StatusCode)
	}

	var result ClassifyResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("brand-safety client: decode: %w", err)
	}
	return &result, nil
}

// Check 调用 /v1/check，判断广告主在目标页面是否可投放。
func (c *Client) Check(ctx context.Context, advertiserID int64, pageURL, title string) (*CheckResult, error) {
	body, err := json.Marshal(map[string]any{
		"advertiser_id": advertiserID,
		"url":           pageURL,
		"title":         title,
	})
	if err != nil {
		return nil, fmt.Errorf("brand-safety client: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/check", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("brand-safety client: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("brand-safety client: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("brand-safety client: unexpected status %d", resp.StatusCode)
	}

	var result CheckResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("brand-safety client: decode: %w", err)
	}
	return &result, nil
}
