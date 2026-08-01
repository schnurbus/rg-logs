package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID    uuid.UUID
	Email string
}

type Client struct {
	baseURL    string
	anonKey    string
	httpClient *http.Client
}

func NewClient(supabaseURL, anonKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(supabaseURL, "/"),
		anonKey: anonKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetUser validates the access token with Supabase Auth and returns the user.
func (c *Client) GetUser(ctx context.Context, accessToken string) (*User, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("missing access token")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/auth/v1/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("apikey", c.anonKey)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("auth request: %w", err)
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid token (%d): %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode user: %w", err)
	}
	id, err := uuid.Parse(payload.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}
	return &User{ID: id, Email: payload.Email}, nil
}
