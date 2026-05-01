// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"context"

	"dappco.re/go"
)

func testInt(value int) *int { return &value }

func testInt64(value int64) *int64 { return &value }

func testString(value string) *string { return &value }

func testWorkspace() *Workspace {
	return &Workspace{ID: 7, UUID: "uuid-7", Slug: "acme", Name: "Acme", IsActive: true}
}

func testUser() *User {
	return &User{ID: 9, UUID: "user-9", Email: "ada@example.uk", Tier: TierApollo}
}

func testLimitFeature() *Feature {
	return &Feature{Code: "pages", Name: "Pages", Type: FeatureTypeLimit, IsActive: true}
}

func testCache(t *core.T) (*TenantCache, *Workspace) {
	cache := NewTenantCache(nil)
	workspace := testWorkspace()
	requireResultOK(t, cache.SetWorkspace(workspace))
	requireResultOK(t, cache.SetFeature(testLimitFeature()))
	requireResultOK(t, cache.SetPackages(workspace.UUID, []Package{{
		Code: "starter", IsActive: true, Features: []PackageFeature{{FeatureCode: "pages", LimitValue: testInt(10)}},
	}}))
	requireResultOK(t, cache.SetUsage(workspace.UUID, "pages", 3))
	return cache, workspace
}

func testTenant(t *core.T) (*Tenant, *Workspace) {
	cache, workspace := testCache(t)
	tenantService := &Tenant{cache: cache, entitlements: NewLocalEntitlementService(cache, nil), alertState: map[string]usageAlertTracker{}}
	return tenantService, workspace
}

func requireResultOK(t *core.T, result core.Result) {
	t.Helper()
	if !result.OK {
		t.Fatalf("expected OK result, got %v", result.Value)
	}
}

func requireResultFail(t *core.T, result core.Result) {
	t.Helper()
	if result.OK {
		t.Fatalf("expected failed result, got %v", result.Value)
	}
}

func TestTenant_Register_Good(t *core.T) {
	result := Register(core.New())
	requireResultOK(t, result)
	core.AssertNotNil(t, result.Value.(*Tenant).cache)
}

func TestTenant_Register_Bad(t *core.T) {
	result := Register(nil)
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoWorkspaceContext == result.Value, false)
}

func TestTenant_Register_Ugly(t *core.T) {
	c := core.New()
	result := Register(c)
	requireResultOK(t, result)
	core.AssertNotNil(t, result.Value.(*Tenant).entitlements)
}

func TestTenant_Tenant_GetWorkspace_Good(t *core.T) {
	tenantService, workspace := testTenant(t)
	result := tenantService.GetWorkspace(context.Background(), workspace.Slug)
	requireResultOK(t, result)
	core.AssertEqual(t, workspace.UUID, result.Value.(*Workspace).UUID)
}

func TestTenant_Tenant_GetWorkspace_Bad(t *core.T) {
	var tenantService *Tenant
	result := tenantService.GetWorkspace(context.Background(), "missing")
	requireResultFail(t, result)
	core.AssertEqual(t, ErrWorkspaceNotFound, result.Value)
}

func TestTenant_Tenant_GetWorkspace_Ugly(t *core.T) {
	tenantService := &Tenant{cache: NewTenantCache(nil)}
	result := tenantService.GetWorkspace(context.Background(), "missing")
	requireResultFail(t, result)
	core.AssertEqual(t, ErrWorkspaceNotFound, result.Value)
}

func TestTenant_Tenant_GetWorkspaceByUUID_Good(t *core.T) {
	tenantService, workspace := testTenant(t)
	result := tenantService.GetWorkspaceByUUID(context.Background(), workspace.UUID)
	requireResultOK(t, result)
	core.AssertEqual(t, workspace.Slug, result.Value.(*Workspace).Slug)
}

func TestTenant_Tenant_GetWorkspaceByUUID_Bad(t *core.T) {
	var tenantService *Tenant
	result := tenantService.GetWorkspaceByUUID(context.Background(), "missing")
	requireResultFail(t, result)
	core.AssertEqual(t, ErrWorkspaceNotFound, result.Value)
}

func TestTenant_Tenant_GetWorkspaceByUUID_Ugly(t *core.T) {
	tenantService := &Tenant{cache: NewTenantCache(nil)}
	result := tenantService.GetWorkspaceByUUID(context.Background(), "")
	requireResultFail(t, result)
	core.AssertEqual(t, ErrWorkspaceNotFound, result.Value)
}

