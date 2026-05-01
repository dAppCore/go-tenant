// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"dappco.re/go"
)

// TenantClient calls the PHP REST API to read and mutate tenant data.
//
//	client := tenant.NewTenantClient("https://api.host.uk.com", token)
//	r := client.GetWorkspaceBySlug(ctx, "acme")
type TenantClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
	timeout    time.Duration
}

// ClientOption applies a TenantClient setting.
//
//	client := tenant.NewTenantClient("https://api.host.uk.com", "secret", tenant.WithTimeout(5*time.Second))
type ClientOption func(*TenantClient)

// NewTenantClient creates a new PHP API transport with the given base URL and bearer token.
//
//	client := tenant.NewTenantClient(url, token, tenant.WithTimeout(5*time.Second))
func NewTenantClient(baseURL, token string, opts ...ClientOption) *TenantClient {
	for core.HasSuffix(baseURL, "/") {
		baseURL = core.TrimSuffix(baseURL, "/")
	}
	client := &TenantClient{
		baseURL: baseURL,
		token:   token,
		timeout: 10 * time.Second,
	}
	for _, opt := range opts {
		opt(client)
	}
	client.ensureHTTPClient()
	return client
}

// WithTimeout overrides the default 10-second request timeout.
//
//	client := tenant.NewTenantClient(url, token, tenant.WithTimeout(5*time.Second))
func WithTimeout(d time.Duration) ClientOption {
	return func(c *TenantClient) {
		c.timeout = d
		c.ensureHTTPClient()
	}
}

func (c *TenantClient) request(ctx context.Context, method, path string, body any) core.Result {
	if c == nil {
		return core.Fail(core.E("tenant", "tenant client is nil", nil))
	}
	c.ensureHTTPClient()
	endpoint, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return core.Fail(core.E("tenant", "failed to build api request path", err))
	}
	var payload io.Reader
	if body != nil {
		result := core.JSONMarshal(body)
		if !result.OK {
			if err, ok := result.Value.(error); ok {
				return core.Fail(core.E("tenant", "failed to encode api request body", err))
			}
			return core.Fail(core.E("tenant", "failed to encode api request body", nil))
		}
		payload = core.NewBuffer(result.Value.([]byte))
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, payload)
	if err != nil {
		return core.Fail(core.E("tenant", "failed to create api request", err))
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if err == context.DeadlineExceeded || core.Contains(err.Error(), context.DeadlineExceeded.Error()) || core.Contains(err.Error(), "timeout") {
			return core.Fail(ErrClientTimeout)
		}
		return core.Fail(core.E("tenant", "api request failed", err))
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return core.Fail(core.E("tenant", "failed to read api response", err))
	}
	if resp.StatusCode >= 400 {
		if len(data) == 0 {
			return c.statusError(resp.StatusCode, path, "")
		}
		return c.statusError(resp.StatusCode, path, string(data))
	}
	return core.Ok(data)
}

func (c *TenantClient) statusError(status int, path, body string) core.Result {
	if status == http.StatusNotFound {
		if core.Contains(path, "/features/") {
			return core.Fail(ErrFeatureNotFound)
		}
		return core.Fail(ErrWorkspaceNotFound)
	}
	if body != "" {
		var envelope map[string]any
		if core.JSONUnmarshalString(body, &envelope).OK {
			if message := stringField(envelope, "error"); message != "" {
				return core.Fail(core.E("tenant", message, nil))
			}
		}
	}
	return core.Fail(core.E("tenant", http.StatusText(status), nil))
}

// decodeEnvelope decodes a PHP API response, handling both envelope and direct JSON formats.
// Envelope format: {"ok": true, "data": ...} or {"ok": false, "error": "message"}.
//
//	var workspace Workspace
//	r := decodeEnvelope(responseBytes, &workspace)
func decodeEnvelope(data []byte, target any) core.Result {
	if len(data) == 0 {
		return core.Fail(io.EOF)
	}
	payload := string(data)
	var envelope map[string]any
	if core.JSONUnmarshalString(payload, &envelope).OK && looksLikeEnvelope(envelope) {
		if message := stringField(envelope, "error"); message != "" && !boolField(envelope, "ok") {
			return core.Fail(core.E("tenant", message, nil))
		}
		if nested, ok := envelope["data"]; ok && nested != nil {
			nestedResult := core.JSONMarshal(nested)
			if !nestedResult.OK {
				return core.Fail(core.E("tenant", "invalid api payload", nil))
			}
			nestedJSON := string(nestedResult.Value.([]byte))
			if result := core.JSONUnmarshalString(nestedJSON, target); result.OK {
				return core.Ok(target)
			}
			return core.Fail(core.E("tenant", "invalid api payload", nil))
		}
	}
	if result := core.JSONUnmarshalString(payload, target); result.OK {
		return core.Ok(target)
	}
	return core.Fail(core.E("tenant", "invalid api payload", nil))
}

