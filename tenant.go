// SPDX-License-Identifier: EUPL-1.2

// Package tenant provides multi-tenancy, workspace isolation, user identity,
// feature entitlements, and usage tracking for CoreGO services.
//
// The Go layer is a consumer — PHP owns the database. Go calls PHP's REST API
// for all mutations and queries, then caches the results locally via go-store.
//
//	tenantService, _ := core.ServiceFor[*tenant.Tenant](c, "tenant")
//	result := tenantService.Can(ctx, workspace, "pages", 1)
//	if result.IsDenied() { return core.E("pages", result.Reason, nil) }
package tenant

import (
	"context"
	"sync"
	"time"

	"dappco.re/go/core"
)

// Tenant is the root service. Register once per application instance.
// All tenant operations flow through this type.
//
//	ten, _ := core.ServiceFor[*tenant.Tenant](c, "tenant")
//	result := ten.Can(ctx, ws, "pages", 1)
type Tenant struct {
	*core.ServiceRuntime[TenantOptions]

	client        *TenantClient
	cache         *TenantCache
	entitlements  EntitlementService
	alertHandlers []AlertHandler
	lock          sync.Mutex
	alertState    map[string]usageAlertTracker
}

type usageAlertTracker struct {
	lastUsedCount         int
	highestTriggeredLevel int
}

// TenantOptions configures the tenant service via Core config.
//
//	opts := tenant.TenantOptions{
//		APIURL:   "https://api.host.uk.com",
//		APIToken: "bearer-token",
//		Timeout:  5 * time.Second,
//	}
type TenantOptions struct {
	APIURL   string        `json:"api_url"`   // PHP API base URL
	APIToken string        `json:"api_token"` // Bearer token
	Timeout  time.Duration `json:"timeout"`   // HTTP timeout (default 10s)
}

// Register is the Core service factory. Called by core.WithService.
//
//	core.New(core.WithService(tenant.Register))
func Register(c *core.Core) core.Result {
	if c == nil {
		return core.Result{Value: core.E("tenant", "core is nil", nil), OK: false}
	}
	options := tenantOptionsFromCoreConfig(c)
	service := &Tenant{
		ServiceRuntime: core.NewServiceRuntime(c, options),
	}
	if options.APIURL != "" && options.APIToken != "" {
		service.client = NewTenantClient(options.APIURL, options.APIToken, WithTimeout(options.Timeout))
	}
	service.cache = NewTenantCache(nil)
	service.entitlements = NewLocalEntitlementService(service.cache, service.client)
	return core.Result{Value: service, OK: true}
}

func tenantOptionsFromCoreConfig(c *core.Core) TenantOptions {
	options := TenantOptions{Timeout: 10 * time.Second}
	if c == nil || c.Config() == nil {
		return options
	}
	options.APIURL = coreConfigStringValue(c, "api_url", "tenant.api_url")
	options.APIToken = coreConfigStringValue(c, "api_token", "tenant.api_token")
	if timeout := coreConfigDurationValue(c, "timeout", "tenant.timeout"); timeout > 0 {
		options.Timeout = timeout
	}
	return options
}

func coreConfigStringValue(c *core.Core, keys ...string) string {
	if c == nil || c.Config() == nil {
		return ""
	}
	for _, key := range keys {
		if value := c.Config().String(key); value != "" {
			return value
		}
	}
	return ""
}

func coreConfigDurationValue(c *core.Core, keys ...string) time.Duration {
	if c == nil || c.Config() == nil {
		return 0
	}
	for _, key := range keys {
		value := c.Config().Get(key)
		if !value.OK || value.Value == nil {
			continue
		}
		switch typed := value.Value.(type) {
		case time.Duration:
			return typed
		case string:
			if parsed, err := time.ParseDuration(typed); err == nil {
				return parsed
			}
		case int:
			return time.Duration(typed) * time.Second
		case int64:
			return time.Duration(typed) * time.Second
		case float64:
			return time.Duration(typed) * time.Second
		}
	}
	return 0
}

// GetWorkspace resolves a workspace by slug. Checks cache first, then PHP API.
//
//	ws, err := ten.GetWorkspace(ctx, "acme")
func (tenantService *Tenant) GetWorkspace(ctx context.Context, slug string) (*Workspace, error) {
	if tenantService == nil {
		return nil, ErrWorkspaceNotFound
	}
	if tenantService.cache != nil {
		if workspace, ok := tenantService.cache.GetWorkspaceBySlug(slug); ok {
			return workspace, nil
		}
	}
	if tenantService.client == nil {
		return nil, ErrWorkspaceNotFound
	}
	workspace, err := tenantService.client.GetWorkspaceBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if tenantService.cache != nil {
		_ = tenantService.cache.SetWorkspace(workspace)
	}
	return workspace, nil
}

