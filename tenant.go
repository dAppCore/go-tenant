// SPDX-License-Identifier: EUPL-1.2

// Package tenant provides multi-tenancy, workspace isolation, user identity,
// feature entitlements, and usage tracking for CoreGO services.
//
// The Go layer is a consumer — PHP owns the database. Go calls PHP's REST API
// for all mutations and queries, then caches the results locally via go-store.
//
//	ten := core.MustServiceFor[*tenant.Tenant](c, "tenant")
//	result := ten.Can(ctx, ws, "pages", 1)
//	if result.IsDenied() { return core.E("pages", result.Reason, nil) }
package tenant

import (
	"context"
	"time"
)

// Tenant is the root service. Register once per application instance.
// All tenant operations flow through this type.
//
//	ten := core.MustServiceFor[*tenant.Tenant](c, "tenant")
//	result := ten.Can(ctx, ws, "pages", 1)
type Tenant struct {
	client        *TenantClient
	cache         *TenantCache
	entitlements  EntitlementService
	alertHandlers []AlertHandler
}

// TenantOptions configures the tenant service via Core config.
//
//	opts := tenant.TenantOptions{APIURL: "https://api.host.uk.com", APIToken: "bearer-token"}
type TenantOptions struct {
	APIURL   string        `json:"api_url"`   // PHP API base URL
	APIToken string        `json:"api_token"` // Bearer token
	Timeout  time.Duration `json:"timeout"`   // HTTP timeout (default 10s)
}

// Register is the Core service factory. Called by core.WithService.
//
//	core.New(core.WithService(tenant.Register))
func Register() {
	// TODO: implement — create Tenant, wire client + cache + entitlements
}

// GetWorkspace resolves a workspace by slug. Checks cache first, then PHP API.
//
//	ws, err := ten.GetWorkspace(ctx, "acme")
func (t *Tenant) GetWorkspace(ctx context.Context, slug string) (*Workspace, error) {
	// TODO: implement
	return nil, ErrWorkspaceNotFound
}

// GetWorkspaceByUUID resolves a workspace by UUID.
//
//	ws, err := ten.GetWorkspaceByUUID(ctx, "550e8400-...")
func (t *Tenant) GetWorkspaceByUUID(ctx context.Context, uuid string) (*Workspace, error) {
	// TODO: implement
	return nil, ErrWorkspaceNotFound
}

// GetWorkspaceBySubdomain resolves the workspace for an incoming hostname.
//
//	ws, err := ten.GetWorkspaceBySubdomain(ctx, r.Host)
func (t *Tenant) GetWorkspaceBySubdomain(ctx context.Context, host string) (*Workspace, error) {
	// TODO: implement
	return nil, ErrWorkspaceNotFound
}

// Can checks whether ws can consume quantity units of featureCode.
//
//	result := ten.Can(ctx, ws, "pages", 1)
func (t *Tenant) Can(ctx context.Context, ws *Workspace, featureCode string, quantity int) EntitlementResult {
	// TODO: implement — delegate to t.entitlements.Can
	return Deny(featureCode, "not implemented", nil, nil)
}

// RecordUsage records feature consumption for ws after a successful operation.
//
//	ten.RecordUsage(ctx, ws, "pages", 1, &userID, nil)
func (t *Tenant) RecordUsage(ctx context.Context, ws *Workspace, featureCode string, quantity int, userID *int64, metadata map[string]any) error {
	// TODO: implement — delegate to t.entitlements.RecordUsage, then CheckUsageAlerts
	return nil
}

// GetUsageSummary returns all features with their current usage for ws.
//
//	items, err := ten.GetUsageSummary(ctx, ws)
func (t *Tenant) GetUsageSummary(ctx context.Context, ws *Workspace) ([]UsageSummaryItem, error) {
	// TODO: implement
	return nil, nil
}

// InvalidateWorkspace drops the local cache for ws.
//
//	ten.InvalidateWorkspace(ws.UUID)
func (t *Tenant) InvalidateWorkspace(wsUUID string) {
	// TODO: implement — delegate to t.entitlements.InvalidateWorkspace
}

// OnUsageAlert registers a handler invoked when a usage threshold is crossed.
// Multiple handlers can be registered; all fire in registration order.
//
//	ten.OnUsageAlert(func(a tenant.UsageAlert) { notify(a.WorkspaceUUID, a.Threshold) })
func (t *Tenant) OnUsageAlert(h AlertHandler) {
	t.alertHandlers = append(t.alertHandlers, h)
}

// Scope returns a configured WorkspaceScope for middleware wiring.
//
//	router.Use(ten.Scope().Middleware())
func (t *Tenant) Scope() *WorkspaceScope {
	return NewWorkspaceScope(t)
}

// CheckUsageAlerts inspects an EntitlementResult after RecordUsage and fires
// registered AlertHandlers if a threshold is newly crossed.
// Called internally by Tenant.RecordUsage — not usually called directly.
//
//	ten.CheckUsageAlerts(ws, "pages", result)
func (t *Tenant) CheckUsageAlerts(ws *Workspace, featureCode string, result EntitlementResult) {
	// TODO: implement — check thresholds 80/90/100, fire handlers
}
