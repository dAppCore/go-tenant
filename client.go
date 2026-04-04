// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"context"
	"net/http"
	"time"
)

// TenantClient calls the PHP REST API to read and mutate tenant data.
// Reads are cached by TenantCache; the client is called on cache miss.
//
//	client := tenant.NewTenantClient("https://api.host.uk.com", token)
//	ws, err := client.GetWorkspaceBySlug(ctx, "acme")
type TenantClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
	timeout    time.Duration
}

// ClientOption configures TenantClient behaviour.
type ClientOption func(*TenantClient)

// NewTenantClient creates a new PHP API transport with the given base URL and bearer token.
//
//	client := tenant.NewTenantClient("https://api.host.uk.com", token)
//	client := tenant.NewTenantClient(url, token, tenant.WithTimeout(5*time.Second))
func NewTenantClient(baseURL, token string, opts ...ClientOption) *TenantClient {
	// TODO: implement
	return nil
}

// WithTimeout overrides the default 10-second request timeout.
//
//	client := tenant.NewTenantClient(url, token, tenant.WithTimeout(5*time.Second))
func WithTimeout(d time.Duration) ClientOption {
	return func(c *TenantClient) {
		c.timeout = d
	}
}

// GetWorkspaceBySlug fetches a workspace by its slug.
//
//	ws, err := client.GetWorkspaceBySlug(ctx, "acme")
func (c *TenantClient) GetWorkspaceBySlug(ctx context.Context, slug string) (*Workspace, error) {
	// TODO: implement — GET /api/v1/workspaces/{slug}
	return nil, ErrWorkspaceNotFound
}

// GetWorkspaceByUUID fetches a workspace by UUID.
//
//	ws, err := client.GetWorkspaceByUUID(ctx, "550e8400-...")
func (c *TenantClient) GetWorkspaceByUUID(ctx context.Context, uuid string) (*Workspace, error) {
	// TODO: implement — GET /api/v1/workspaces/uuid/{uuid}
	return nil, ErrWorkspaceNotFound
}

// GetWorkspaceBySubdomain resolves a hostname to a workspace.
// Checks slug match first, then domain prefix match (matches PHP's WorkspaceService).
//
//	ws, err := client.GetWorkspaceBySubdomain(ctx, "acme.host.uk.com")
func (c *TenantClient) GetWorkspaceBySubdomain(ctx context.Context, host string) (*Workspace, error) {
	// TODO: implement — GET /api/v1/workspaces/subdomain/{host}
	return nil, ErrWorkspaceNotFound
}

// GetUser fetches the authenticated user by the bearer token on the client.
//
//	user, err := client.GetUser(ctx)
func (c *TenantClient) GetUser(ctx context.Context) (*User, error) {
	// TODO: implement — GET /api/v1/user
	return nil, ErrNoUserContext
}

// GetPackagesForWorkspace returns all active packages assigned to the workspace.
//
//	pkgs, err := client.GetPackagesForWorkspace(ctx, ws.UUID)
func (c *TenantClient) GetPackagesForWorkspace(ctx context.Context, wsUUID string) ([]Package, error) {
	// TODO: implement — GET /api/v1/workspaces/{uuid}/packages
	return nil, nil
}

// GetBoostsForWorkspace returns all active, usable boosts for the workspace.
//
//	boosts, err := client.GetBoostsForWorkspace(ctx, ws.UUID)
func (c *TenantClient) GetBoostsForWorkspace(ctx context.Context, wsUUID string) ([]Boost, error) {
	// TODO: implement — GET /api/v1/workspaces/{uuid}/boosts
	return nil, nil
}

// GetCurrentUsage returns total usage count for wsUUID+featureCode since reset boundary.
//
//	used, err := client.GetCurrentUsage(ctx, ws.UUID, "pages")
func (c *TenantClient) GetCurrentUsage(ctx context.Context, wsUUID, featureCode string) (int, error) {
	// TODO: implement — GET /api/v1/workspaces/{uuid}/usage/{code}
	return 0, nil
}

// RecordUsage POSTs a usage record to the PHP API.
//
//	err := client.RecordUsage(ctx, ws.UUID, "pages", 1, &userID, nil)
func (c *TenantClient) RecordUsage(ctx context.Context, wsUUID, featureCode string, quantity int, userID *int64, metadata map[string]any) error {
	// TODO: implement — POST /api/v1/workspaces/{uuid}/usage
	return nil
}

// GetFeature fetches a single feature definition by code.
//
//	feat, err := client.GetFeature(ctx, "pages")
func (c *TenantClient) GetFeature(ctx context.Context, code string) (*Feature, error) {
	// TODO: implement — GET /api/v1/features/{code}
	return nil, ErrFeatureNotFound
}