// GetWorkspaceByUUID resolves a workspace by UUID.
//
//	ws, err := ten.GetWorkspaceByUUID(ctx, "550e8400-...")
func (tenantService *Tenant) GetWorkspaceByUUID(ctx context.Context, uuid string) (*Workspace, error) {
	if tenantService == nil {
		return nil, ErrWorkspaceNotFound
	}
	if tenantService.cache != nil {
		if workspace, ok := tenantService.cache.GetWorkspace(uuid); ok {
			return workspace, nil
		}
	}
	if tenantService.client == nil {
		return nil, ErrWorkspaceNotFound
	}
	workspace, err := tenantService.client.GetWorkspaceByUUID(ctx, uuid)
	if err != nil {
		return nil, err
	}
	if tenantService.cache != nil {
		_ = tenantService.cache.SetWorkspace(workspace)
	}
	return workspace, nil
}

// GetWorkspaceByID resolves a workspace by integer ID.
//
//	ws, err := ten.GetWorkspaceByID(ctx, 42)
func (tenantService *Tenant) GetWorkspaceByID(ctx context.Context, id int64) (*Workspace, error) {
	if tenantService == nil {
		return nil, ErrWorkspaceNotFound
	}
	if tenantService.cache != nil {
		if workspace, ok := tenantService.cache.GetWorkspaceByID(id); ok {
			return workspace, nil
		}
	}
	if tenantService.client == nil {
		return nil, ErrWorkspaceNotFound
	}
	workspace, err := tenantService.client.GetWorkspaceByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tenantService.cache != nil {
		_ = tenantService.cache.SetWorkspace(workspace)
	}
	return workspace, nil
}

// GetUser resolves the authenticated user for ctx.
//
//	user, err := ten.GetUser(ctx)
func (tenantService *Tenant) GetUser(ctx context.Context) (*User, error) {
	if tenantService == nil {
		return nil, ErrNoUserContext
	}
	if user, err := UserFromCtx(ctx); err == nil && user != nil {
		if tenantService.cache != nil {
			if cached, ok := tenantService.cache.GetUser(user.UUID); ok {
				return cached, nil
			}
			_ = tenantService.cache.SetUser(user)
		}
		return cloneUser(user), nil
	}
	if tenantService.client == nil {
		return nil, ErrNoUserContext
	}
	user, err := tenantService.client.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if tenantService.cache != nil {
		_ = tenantService.cache.SetUser(user)
	}
	return user, nil
}

// GetWorkspaceBySubdomain resolves the workspace for an incoming hostname.
//
//	workspace, err := tenantService.GetWorkspaceBySubdomain(ctx, "acme.host.uk.com")
func (tenantService *Tenant) GetWorkspaceBySubdomain(ctx context.Context, host string) (*Workspace, error) {
	if tenantService == nil {
		return nil, ErrWorkspaceNotFound
	}
	slug := workspaceSlugFromHost(host)
	if slug != "" {
		if workspace, err := tenantService.GetWorkspace(ctx, slug); err == nil {
			return workspace, nil
		}
	}
	if tenantService.client == nil {
		return nil, ErrWorkspaceNotFound
	}
	workspace, err := tenantService.client.GetWorkspaceBySubdomain(ctx, host)
	if err != nil {
		return nil, err
	}
	if tenantService.cache != nil {
		_ = tenantService.cache.SetWorkspace(workspace)
	}
	return workspace, nil
}

// Can checks whether ws can consume quantity units of featureCode.
//
//	result := tenantService.Can(ctx, workspace, "pages", 1)
func (tenantService *Tenant) Can(ctx context.Context, ws *Workspace, featureCode string, quantity int) EntitlementResult {
	if tenantService == nil {
		return Deny(featureCode, "tenant not configured", nil, nil)
	}
	if tenantService.entitlements == nil {
		tenantService.entitlements = NewLocalEntitlementService(tenantService.cache, tenantService.client)
	}
	return tenantService.entitlements.Can(ctx, ws, featureCode, quantity)
}

// RecordUsage records feature consumption for ws after a successful operation.
//
//	tenantService.RecordUsage(ctx, workspace, "pages", 1, &userID, nil)
func (tenantService *Tenant) RecordUsage(ctx context.Context, ws *Workspace, featureCode string, quantity int, userID *int64, metadata map[string]any) error {
	if tenantService == nil {
		return ErrNoWorkspaceContext
	}
	if tenantService.entitlements == nil {
		tenantService.entitlements = NewLocalEntitlementService(tenantService.cache, tenantService.client)
	}
	if err := tenantService.entitlements.RecordUsage(ctx, ws, featureCode, quantity, userID, metadata); err != nil {
		return err
	}
	result := tenantService.Can(ctx, ws, featureCode, 0)
	tenantService.CheckUsageAlerts(ws, featureCode, result)
	return nil
}