func TestTenant_Tenant_GetWorkspaceByID_Good(t *core.T) {
	tenantService, workspace := testTenant(t)
	result := tenantService.GetWorkspaceByID(context.Background(), workspace.ID)
	requireResultOK(t, result)
	core.AssertEqual(t, workspace.UUID, result.Value.(*Workspace).UUID)
}

func TestTenant_Tenant_GetWorkspaceByID_Bad(t *core.T) {
	var tenantService *Tenant
	result := tenantService.GetWorkspaceByID(context.Background(), 404)
	requireResultFail(t, result)
	core.AssertEqual(t, ErrWorkspaceNotFound, result.Value)
}

func TestTenant_Tenant_GetWorkspaceByID_Ugly(t *core.T) {
	tenantService := &Tenant{cache: NewTenantCache(nil)}
	result := tenantService.GetWorkspaceByID(context.Background(), 0)
	requireResultFail(t, result)
	core.AssertEqual(t, ErrWorkspaceNotFound, result.Value)
}

func TestTenant_Tenant_GetUser_Good(t *core.T) {
	tenantService, _ := testTenant(t)
	result := tenantService.GetUser(WithUser(context.Background(), testUser()))
	requireResultOK(t, result)
	core.AssertEqual(t, "user-9", result.Value.(*User).UUID)
}

func TestTenant_Tenant_GetUser_Bad(t *core.T) {
	var tenantService *Tenant
	result := tenantService.GetUser(context.Background())
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoUserContext, result.Value)
}

func TestTenant_Tenant_GetUser_Ugly(t *core.T) {
	tenantService := &Tenant{cache: NewTenantCache(nil)}
	result := tenantService.GetUser(context.Background())
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoUserContext, result.Value)
}

func TestTenant_Tenant_GetWorkspaceBySubdomain_Good(t *core.T) {
	tenantService, workspace := testTenant(t)
	result := tenantService.GetWorkspaceBySubdomain(context.Background(), "acme.host.uk.com")
	requireResultOK(t, result)
	core.AssertEqual(t, workspace.UUID, result.Value.(*Workspace).UUID)
}

func TestTenant_Tenant_GetWorkspaceBySubdomain_Bad(t *core.T) {
	var tenantService *Tenant
	result := tenantService.GetWorkspaceBySubdomain(context.Background(), "missing.host.uk.com")
	requireResultFail(t, result)
	core.AssertEqual(t, ErrWorkspaceNotFound, result.Value)
}

func TestTenant_Tenant_GetWorkspaceBySubdomain_Ugly(t *core.T) {
	tenantService := &Tenant{cache: NewTenantCache(nil)}
	result := tenantService.GetWorkspaceBySubdomain(context.Background(), "")
	requireResultFail(t, result)
	core.AssertEqual(t, ErrWorkspaceNotFound, result.Value)
}

func TestTenant_Tenant_Can_Good(t *core.T) {
	tenantService, workspace := testTenant(t)
	result := tenantService.Can(context.Background(), workspace, "pages", 1)
	core.AssertTrue(t, result.IsAllowed())
	core.AssertEqual(t, "pages", result.FeatureCode)
}

func TestTenant_Tenant_Can_Bad(t *core.T) {
	tenantService, _ := testTenant(t)
	result := tenantService.Can(context.Background(), nil, "pages", 1)
	core.AssertTrue(t, result.IsDenied())
	core.AssertEqual(t, "no workspace provided", result.Reason)
}

func TestTenant_Tenant_Can_Ugly(t *core.T) {
	var tenantService *Tenant
	result := tenantService.Can(context.Background(), testWorkspace(), "pages", 1)
	core.AssertTrue(t, result.IsDenied())
	core.AssertEqual(t, "tenant not configured", result.Reason)
}

func TestTenant_Tenant_RecordUsage_Good(t *core.T) {
	tenantService, workspace := testTenant(t)
	result := tenantService.RecordUsage(context.Background(), workspace, "pages", 2, testInt64(9), nil)
	requireResultOK(t, result)
	used, ok := tenantService.cache.GetUsage(workspace.UUID, "pages")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, 5, used)
}

func TestTenant_Tenant_RecordUsage_Bad(t *core.T) {
	var tenantService *Tenant
	result := tenantService.RecordUsage(context.Background(), testWorkspace(), "pages", 1, nil, nil)
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoWorkspaceContext, result.Value)
}

func TestTenant_Tenant_RecordUsage_Ugly(t *core.T) {
	tenantService, _ := testTenant(t)
	result := tenantService.RecordUsage(context.Background(), nil, "pages", 1, nil, nil)
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoWorkspaceContext, result.Value)
}

