// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// TenantClient calls the PHP REST API to read and mutate tenant data.
// Reads are cached by TenantCache; the client is called on cache miss.
type TenantClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
	timeout    time.Duration
}

// ClientOption configures TenantClient behaviour.
type ClientOption func(*TenantClient)

// NewTenantClient creates a new PHP API transport with the given base URL and bearer token.
func NewTenantClient(baseURL, token string, opts ...ClientOption) *TenantClient {
	client := &TenantClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		timeout: 10 * time.Second,
	}
	for _, opt := range opts {
		opt(client)
	}
	client.httpClient = &http.Client{Timeout: client.timeout}
	return client
}

// WithTimeout overrides the default 10-second request timeout.
func WithTimeout(d time.Duration) ClientOption {
	return func(c *TenantClient) {
		c.timeout = d
		if c.httpClient != nil {
			c.httpClient.Timeout = d
		}
	}
}

func (c *TenantClient) request(ctx context.Context, method, path string, body any) ([]byte, int, error) {
	if c == nil {
		return nil, 0, ErrWorkspaceNotFound
	}
	endpoint, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return nil, 0, err
	}
	var payload io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		payload = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, payload)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "timeout") {
			return nil, 0, ErrClientTimeout
		}
		return nil, 0, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode >= 400 {
		if len(data) == 0 {
			return nil, resp.StatusCode, c.statusError(resp.StatusCode, path, "")
		}
		return nil, resp.StatusCode, c.statusError(resp.StatusCode, path, string(data))
	}
	return data, resp.StatusCode, nil
}

func (c *TenantClient) statusError(status int, path, body string) error {
	if status == http.StatusNotFound {
		if strings.Contains(path, "/features/") {
			return ErrFeatureNotFound
		}
		return ErrWorkspaceNotFound
	}
	var envelope struct {
		Ok    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if body != "" && json.Unmarshal([]byte(body), &envelope) == nil && envelope.Error != "" {
		return errors.New(envelope.Error)
	}
	return errors.New(http.StatusText(status))
}

func decodeEnvelope[T any](data []byte, target *T) error {
	if len(data) == 0 {
		return io.EOF
	}
	var envelope struct {
		Ok    bool            `json:"ok"`
		Error string          `json:"error"`
		Data  json.RawMessage `json:"data"`
	}
	if json.Unmarshal(data, &envelope) == nil && (envelope.Data != nil || envelope.Error != "") {
		if envelope.Error != "" && !envelope.Ok {
			return errors.New(envelope.Error)
		}
		if len(envelope.Data) > 0 {
			return json.Unmarshal(envelope.Data, target)
		}
	}
	return json.Unmarshal(data, target)
}

func decodeCount(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, io.EOF
	}
	var envelope struct {
		Ok    bool            `json:"ok"`
		Error string          `json:"error"`
		Data  json.RawMessage `json:"data"`
		Count *int            `json:"count"`
		Usage *int            `json:"usage"`
		Value *int            `json:"value"`
	}
	if json.Unmarshal(data, &envelope) == nil {
		if envelope.Error != "" && !envelope.Ok {
			return 0, errors.New(envelope.Error)
		}
		if envelope.Count != nil {
			return *envelope.Count, nil
		}
		if envelope.Usage != nil {
			return *envelope.Usage, nil
		}
		if envelope.Value != nil {
			return *envelope.Value, nil
		}
		if len(envelope.Data) > 0 {
			return decodeCount(envelope.Data)
		}
	}
	if value, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
		return value, nil
	}
	var direct int
	if err := json.Unmarshal(data, &direct); err == nil {
		return direct, nil
	}
	return 0, errors.New("invalid count payload")
}

// GetWorkspaceBySlug fetches a workspace by its slug.
func (c *TenantClient) GetWorkspaceBySlug(ctx context.Context, slug string) (*Workspace, error) {
	data, _, err := c.request(ctx, http.MethodGet, "/api/v1/workspaces/"+url.PathEscape(slug), nil)
	if err != nil {
		return nil, err
	}
	var workspace Workspace
	if err := decodeEnvelope(data, &workspace); err != nil {
		return nil, err
	}
	return &workspace, nil
}