// GetUsageSummary returns all features with their current usage for ws.
//
//	items, err := tenantService.GetUsageSummary(ctx, workspace)
func (tenantService *Tenant) GetUsageSummary(ctx context.Context, ws *Workspace) ([]UsageSummaryItem, error) {
	if tenantService == nil {
		return nil, ErrNoWorkspaceContext
	}
	if tenantService.entitlements == nil {
		tenantService.entitlements = NewLocalEntitlementService(tenantService.cache, tenantService.client)
	}
	return tenantService.entitlements.GetUsageSummary(ctx, ws)
}

// InvalidateWorkspace drops the local cache for ws.
//
//	tenantService.InvalidateWorkspace("workspace-uuid")
func (tenantService *Tenant) InvalidateWorkspace(wsUUID string) {
	if tenantService == nil {
		return
	}
	if tenantService.entitlements != nil {
		tenantService.entitlements.InvalidateWorkspace(wsUUID)
	}
	if tenantService.cache != nil {
		_ = tenantService.cache.InvalidateWorkspace(wsUUID)
	}
	tenantService.lock.Lock()
	if len(tenantService.alertState) > 0 {
		for key := range tenantService.alertState {
			if hasAlertPrefix(key, wsUUID) {
				delete(tenantService.alertState, key)
			}
		}
	}
	tenantService.lock.Unlock()
}

// OnUsageAlert registers a handler invoked when a usage threshold is crossed.
// Multiple handlers can be registered; all fire in registration order.
//
//	tenantService.OnUsageAlert(func(alert tenant.UsageAlert) { notify(alert.WorkspaceUUID, alert.Threshold) })
func (tenantService *Tenant) OnUsageAlert(h AlertHandler) {
	if tenantService == nil || h == nil {
		return
	}
	tenantService.lock.Lock()
	tenantService.alertHandlers = append(tenantService.alertHandlers, h)
	tenantService.lock.Unlock()
}

// Scope returns a configured WorkspaceScope for middleware wiring.
//
//	router.Use(tenantService.Scope().Middleware())
func (tenantService *Tenant) Scope() *WorkspaceScope {
	return NewWorkspaceScope(tenantService)
}

// CheckUsageAlerts inspects an EntitlementResult after RecordUsage and fires
// registered AlertHandlers if a threshold is newly crossed.
// Called internally by Tenant.RecordUsage — not usually called directly.
//
//	tenantService.CheckUsageAlerts(workspace, "pages", result)
func (tenantService *Tenant) CheckUsageAlerts(ws *Workspace, featureCode string, result EntitlementResult) {
	if tenantService == nil || ws == nil || result.Limit == nil || result.Used == nil {
		return
	}
	pct := result.UsagePercent()
	if pct == nil {
		return
	}
	percentage := *pct
	used := *result.Used
	limit := *result.Limit
	key := alertStateKey(ws.UUID, featureCode)
	now := time.Now()

	tenantService.lock.Lock()
	state := tenantService.alertState[key]
	if used < state.lastUsedCount {
		state.highestTriggeredLevel = 0
	}
	alerts := make([]UsageAlert, 0, 3)
	for _, threshold := range []int{AlertThresholdWarning, AlertThresholdCritical, AlertThresholdLimit} {
		if percentage >= float64(threshold) && threshold > state.highestTriggeredLevel {
			alerts = append(alerts, UsageAlert{
				WorkspaceUUID: ws.UUID,
				FeatureCode:   featureCode,
				Threshold:     threshold,
				Used:          used,
				Limit:         limit,
				Percentage:    percentage,
				TriggeredAt:   now,
			})
			state.highestTriggeredLevel = threshold
		}
	}
	state.lastUsedCount = used
	if tenantService.alertState == nil {
		tenantService.alertState = map[string]usageAlertTracker{}
	}
	tenantService.alertState[key] = state
	handlers := append([]AlertHandler(nil), tenantService.alertHandlers...)
	tenantService.lock.Unlock()

	for _, alert := range alerts {
		for _, handler := range handlers {
			func() {
				defer func() {
					_ = recover()
				}()
				handler(alert)
			}()
		}
	}
}

func alertStateKey(wsUUID, featureCode string) string {
	return wsUUID + "\x00" + featureCode
}

func hasAlertPrefix(key, wsUUID string) bool {
	if len(key) < len(wsUUID)+1 {
		return false
	}
	return key[:len(wsUUID)] == wsUUID && key[len(wsUUID)] == '\x00'
}
