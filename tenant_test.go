// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
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

func TestTenant_RecordUsage_Good(t *testing.T) {
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
	if err := tenant.RecordUsage(context.Background(), workspace, "pages", 1, nil, nil); err != nil {
		t.Fatalf("record usage: %v", err)
	}

	result := tenant.Can(context.Background(), workspace, "pages", 0)
	if result.Used == nil || *result.Used != 4 {
		t.Fatalf("expected used=4 after cache-only record, got %+v", result.Used)
	}
	if result.IsDenied() {
		t.Fatalf("expected allowed after cache-only record, got %+v", result)
	}
}

func TestTenant_RecordUsage_Bad(t *testing.T) {
	tenant := &Tenant{}
	if err := tenant.RecordUsage(context.Background(), nil, "pages", 1, nil, nil); !errors.Is(err, ErrNoWorkspaceContext) {
		t.Fatalf("expected ErrNoWorkspaceContext, got %v", err)
	}
}

func TestTenant_RecordUsage_Ugly(t *testing.T) {
	serverUsage := 3
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/features/pages":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":1,"code":"pages","name":"Pages","type":"limit","reset_type":"none","is_active":true}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/acme/packages":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":1,"code":"starter","name":"Starter","is_active":true,"features":[{"feature_code":"pages","limit_value":10}]}]`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/acme/boosts":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/acme/usage/pages":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true,"count":` + strconv.Itoa(serverUsage) + `}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/acme/usage":
			serverUsage++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cache := NewTenantCache(nil)
	workspace := &Workspace{UUID: "acme", Slug: "acme"}
	_ = cache.SetWorkspace(workspace)
	_ = cache.SetFeature(&Feature{Code: "pages", Name: "Pages", Type: FeatureTypeLimit})
	_ = cache.SetPackages(workspace.UUID, []Package{{Code: "starter", IsActive: true, Features: []PackageFeature{{FeatureCode: "pages", LimitValue: intPtr(10)}}}})
	_ = cache.SetUsage(workspace.UUID, "pages", 3)

	tenant := &Tenant{
		client: NewTenantClient(server.URL, "token"),
		cache:  cache,
	}

	if err := tenant.RecordUsage(context.Background(), workspace, "pages", 1, nil, nil); err != nil {
		t.Fatalf("record usage: %v", err)
	}

	result := tenant.Can(context.Background(), workspace, "pages", 0)
	if result.Used == nil || *result.Used != 4 {
		t.Fatalf("expected refreshed usage=4, got %+v", result.Used)
	}
	if serverUsage != 4 {
		t.Fatalf("expected server usage=4, got %d", serverUsage)
	}
}

func TestTenant_GetWorkspaceByID_Good(t *testing.T) {
	cache := NewTenantCache(nil)
	workspace := &Workspace{ID: 42, UUID: "uuid-42", Slug: "acme"}
	if err := cache.SetWorkspace(workspace); err != nil {
		t.Fatalf("set workspace: %v", err)
	}
	tenant := &Tenant{cache: cache}
	got, err := tenant.GetWorkspaceByID(context.Background(), 42)
	if err != nil {
		t.Fatalf("get workspace: %v", err)
	}
	if got == nil || got.UUID != "uuid-42" {
		t.Fatalf("unexpected workspace: %+v", got)
	}
}

func TestTenant_GetWorkspaceByID_Bad(t *testing.T) {
	tenant := &Tenant{cache: NewTenantCache(nil)}
	if _, err := tenant.GetWorkspaceByID(context.Background(), 99); !errors.Is(err, ErrWorkspaceNotFound) {
		t.Fatalf("expected ErrWorkspaceNotFound, got %v", err)
	}
}

func TestTenant_GetWorkspaceByID_Ugly(t *testing.T) {
	var tenant *Tenant
	if _, err := tenant.GetWorkspaceByID(context.Background(), 99); !errors.Is(err, ErrWorkspaceNotFound) {
		t.Fatalf("expected ErrWorkspaceNotFound for nil tenant, got %v", err)
	}
}

func TestTenantClient_GetWorkspaceBySubdomain_Good(t *testing.T) {
	var subdomainHit atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/acme":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":1,"uuid":"uuid-1","slug":"acme","name":"Acme","is_active":true}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/subdomain/acme.host.uk.com":
			subdomainHit.Store(true)
			http.Error(w, "unexpected subdomain lookup", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewTenantClient(server.URL, "token")
	workspace, err := client.GetWorkspaceBySubdomain(context.Background(), "acme.host.uk.com")
	if err != nil {
		t.Fatalf("get workspace: %v", err)
	}
	if workspace == nil || workspace.Slug != "acme" {
		t.Fatalf("unexpected workspace: %+v", workspace)
	}
	if subdomainHit.Load() {
		t.Fatal("expected slug lookup to short-circuit before subdomain endpoint")
	}
}

func TestTenantClient_GetWorkspaceBySubdomain_Bad(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := NewTenantClient(server.URL, "token")
	if _, err := client.GetWorkspaceBySubdomain(context.Background(), "missing.host.uk.com"); !errors.Is(err, ErrWorkspaceNotFound) {
		t.Fatalf("expected ErrWorkspaceNotFound, got %v", err)
	}
}

func TestTenantClient_GetWorkspaceBySubdomain_Ugly(t *testing.T) {
	client := NewTenantClient("://bad-url", "token")
	if _, err := client.GetWorkspaceBySubdomain(context.Background(), "acme.host.uk.com"); err == nil {
		t.Fatal("expected error for invalid base URL")
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

func TestTenant_CheckUsageAlerts_Overflow_Good(t *testing.T) {
	tenant := &Tenant{}
	workspace := &Workspace{UUID: "uuid-7"}
	var fired []int
	tenant.OnUsageAlert(func(alert UsageAlert) {
		fired = append(fired, alert.Threshold)
	})

	limit := 10
	used := 11
	tenant.CheckUsageAlerts(workspace, "pages", Deny("pages", "limit reached", &limit, &used))

	if len(fired) != 3 {
		t.Fatalf("expected all thresholds to fire when already over limit, got %v", fired)
	}
	if fired[0] != AlertThresholdWarning || fired[1] != AlertThresholdCritical || fired[2] != AlertThresholdLimit {
		t.Fatalf("unexpected thresholds: %v", fired)
	}
}

func TestWorkspaceScope_Middleware_InvalidID_Bad(t *testing.T) {
	tenant := &Tenant{
		cache: NewTenantCache(nil),
	}
	_ = tenant.cache.SetWorkspace(&Workspace{ID: 7, UUID: "uuid-7", Slug: "acme"})
	scope := NewWorkspaceScope(tenant)

	handlerCalled := false
	handler := scope.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	req.Header.Set("X-Workspace-ID", "bogus")
	req.Header.Set("X-Workspace-Slug", "acme")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
	if handlerCalled {
		t.Fatal("expected request to stop before handler")
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
