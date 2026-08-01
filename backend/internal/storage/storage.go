package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultBucket = "combat-logs"

type Client struct {
	baseURL        string
	serviceRoleKey string
	bucket         string
	httpClient     *http.Client
}

func NewClient(supabaseURL, serviceRoleKey, bucket string) *Client {
	if bucket == "" {
		bucket = DefaultBucket
	}
	return &Client{
		baseURL:        strings.TrimRight(supabaseURL, "/"),
		serviceRoleKey: serviceRoleKey,
		bucket:         bucket,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *Client) Bucket() string {
	return c.bucket
}

func (c *Client) objectURL(path string) string {
	escaped := strings.ReplaceAll(url.PathEscape(path), "%2F", "/")
	return fmt.Sprintf("%s/storage/v1/object/%s/%s", c.baseURL, c.bucket, escaped)
}

func (c *Client) Upload(ctx context.Context, path string, contentType string, data []byte) error {
	if contentType == "" {
		contentType = "text/plain"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.objectURL(path), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.serviceRoleKey)
	req.Header.Set("apikey", c.serviceRoleKey)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-upsert", "true")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("storage upload: %w", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("storage upload failed (%d): %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *Client) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.objectURL(path), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.serviceRoleKey)
	req.Header.Set("apikey", c.serviceRoleKey)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("storage download: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		defer res.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		return nil, fmt.Errorf("storage download failed (%d): %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	return res.Body, nil
}

func (c *Client) Delete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.objectURL(path), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.serviceRoleKey)
	req.Header.Set("apikey", c.serviceRoleKey)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("storage delete: %w", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode == http.StatusNotFound {
		return nil
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("storage delete failed (%d): %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}