// decodeCount extracts a numeric count from a PHP API response.
// Supports envelope format ({"count": N}), direct integer, and string representations.
//
//	r := decodeCount([]byte(`{"ok":true,"count":7}`))  // Value is 7
//	r := decodeCount([]byte(`42`))                     // Value is 42
func decodeCount(data []byte) core.Result {
	if len(data) == 0 {
		return core.Fail(io.EOF)
	}
	payload := string(data)
	var envelope map[string]any
	if core.JSONUnmarshalString(payload, &envelope).OK && looksLikeEnvelope(envelope) {
		if message := stringField(envelope, "error"); message != "" && !boolField(envelope, "ok") {
			return core.Fail(core.E("tenant", message, nil))
		}
		for _, key := range []string{"count", "usage", "value"} {
			if count, ok := intField(envelope, key); ok {
				return core.Ok(count)
			}
		}
		if nested, ok := envelope["data"]; ok && nested != nil {
			nestedResult := core.JSONMarshal(nested)
			if !nestedResult.OK {
				return core.Fail(core.E("tenant", "invalid count payload", nil))
			}
			return decodeCount(nestedResult.Value.([]byte))
		}
	}
	if value, err := strconv.Atoi(core.Trim(payload)); err == nil {
		return core.Ok(value)
	}
	var direct int
	if result := core.JSONUnmarshalString(payload, &direct); result.OK {
		return core.Ok(direct)
	}
	return core.Fail(core.E("tenant", "invalid count payload", nil))
}

func (c *TenantClient) ensureHTTPClient() {
	if c == nil {
		return
	}
	if c.timeout <= 0 {
		c.timeout = 10 * time.Second
	}
	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: c.timeout}
		return
	}
	c.httpClient.Timeout = c.timeout
}

// GetWorkspaceBySlug fetches a workspace by its slug.
//
//	r := client.GetWorkspaceBySlug(ctx, "acme")
func (c *TenantClient) GetWorkspaceBySlug(ctx context.Context, slug string) core.Result {
	r := c.request(ctx, http.MethodGet, "/api/v1/workspaces/"+url.PathEscape(slug), nil)
	if !r.OK {
		return r
	}
	var workspace Workspace
	if decoded := decodeEnvelope(r.Value.([]byte), &workspace); !decoded.OK {
		return decoded
	}
	return core.Ok(&workspace)
}

// GetWorkspaceByUUID fetches a workspace by UUID.
//
//	workspace, err := client.GetWorkspaceByUUID(ctx, "550e8400-...")
func (c *TenantClient) GetWorkspaceByUUID(ctx context.Context, uuid string) core.Result {
	r := c.request(ctx, http.MethodGet, "/api/v1/workspaces/uuid/"+url.PathEscape(uuid), nil)
	if !r.OK {
		return r
	}
	var workspace Workspace
	if decoded := decodeEnvelope(r.Value.([]byte), &workspace); !decoded.OK {
		return decoded
	}
	return core.Ok(&workspace)
}

// GetWorkspaceByID fetches a workspace by integer ID.
//
//	workspace, err := client.GetWorkspaceByID(ctx, 42)
func (c *TenantClient) GetWorkspaceByID(ctx context.Context, id int64) core.Result {
	r := c.request(ctx, http.MethodGet, "/api/v1/workspaces/id/"+strconv.FormatInt(id, 10), nil)
	if !r.OK {
		return r
	}
	var workspace Workspace
	if decoded := decodeEnvelope(r.Value.([]byte), &workspace); !decoded.OK {
		return decoded
	}
	return core.Ok(&workspace)
}

// GetWorkspaceBySubdomain resolves a hostname to a workspace.
// It checks the slug first, then falls back to the domain-prefix endpoint.
//
//	workspace, err := tenantClient.GetWorkspaceBySubdomain(ctx, "acme.host.uk.com")
func (c *TenantClient) GetWorkspaceBySubdomain(ctx context.Context, host string) core.Result {
	if slug := workspaceSlugFromHost(host); slug != "" {
		if r := c.GetWorkspaceBySlug(ctx, slug); r.OK {
			return r
		} else if r.Value != ErrWorkspaceNotFound {
			return r
		}
	}
	r := c.request(ctx, http.MethodGet, "/api/v1/workspaces/subdomain/"+url.PathEscape(host), nil)
	if !r.OK {
		return r
	}
	var workspace Workspace
	if decoded := decodeEnvelope(r.Value.([]byte), &workspace); !decoded.OK {
		return decoded
	}
	return core.Ok(&workspace)
}

// GetUser fetches the authenticated user by the bearer token on the client.
//
//	user, err := client.GetUser(ctx)
func (c *TenantClient) GetUser(ctx context.Context) core.Result {
	r := c.request(ctx, http.MethodGet, "/api/v1/user", nil)
	if !r.OK {
		return r
	}
	var user User
	if decoded := decodeEnvelope(r.Value.([]byte), &user); !decoded.OK {
		return decoded
	}
	return core.Ok(&user)
}