// GetWorkspaceByUUID fetches a workspace by UUID.
func (c *TenantClient) GetWorkspaceByUUID(ctx context.Context, uuid string) (*Workspace, error) {
	data, _, err := c.request(ctx, http.MethodGet, "/api/v1/workspaces/uuid/"+url.PathEscape(uuid), nil)
	if err != nil {
		return nil, err
	}
	var workspace Workspace
	if err := decodeEnvelope(data, &workspace); err != nil {
		return nil, err
	}
	return &workspace, nil
}

// GetWorkspaceBySubdomain resolves a hostname to a workspace.
func (c *TenantClient) GetWorkspaceBySubdomain(ctx context.Context, host string) (*Workspace, error) {
	data, _, err := c.request(ctx, http.MethodGet, "/api/v1/workspaces/subdomain/"+url.PathEscape(host), nil)
	if err != nil {
		return nil, err
	}
	var workspace Workspace
	if err := decodeEnvelope(data, &workspace); err != nil {
		return nil, err
	}
	return &workspace, nil
}

// GetUser fetches the authenticated user by the bearer token on the client.
func (c *TenantClient) GetUser(ctx context.Context) (*User, error) {
	data, _, err := c.request(ctx, http.MethodGet, "/api/v1/user", nil)
	if err != nil {
		return nil, err
	}
	var user User
	if err := decodeEnvelope(data, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// GetPackagesForWorkspace returns all active packages assigned to the workspace.
func (c *TenantClient) GetPackagesForWorkspace(ctx context.Context, wsUUID string) ([]Package, error) {
	data, _, err := c.request(ctx, http.MethodGet, "/api/v1/workspaces/"+url.PathEscape(wsUUID)+"/packages", nil)
	if err != nil {
		return nil, err
	}
	var packages []Package
	if err := decodeEnvelope(data, &packages); err != nil {
		return nil, err
	}
	return packages, nil
}

// GetBoostsForWorkspace returns all active, usable boosts for the workspace.
func (c *TenantClient) GetBoostsForWorkspace(ctx context.Context, wsUUID string) ([]Boost, error) {
	data, _, err := c.request(ctx, http.MethodGet, "/api/v1/workspaces/"+url.PathEscape(wsUUID)+"/boosts", nil)
	if err != nil {
		return nil, err
	}
	var boosts []Boost
	if err := decodeEnvelope(data, &boosts); err != nil {
		return nil, err
	}
	return boosts, nil
}

// GetCurrentUsage returns total usage count for wsUUID+featureCode since reset boundary.
func (c *TenantClient) GetCurrentUsage(ctx context.Context, wsUUID, featureCode string) (int, error) {
	data, _, err := c.request(ctx, http.MethodGet, "/api/v1/workspaces/"+url.PathEscape(wsUUID)+"/usage/"+url.PathEscape(featureCode), nil)
	if err != nil {
		return 0, err
	}
	return decodeCount(data)
}

// RecordUsage POSTs a usage record to the PHP API.
func (c *TenantClient) RecordUsage(ctx context.Context, wsUUID, featureCode string, quantity int, userID *int64, metadata map[string]any) error {
	payload := map[string]any{
		"feature_code": featureCode,
		"quantity":     quantity,
		"metadata":     metadata,
	}
	if userID != nil {
		payload["user_id"] = *userID
	}
	_, _, err := c.request(ctx, http.MethodPost, "/api/v1/workspaces/"+url.PathEscape(wsUUID)+"/usage", payload)
	return err
}

// GetFeature fetches a single feature definition by code.
func (c *TenantClient) GetFeature(ctx context.Context, code string) (*Feature, error) {
	data, _, err := c.request(ctx, http.MethodGet, "/api/v1/features/"+url.PathEscape(code), nil)
	if err != nil {
		return nil, err
	}
	var feature Feature
	if err := decodeEnvelope(data, &feature); err != nil {
		return nil, err
	}
	return &feature, nil
}
