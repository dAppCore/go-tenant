// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"context"
	"errors"
	"testing"
	"time"
)

func intPtr(value int) *int {
	return &value
}

func TestBoost_IsUsable_Good(t *testing.T) {
	boost := Boost{
		BoostType:        BoostTypeAddLimit,
		Status:           BoostStatusActive,
		LimitValue:       10,
		ConsumedQuantity: 2,
	}
	if !boost.IsUsable() {
		t.Fatal("expected active boost to be usable")
	}
}

func TestBoost_IsUsable_Bad(t *testing.T) {
	expiredAt := time.Now().Add(-time.Minute)
	boost := Boost{
		BoostType:        BoostTypeAddLimit,
		Status:           BoostStatusActive,
		LimitValue:       10,
		ConsumedQuantity: 2,
		ExpiresAt:        &expiredAt,
	}
	if boost.IsUsable() {
		t.Fatal("expected expired boost to be unusable")
	}
}

func TestBoost_IsUsable_Ugly(t *testing.T) {
	boost := Boost{
		BoostType:        BoostTypeAddLimit,
		Status:           BoostStatusActive,
		LimitValue:       10,
		ConsumedQuantity: 10,
	}
	if boost.IsUsable() {
		t.Fatal("expected exhausted boost to be unusable")
	}
}