// GetPackagesForWorkspace returns all active packages assigned to the workspace.
//
//	packages, err := client.GetPackagesForWorkspace(ctx, ws.UUID)
func (c *TenantClient) GetPackagesForWorkspace(ctx context.Context, wsUUID string) core.Result {
	r := c.request(ctx, http.MethodGet, "/api/v1/workspaces/"+url.PathEscape(wsUUID)+"/packages", nil)
	if !r.OK {
		return r
	}
	var packages []Package
	if decoded := decodeEnvelope(r.Value.([]byte), &packages); !decoded.OK {
		return decoded
	}
	return core.Ok(packages)
}

// GetBoostsForWorkspace returns all active, usable boosts for the workspace.
//
//	boosts, err := client.GetBoostsForWorkspace(ctx, ws.UUID)
func (c *TenantClient) GetBoostsForWorkspace(ctx context.Context, wsUUID string) core.Result {
	r := c.request(ctx, http.MethodGet, "/api/v1/workspaces/"+url.PathEscape(wsUUID)+"/boosts", nil)
	if !r.OK {
		return r
	}
	var boosts []Boost
	if decoded := decodeEnvelope(r.Value.([]byte), &boosts); !decoded.OK {
		return decoded
	}
	return core.Ok(boosts)
}

// GetCurrentUsage returns total usage count for wsUUID+featureCode since reset boundary.
//
//	used, err := client.GetCurrentUsage(ctx, ws.UUID, "pages")
func (c *TenantClient) GetCurrentUsage(ctx context.Context, wsUUID, featureCode string) core.Result {
	featureCode = normalizedFeatureCode(featureCode)
	r := c.request(ctx, http.MethodGet, "/api/v1/workspaces/"+url.PathEscape(wsUUID)+"/usage/"+url.PathEscape(featureCode), nil)
	if !r.OK {
		return r
	}
	return decodeCount(r.Value.([]byte))
}

// RecordUsage POSTs a usage record to the PHP API.
//
//	err := client.RecordUsage(ctx, ws.UUID, "pages", 1, &userID, nil)
func (c *TenantClient) RecordUsage(ctx context.Context, wsUUID, featureCode string, quantity int, userID *int64, metadata map[string]any) core.Result {
	record := newUsageRecord(0, featureCode, quantity, userID, metadata)
	payload := record.payload()
	return c.request(ctx, http.MethodPost, "/api/v1/workspaces/"+url.PathEscape(wsUUID)+"/usage", payload)
}

// GetFeature fetches a single feature definition by code.
//
//	feature, err := tenantClient.GetFeature(ctx, "pages")
func (c *TenantClient) GetFeature(ctx context.Context, code string) core.Result {
	code = normalizedFeatureCode(code)
	r := c.request(ctx, http.MethodGet, "/api/v1/features/"+url.PathEscape(code), nil)
	if !r.OK {
		return r
	}
	var feature Feature
	if decoded := decodeEnvelope(r.Value.([]byte), &feature); !decoded.OK {
		return decoded
	}
	return core.Ok(&feature)
}

// workspaceSlugFromHost extracts the subdomain slug from a hostname.
//
//	workspaceSlugFromHost("acme.host.uk.com")  // "acme"
//	workspaceSlugFromHost("localhost")          // "localhost"
//	workspaceSlugFromHost("")                   // ""
func workspaceSlugFromHost(host string) string {
	host = core.Lower(core.Trim(host))
	if host == "" {
		return ""
	}
	if parsed, err := url.Parse("scheme://" + host); err == nil && parsed.Hostname() != "" {
		host = parsed.Hostname()
	}
	parts := core.SplitN(host, ".", 2)
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return host
}

// looksLikeEnvelope checks whether a decoded JSON map has API envelope structure.
//
//	looksLikeEnvelope(map[string]any{"ok": true, "data": ws})  // true
//	looksLikeEnvelope(map[string]any{"slug": "acme"})          // false
func looksLikeEnvelope(fields map[string]any) bool {
	if len(fields) == 0 {
		return false
	}
	_, hasOK := fields["ok"]
	_, hasError := fields["error"]
	_, hasData := fields["data"]
	return hasOK || hasError || hasData
}

// stringField extracts a string value from a decoded JSON map.
//
//	stringField(map[string]any{"error": "not found"}, "error")  // "not found"
//	stringField(map[string]any{"error": 42}, "error")           // ""
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

// boolField extracts a boolean value from a decoded JSON map.
//
//	boolField(map[string]any{"ok": true}, "ok")  // true
//	boolField(map[string]any{"ok": "yes"}, "ok") // false
func boolField(fields map[string]any, key string) bool {
	value, ok := fields[key]
	if !ok || value == nil {
		return false
	}
	typed, ok := value.(bool)
	return ok && typed
}

// intField extracts a numeric value from a decoded JSON map as int.
// Handles int, float64, and string representations.
//
//	intField(map[string]any{"count": 7.0}, "count")   // 7, true
//	intField(map[string]any{"count": "42"}, "count")  // 42, true
//	intField(map[string]any{}, "count")               // 0, false
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
		if parsed, err := strconv.Atoi(core.Trim(typed)); err == nil {
			return parsed, true
		}
	}
	return 0, false
}
