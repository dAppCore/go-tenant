// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"dappco.re/go/core"
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
		return nil, 0, core.E("tenant", "tenant client is nil", nil)
	}
	endpoint, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return nil, 0, core.E("tenant", "failed to build api request path", err)
	}
	var payload io.Reader
	if body != nil {
		result := core.JSONMarshal(body)
		if !result.OK {
			if err, ok := result.Value.(error); ok {
				return nil, 0, core.E("tenant", "failed to encode api request body", err)
			}
			return nil, 0, core.E("tenant", "failed to encode api request body", nil)
		}
		payload = bytes.NewReader(result.Value.([]byte))
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, payload)
	if err != nil {
		return nil, 0, core.E("tenant", "failed to create api request", err)
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
		return nil, 0, core.E("tenant", "api request failed", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, core.E("tenant", "failed to read api response", err)
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
	if body != "" {
		var envelope map[string]any
		if core.JSONUnmarshalString(body, &envelope).OK {
			if message := stringField(envelope, "error"); message != "" {
				return core.E("tenant", message, nil)
			}
		}
	}
	return core.E("tenant", http.StatusText(status), nil)
}

func decodeEnvelope[T any](data []byte, target *T) error {
	if len(data) == 0 {
		return io.EOF
	}
	payload := string(data)
	var envelope map[string]any
	if core.JSONUnmarshalString(payload, &envelope).OK && looksLikeEnvelope(envelope) {
		if message := stringField(envelope, "error"); message != "" && !boolField(envelope, "ok") {
			return core.E("tenant", message, nil)
		}
		if nested, ok := envelope["data"]; ok && nested != nil {
			nestedResult := core.JSONMarshal(nested)
			if !nestedResult.OK {
				return core.E("tenant", "invalid api payload", nil)
			}
			nestedJSON := string(nestedResult.Value.([]byte))
			if result := core.JSONUnmarshalString(nestedJSON, target); result.OK {
				return nil
			}
			return core.E("tenant", "invalid api payload", nil)
		}
	}
	if result := core.JSONUnmarshalString(payload, target); result.OK {
		return nil
	}
	return core.E("tenant", "invalid api payload", nil)
}

func decodeCount(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, io.EOF
	}
	payload := string(data)
	var envelope map[string]any
	if core.JSONUnmarshalString(payload, &envelope).OK && looksLikeEnvelope(envelope) {
		if message := stringField(envelope, "error"); message != "" && !boolField(envelope, "ok") {
			return 0, core.E("tenant", message, nil)
		}
		for _, key := range []string{"count", "usage", "value"} {
			if count, ok := intField(envelope, key); ok {
				return count, nil
			}
		}
		if nested, ok := envelope["data"]; ok && nested != nil {
			nestedResult := core.JSONMarshal(nested)
			if !nestedResult.OK {
				return 0, core.E("tenant", "invalid count payload", nil)
			}
			return decodeCount(nestedResult.Value.([]byte))
		}
	}
	if value, err := strconv.Atoi(strings.TrimSpace(payload)); err == nil {
		return value, nil
	}
	var direct int
	if result := core.JSONUnmarshalString(payload, &direct); result.OK {
		return direct, nil
	}
	return 0, core.E("tenant", "invalid count payload", nil)
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
	if slug := workspaceSlugFromHost(host); slug != "" {
		if workspace, err := c.GetWorkspaceBySlug(ctx, slug); err == nil {
			return workspace, nil
		} else if !errors.Is(err, ErrWorkspaceNotFound) {
			return nil, err
		}
	}
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
	featureCode = normalizedFeatureCode(featureCode)
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
	code = normalizedFeatureCode(code)
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

func workspaceSlugFromHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return ""
	}
	if parsed, err := url.Parse("scheme://" + host); err == nil && parsed.Hostname() != "" {
		host = parsed.Hostname()
	}
	if dot := strings.Index(host, "."); dot > 0 {
		return host[:dot]
	}
	return host
}

func looksLikeEnvelope(fields map[string]any) bool {
	if len(fields) == 0 {
		return false
	}
	_, hasOK := fields["ok"]
	_, hasError := fields["error"]
	_, hasData := fields["data"]
	return hasOK || hasError || hasData
}

func stringField(fields map[string]any, key string) string {
	value, ok := fields[key]
	if !ok || value == nil {
		return ""
	}
	if typed, ok := value.(string); ok {
		return typed
	}
	return ""
}

func boolField(fields map[string]any, key string) bool {
	value, ok := fields[key]
	if !ok || value == nil {
		return false
	}
	typed, ok := value.(bool)
	return ok && typed
}

func intField(fields map[string]any, key string) (int, bool) {
	value, ok := fields[key]
	if !ok || value == nil {
		return 0, false
	}
	switch typed := value.(type) {
	case int:
		return typed, true
	case int8:
		return int(typed), true
	case int16:
		return int(typed), true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case uint:
		return int(typed), true
	case uint8:
		return int(typed), true
	case uint16:
		return int(typed), true
	case uint32:
		return int(typed), true
	case uint64:
		return int(typed), true
	case float32:
		return int(typed), true
	case float64:
		return int(typed), true
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(typed)); err == nil {
			return parsed, true
		}
	}
	return 0, false
}
