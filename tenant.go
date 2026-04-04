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
	"sync"
	"time"

	"dappco.re/go/core"
)

// Tenant is the root service. Register once per application instance.
// All tenant operations flow through this type.
//
//	ten := core.MustServiceFor[*tenant.Tenant](c, "tenant")
//	result := ten.Can(ctx, ws, "pages", 1)
type Tenant struct {
	*core.ServiceRuntime[TenantOptions]

	client        *TenantClient
	cache         *TenantCache
	entitlements  EntitlementService
	alertHandlers []AlertHandler
	lock          sync.Mutex
	alertState    map[string]usageAlertProgress
}

type usageAlertProgress struct {
	lastUsedCount         int
	highestTriggeredLevel int
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
func Register(c *core.Core) core.Result {
	if c == nil {
		return core.Result{Value: core.E("tenant", "core is nil", nil), OK: false}
	}
	options := tenantOptionsFromCore(c)
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

func tenantOptionsFromCore(c *core.Core) TenantOptions {
	options := TenantOptions{Timeout: 10 * time.Second}
	if c == nil || c.Config() == nil {
		return options
	}
	options.APIURL = configString(c, "api_url", "tenant.api_url")
	options.APIToken = configString(c, "api_token", "tenant.api_token")
	if timeout := configDuration(c, "timeout", "tenant.timeout"); timeout > 0 {
		options.Timeout = timeout
	}
	return options
}

func configString(c *core.Core, keys ...string) string {
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

func configDuration(c *core.Core, keys ...string) time.Duration {
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
func (t *Tenant) GetWorkspace(ctx context.Context, slug string) (*Workspace, error) {
	if t == nil {
		return nil, ErrWorkspaceNotFound
	}
	if t.cache != nil {
		if workspace, ok := t.cache.GetWorkspaceBySlug(slug); ok {
			return workspace, nil
		}
	}
	if t.client == nil {
		return nil, ErrWorkspaceNotFound
	}
	workspace, err := t.client.GetWorkspaceBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if t.cache != nil {
		_ = t.cache.SetWorkspace(workspace)
	}
	return workspace, nil
}

// GetWorkspaceByUUID resolves a workspace by UUID.
//
//	ws, err := ten.GetWorkspaceByUUID(ctx, "550e8400-...")
func (t *Tenant) GetWorkspaceByUUID(ctx context.Context, uuid string) (*Workspace, error) {
	if t == nil {
		return nil, ErrWorkspaceNotFound
	}
	if t.cache != nil {
		if workspace, ok := t.cache.GetWorkspace(uuid); ok {
			return workspace, nil
		}
	}
	if t.client == nil {
		return nil, ErrWorkspaceNotFound
	}
	workspace, err := t.client.GetWorkspaceByUUID(ctx, uuid)
	if err != nil {
		return nil, err
	}
	if t.cache != nil {
		_ = t.cache.SetWorkspace(workspace)
	}
	return workspace, nil
}

// GetWorkspaceByID resolves a workspace by integer ID from the local cache.
//
//	ws, err := ten.GetWorkspaceByID(ctx, 42)
func (t *Tenant) GetWorkspaceByID(ctx context.Context, id int64) (*Workspace, error) {
	if t == nil || t.cache == nil {
		return nil, ErrWorkspaceNotFound
	}
	if workspace, ok := t.cache.GetWorkspaceByID(id); ok {
		return workspace, nil
	}
	return nil, ErrWorkspaceNotFound
}

// GetWorkspaceBySubdomain resolves the workspace for an incoming hostname.
//
//	ws, err := ten.GetWorkspaceBySubdomain(ctx, r.Host)
func (t *Tenant) GetWorkspaceBySubdomain(ctx context.Context, host string) (*Workspace, error) {
	if t == nil {
		return nil, ErrWorkspaceNotFound
	}
	slug := host
	if dot := indexRune(host, '.'); dot > 0 {
		slug = host[:dot]
	}
	if slug != "" {
		if workspace, err := t.GetWorkspace(ctx, slug); err == nil {
			return workspace, nil
		}
	}
	if t.client == nil {
		return nil, ErrWorkspaceNotFound
	}
	workspace, err := t.client.GetWorkspaceBySubdomain(ctx, host)
	if err != nil {
		return nil, err
	}
	if t.cache != nil {
		_ = t.cache.SetWorkspace(workspace)
	}
	return workspace, nil
}

// Can checks whether ws can consume quantity units of featureCode.
//
//	result := ten.Can(ctx, ws, "pages", 1)
func (t *Tenant) Can(ctx context.Context, ws *Workspace, featureCode string, quantity int) EntitlementResult {
	if t == nil {
		return Deny(featureCode, "tenant not configured", nil, nil)
	}
	if t.entitlements == nil {
		t.entitlements = NewLocalEntitlementService(t.cache, t.client)
	}
	return t.entitlements.Can(ctx, ws, featureCode, quantity)
}

// RecordUsage records feature consumption for ws after a successful operation.
//
//	ten.RecordUsage(ctx, ws, "pages", 1, &userID, nil)
func (t *Tenant) RecordUsage(ctx context.Context, ws *Workspace, featureCode string, quantity int, userID *int64, metadata map[string]any) error {
	if t == nil {
		return ErrNoWorkspaceContext
	}
	if t.entitlements == nil {
		t.entitlements = NewLocalEntitlementService(t.cache, t.client)
	}
	if err := t.entitlements.RecordUsage(ctx, ws, featureCode, quantity, userID, metadata); err != nil {
		return err
	}
	result := t.Can(ctx, ws, featureCode, 0)
	t.CheckUsageAlerts(ws, featureCode, result)
	return nil
}

// GetUsageSummary returns all features with their current usage for ws.
//
//	items, err := ten.GetUsageSummary(ctx, ws)
func (t *Tenant) GetUsageSummary(ctx context.Context, ws *Workspace) ([]UsageSummaryItem, error) {
	if t == nil {
		return nil, ErrNoWorkspaceContext
	}
	if t.entitlements == nil {
		t.entitlements = NewLocalEntitlementService(t.cache, t.client)
	}
	return t.entitlements.GetUsageSummary(ctx, ws)
}

// InvalidateWorkspace drops the local cache for ws.
//
//	ten.InvalidateWorkspace(ws.UUID)
func (t *Tenant) InvalidateWorkspace(wsUUID string) {
	if t == nil {
		return
	}
	if t.entitlements != nil {
		t.entitlements.InvalidateWorkspace(wsUUID)
	}
	if t.cache != nil {
		_ = t.cache.InvalidateWorkspace(wsUUID)
	}
	t.lock.Lock()
	if len(t.alertState) > 0 {
		for key := range t.alertState {
			if hasAlertPrefix(key, wsUUID) {
				delete(t.alertState, key)
			}
		}
	}
	t.lock.Unlock()
}

// OnUsageAlert registers a handler invoked when a usage threshold is crossed.
// Multiple handlers can be registered; all fire in registration order.
//
//	ten.OnUsageAlert(func(a tenant.UsageAlert) { notify(a.WorkspaceUUID, a.Threshold) })
func (t *Tenant) OnUsageAlert(h AlertHandler) {
	if t == nil || h == nil {
		return
	}
	t.lock.Lock()
	t.alertHandlers = append(t.alertHandlers, h)
	t.lock.Unlock()
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
	if t == nil || ws == nil || result.Limit == nil || result.Used == nil {
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

	t.lock.Lock()
	state := t.alertState[key]
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
	if t.alertState == nil {
		t.alertState = map[string]usageAlertProgress{}
	}
	t.alertState[key] = state
	handlers := append([]AlertHandler(nil), t.alertHandlers...)
	t.lock.Unlock()

	for _, alert := range alerts {
		for _, handler := range handlers {
			handler(alert)
		}
	}
}

func indexRune(value string, target rune) int {
	for i, r := range value {
		if r == target {
			return i
		}
	}
	return -1
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