func TestTenant_Tenant_GetUsageSummary_Good(t *core.T) {
	tenantService, workspace := testTenant(t)
	result := tenantService.GetUsageSummary(context.Background(), workspace)
	requireResultOK(t, result)
	core.AssertEqual(t, "pages", result.Value.([]UsageSummaryItem)[0].FeatureCode)
}

func TestTenant_Tenant_GetUsageSummary_Bad(t *core.T) {
	var tenantService *Tenant
	result := tenantService.GetUsageSummary(context.Background(), testWorkspace())
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoWorkspaceContext, result.Value)
}

func TestTenant_Tenant_GetUsageSummary_Ugly(t *core.T) {
	tenantService, _ := testTenant(t)
	result := tenantService.GetUsageSummary(context.Background(), nil)
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoWorkspaceContext, result.Value)
}

func TestTenant_Tenant_InvalidateWorkspace_Good(t *core.T) {
	tenantService, workspace := testTenant(t)
	tenantService.InvalidateWorkspace(workspace.UUID)
	got, ok := tenantService.cache.GetWorkspace(workspace.UUID)
	core.AssertFalse(t, ok)
	core.AssertNil(t, got)
}

func TestTenant_Tenant_InvalidateWorkspace_Bad(t *core.T) {
	var tenantService *Tenant
	tenantService.InvalidateWorkspace("missing")
	core.AssertNil(t, tenantService)
}

func TestTenant_Tenant_InvalidateWorkspace_Ugly(t *core.T) {
	tenantService, _ := testTenant(t)
	tenantService.alertState[alertStateKey("uuid-7", "pages")] = usageAlertTracker{lastUsedCount: 9}
	tenantService.InvalidateWorkspace("uuid-7")
	core.AssertEqual(t, 0, len(tenantService.alertState))
}

func TestTenant_Tenant_OnUsageAlert_Good(t *core.T) {
	tenantService, _ := testTenant(t)
	tenantService.OnUsageAlert(func(UsageAlert) {})
	core.AssertEqual(t, 1, len(tenantService.alertHandlers))
}

func TestTenant_Tenant_OnUsageAlert_Bad(t *core.T) {
	var tenantService *Tenant
	tenantService.OnUsageAlert(func(UsageAlert) {})
	core.AssertNil(t, tenantService)
}

func TestTenant_Tenant_OnUsageAlert_Ugly(t *core.T) {
	tenantService, _ := testTenant(t)
	tenantService.OnUsageAlert(nil)
	core.AssertEqual(t, 0, len(tenantService.alertHandlers))
}

func TestTenant_Tenant_Scope_Good(t *core.T) {
	tenantService, _ := testTenant(t)
	scope := tenantService.Scope()
	core.AssertNotNil(t, scope)
	core.AssertEqual(t, tenantService, scope.tenant)
}

func TestTenant_Tenant_Scope_Bad(t *core.T) {
	var tenantService *Tenant
	scope := tenantService.Scope()
	core.AssertNotNil(t, scope)
	core.AssertNil(t, scope.tenant)
}

func TestTenant_Tenant_Scope_Ugly(t *core.T) {
	tenantService := &Tenant{}
	scope := tenantService.Scope()
	core.AssertTrue(t, scope.strict)
	core.AssertEqual(t, tenantService, scope.tenant)
}

func TestTenant_Tenant_CheckUsageAlerts_Good(t *core.T) {
	tenantService, workspace := testTenant(t)
	count := 0
	tenantService.OnUsageAlert(func(UsageAlert) { count++ })
	tenantService.CheckUsageAlerts(workspace, "pages", Deny("pages", "limit", testInt(10), testInt(8)))
	core.AssertEqual(t, 1, count)
}

func TestTenant_Tenant_CheckUsageAlerts_Bad(t *core.T) {
	var tenantService *Tenant
	tenantService.CheckUsageAlerts(testWorkspace(), "pages", Allow("pages", testInt(10), testInt(8)))
	core.AssertNil(t, tenantService)
}

func TestTenant_Tenant_CheckUsageAlerts_Ugly(t *core.T) {
	tenantService, workspace := testTenant(t)
	count := 0
	tenantService.OnUsageAlert(func(UsageAlert) { count++ })
	tenantService.CheckUsageAlerts(workspace, "pages", AllowUnlimited("pages"))
	core.AssertEqual(t, 0, count)
}