func TestEntitlementResult_AsError_Good(t *testing.T) {
	if err := Allow("pages", intPtr(10), intPtr(3)).AsError(); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestEntitlementResult_AsError_Bad(t *testing.T) {
	err := Deny("pages", "limit reached", intPtr(10), intPtr(10)).AsError()
	if err == nil || !errors.Is(err, ErrEntitlementDenied) {
		t.Fatalf("expected ErrEntitlementDenied, got %v", err)
	}
}

func TestEntitlementResult_AsError_Ugly(t *testing.T) {
	if err := AllowUnlimited("pages").AsError(); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestFeature_PoolCode_Good(t *testing.T) {
	feature := Feature{Code: "pages"}
	if got := feature.PoolCode(); got != "pages" {
		t.Fatalf("expected root code, got %q", got)
	}
}

func TestFeature_PoolCode_Bad(t *testing.T) {
	parent := "pages"
	feature := Feature{Code: "pages.bio", ParentCode: &parent}
	if got := feature.PoolCode(); got != "pages" {
		t.Fatalf("expected parent code, got %q", got)
	}
}

func TestFeature_PoolCode_Ugly(t *testing.T) {
	feature := Feature{Code: "pages.bio"}
	if got := feature.PoolCode(); got != "pages.bio" {
		t.Fatalf("expected fallback code, got %q", got)
	}
}

func TestTenantCache_SetGet_Good(t *testing.T) {
	cache := NewTenantCache(nil)
	workspace := &Workspace{ID: 7, UUID: "uuid-7", Slug: "acme"}
	if err := cache.SetWorkspace(workspace); err != nil {
		t.Fatalf("set workspace: %v", err)
	}
	got, ok := cache.GetWorkspace("uuid-7")
	if !ok || got == nil {
		t.Fatal("expected workspace cache hit")
	}
	if got.UUID != "uuid-7" || got.Slug != "acme" {
		t.Fatalf("unexpected workspace: %+v", got)
	}
}

func TestTenantCache_SetGet_Bad(t *testing.T) {
	cache := NewTenantCache(nil)
	if got, ok := cache.GetWorkspace("missing"); ok || got != nil {
		t.Fatalf("expected miss, got %+v", got)
	}
}

func TestTenantCache_SetGet_Ugly(t *testing.T) {
	cache := NewTenantCache(nil)
	workspace := &Workspace{ID: 7, UUID: "uuid-7", Slug: "acme"}
	_ = cache.SetWorkspace(workspace)
	_ = cache.SetPackages(workspace.UUID, []Package{{Code: "starter", IsActive: true}})
	_ = cache.SetBoosts(workspace.UUID, []Boost{{FeatureCode: "pages", Status: BoostStatusActive, BoostType: BoostTypeAddLimit, LimitValue: 5}})
	_ = cache.SetUsage(workspace.UUID, "pages", 3)
	if err := cache.InvalidateWorkspace(workspace.UUID); err != nil {
		t.Fatalf("invalidate: %v", err)
	}
	if got, ok := cache.GetWorkspace(workspace.UUID); ok || got != nil {
		t.Fatal("expected workspace to be removed")
	}
	if _, ok := cache.GetPackages(workspace.UUID); ok {
		t.Fatal("expected packages to be removed")
	}
	if _, ok := cache.GetBoosts(workspace.UUID); ok {
		t.Fatal("expected boosts to be removed")
	}
	if _, ok := cache.GetUsage(workspace.UUID, "pages"); ok {
		t.Fatal("expected usage to be removed")
	}
}

func TestTenant_Can_Good(t *testing.T) {
	cache := NewTenantCache(nil)
	workspace := &Workspace{UUID: "uuid-7", Slug: "acme"}
	feature := &Feature{Code: "pages", Name: "Pages", Type: FeatureTypeLimit}
	packages := []Package{{Code: "starter", IsActive: true, Features: []PackageFeature{{FeatureCode: "pages", LimitValue: intPtr(10)}}}}
	if err := cache.SetWorkspace(workspace); err != nil {
		t.Fatalf("set workspace: %v", err)
	}
	if err := cache.SetFeature(feature); err != nil {
		t.Fatalf("set feature: %v", err)
	}
	if err := cache.SetPackages(workspace.UUID, packages); err != nil {
		t.Fatalf("set packages: %v", err)
	}
	if err := cache.SetUsage(workspace.UUID, "pages", 3); err != nil {
		t.Fatalf("set usage: %v", err)
	}
	tenant := &Tenant{cache: cache}
	result := tenant.Can(context.Background(), workspace, "pages", 0)
	if !result.IsAllowed() {
		t.Fatalf("expected snapshot check to allow, got %+v", result)
	}
	if result.Used == nil || *result.Used != 3 {
		t.Fatalf("expected used=3, got %+v", result.Used)
	}
	if result.Limit == nil || *result.Limit != 10 {
		t.Fatalf("expected limit=10, got %+v", result.Limit)
	}
}

func TestTenant_Can_Bad(t *testing.T) {
	tenant := &Tenant{}
	if result := tenant.Can(context.Background(), nil, "pages", 0); !result.IsDenied() {
		t.Fatalf("expected denial, got %+v", result)
	}
}

func TestTenant_Can_Ugly(t *testing.T) {
	tenant := &Tenant{cache: NewTenantCache(nil)}
	result := tenant.Can(context.Background(), &Workspace{UUID: "uuid-7"}, "pages", 0)
	if !result.IsDenied() {
		t.Fatalf("expected denial on cache miss, got %+v", result)
	}
}

func TestTenant_CheckUsageAlerts_Good(t *testing.T) {
	tenant := &Tenant{}
	workspace := &Workspace{UUID: "uuid-7"}
	var fired []int
	tenant.OnUsageAlert(func(alert UsageAlert) {
		fired = append(fired, alert.Threshold)
	})

	limit := 10
	for _, used := range []int{8, 9, 10} {
		tenant.CheckUsageAlerts(workspace, "pages", Allow("pages", &limit, &used))
	}

	if len(fired) != 3 {
		t.Fatalf("expected 3 alerts, got %v", fired)
	}
	if fired[0] != AlertThresholdWarning || fired[1] != AlertThresholdCritical || fired[2] != AlertThresholdLimit {
		t.Fatalf("unexpected thresholds: %v", fired)
	}
}

func TestTenant_CheckUsageAlerts_Bad(t *testing.T) {
	tenant := &Tenant{}
	workspace := &Workspace{UUID: "uuid-7"}
	var fired []int
	tenant.OnUsageAlert(func(alert UsageAlert) {
		fired = append(fired, alert.Threshold)
	})

	limit := 10
	used := 8
	result := Allow("pages", &limit, &used)
	tenant.CheckUsageAlerts(workspace, "pages", result)
	tenant.CheckUsageAlerts(workspace, "pages", result)

	if len(fired) != 1 {
		t.Fatalf("expected one alert for a stable threshold, got %v", fired)
	}
}

func TestTenant_CheckUsageAlerts_Ugly(t *testing.T) {
	tenant := &Tenant{}
	workspace := &Workspace{UUID: "uuid-7"}
	var fired []int
	tenant.OnUsageAlert(func(alert UsageAlert) {
		fired = append(fired, alert.Threshold)
	})

	limit := 10
	used := 8
	tenant.CheckUsageAlerts(workspace, "pages", Allow("pages", &limit, &used))
	used = 5
	tenant.CheckUsageAlerts(workspace, "pages", Allow("pages", &limit, &used))
	used = 8
	tenant.CheckUsageAlerts(workspace, "pages", Allow("pages", &limit, &used))

	if len(fired) != 2 {
		t.Fatalf("expected alerts to retrigger after usage reset, got %v", fired)
	}
}

func TestWorkspaceContext_Good(t *testing.T) {
	workspace := &Workspace{UUID: "uuid-1"}
	ctx := WithWorkspace(context.Background(), workspace)
	got, err := WorkspaceFromCtx(ctx)
	if err != nil {
		t.Fatalf("expected workspace, got %v", err)
	}
	if got.UUID != "uuid-1" {
		t.Fatalf("unexpected workspace: %+v", got)
	}
}

func TestWorkspaceContext_Bad(t *testing.T) {
	if _, err := WorkspaceFromCtx(context.Background()); !errors.Is(err, ErrNoWorkspaceContext) {
		t.Fatalf("expected ErrNoWorkspaceContext, got %v", err)
	}
}

func TestWorkspaceContext_Ugly(t *testing.T) {
	if ctx := WithWorkspace(nil, nil); ctx == nil {
		t.Fatal("expected non-nil context")
	}
}
