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

	"dappco.re/go/core"
	"dappco.re/go/core/store"
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
	st, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer st.Close()

	cache := NewTenantCache(st)
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

func TestTenantCache_SetWorkspaceRefresh_Good(t *testing.T) {
	cache := NewTenantCache(nil)
	workspace := &Workspace{ID: 7, UUID: "uuid-7", Slug: "acme"}
	if err := cache.SetWorkspace(workspace); err != nil {
		t.Fatalf("set workspace: %v", err)
	}

	workspace.Slug = "beta"
	workspace.ID = 8
	if err := cache.SetWorkspace(workspace); err != nil {
		t.Fatalf("refresh workspace: %v", err)
	}

	if got, ok := cache.GetWorkspaceBySlug("acme"); ok || got != nil {
		t.Fatal("expected old slug mapping to be removed")
	}
	if got, ok := cache.GetWorkspaceBySlug("beta"); !ok || got == nil || got.Slug != "beta" {
		t.Fatalf("expected refreshed slug mapping, got %+v", got)
	}
	if got, ok := cache.GetWorkspaceByID(7); ok || got != nil {
		t.Fatal("expected old id mapping to be removed")
	}
	if got, ok := cache.GetWorkspaceByID(8); !ok || got == nil || got.Slug != "beta" {
		t.Fatalf("expected refreshed id mapping, got %+v", got)
	}
}

func TestTenantCache_SetUser_Good(t *testing.T) {
	cache := NewTenantCache(nil)
	user := &User{UUID: "user-7", Name: "Ada", Email: "ada@example.uk"}
	if err := cache.SetUser(user); err != nil {
		t.Fatalf("set user: %v", err)
	}
	got, ok := cache.GetUser("user-7")
	if !ok || got == nil {
		t.Fatal("expected user cache hit")
	}
	if got.UUID != "user-7" || got.Email != "ada@example.uk" {
		t.Fatalf("unexpected user: %+v", got)
	}
}

func TestTenantCache_SetUser_Bad(t *testing.T) {
	cache := NewTenantCache(nil)
	if err := cache.SetUser(nil); !errors.Is(err, ErrNoUserContext) {
		t.Fatalf("expected ErrNoUserContext, got %v", err)
	}
}

func TestTenantCache_SetUser_Ugly(t *testing.T) {
	cache := NewTenantCache(nil)
	if err := cache.SetUser(&User{Name: "Ada"}); !errors.Is(err, ErrNoUserContext) {
		t.Fatalf("expected ErrNoUserContext for missing UUID, got %v", err)
	}
	if got, ok := cache.GetUser("missing"); ok || got != nil {
		t.Fatalf("expected user cache miss, got %+v", got)
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

func TestTenant_Can_MixedCase_Good(t *testing.T) {
	cache := NewTenantCache(nil)
	workspace := &Workspace{UUID: "uuid-7", Slug: "acme"}
	feature := &Feature{Code: "pages", Name: "Pages", Type: FeatureTypeLimit}
	packages := []Package{{Code: "starter", IsActive: true, Features: []PackageFeature{{FeatureCode: "PAGES", LimitValue: intPtr(10)}}}}
	if err := cache.SetWorkspace(workspace); err != nil {
		t.Fatalf("set workspace: %v", err)
	}
	if err := cache.SetFeature(feature); err != nil {
		t.Fatalf("set feature: %v", err)
	}
	if err := cache.SetPackages(workspace.UUID, packages); err != nil {
		t.Fatalf("set packages: %v", err)
	}
	if err := cache.SetUsage(workspace.UUID, "PAGES", 3); err != nil {
		t.Fatalf("set usage: %v", err)
	}

	tenant := &Tenant{cache: cache}
	result := tenant.Can(context.Background(), workspace, "PaGeS", 0)
	if !result.IsAllowed() {
		t.Fatalf("expected mixed-case check to allow, got %+v", result)
	}
	if result.Used == nil || *result.Used != 3 {
		t.Fatalf("expected used=3, got %+v", result.Used)
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

func TestTenant_Register_Good(t *testing.T) {
	c := core.New()
	result := Register(c)
	if !result.OK {
		t.Fatalf("expected successful registration, got %+v", result.Value)
	}
	tenant, ok := result.Value.(*Tenant)
	if !ok || tenant == nil {
		t.Fatalf("expected *Tenant value, got %#v", result.Value)
	}
	if tenant.ServiceRuntime == nil {
		t.Fatal("expected service runtime to be initialised")
	}
	if tenant.cache == nil {
		t.Fatal("expected cache to be initialised")
	}
	if tenant.entitlements == nil {
		t.Fatal("expected entitlement service to be initialised")
	}
}

func TestTenant_Can_BoostOnly_Bad(t *testing.T) {
	cache := NewTenantCache(nil)
	workspace := &Workspace{UUID: "uuid-7", Slug: "acme"}
	feature := &Feature{Code: "pages", Name: "Pages", Type: FeatureTypeLimit}
	if err := cache.SetWorkspace(workspace); err != nil {
		t.Fatalf("set workspace: %v", err)
	}
	if err := cache.SetFeature(feature); err != nil {
		t.Fatalf("set feature: %v", err)
	}
	if err := cache.SetBoosts(workspace.UUID, []Boost{{FeatureCode: "pages", Status: BoostStatusActive, BoostType: BoostTypeAddLimit, LimitValue: 5}}); err != nil {
		t.Fatalf("set boosts: %v", err)
	}

	tenant := &Tenant{cache: cache}
	result := tenant.Can(context.Background(), workspace, "pages", 1)
	if result.IsAllowed() {
		t.Fatalf("expected denial without a base package, got %+v", result)
	}
	if result.Reason == "" {
		t.Fatal("expected denial reason to be populated")
	}
}

func TestTenant_Can_UnlimitedBoost_Good(t *testing.T) {
	cache := NewTenantCache(nil)
	workspace := &Workspace{UUID: "uuid-7", Slug: "acme"}
	feature := &Feature{Code: "pages", Name: "Pages", Type: FeatureTypeLimit}
	if err := cache.SetWorkspace(workspace); err != nil {
		t.Fatalf("set workspace: %v", err)
	}
	if err := cache.SetFeature(feature); err != nil {
		t.Fatalf("set feature: %v", err)
	}
	if err := cache.SetBoosts(workspace.UUID, []Boost{{FeatureCode: "pages", Status: BoostStatusActive, BoostType: BoostTypeUnlimited}}); err != nil {
		t.Fatalf("set boosts: %v", err)
	}

	tenant := &Tenant{cache: cache}
	result := tenant.Can(context.Background(), workspace, "pages", 1)
	if !result.IsAllowed() || !result.Unlimited {
		t.Fatalf("expected unlimited allowance from boost, got %+v", result)
	}
}

func TestTenant_Can_UnlimitedFeature_Good(t *testing.T) {
	cache := NewTenantCache(nil)
	workspace := &Workspace{UUID: "uuid-7", Slug: "acme"}
	feature := &Feature{Code: "pages", Name: "Pages", Type: FeatureTypeUnlimited}
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

	tenant := &Tenant{cache: cache}
	result := tenant.Can(context.Background(), workspace, "pages", 1)
	if !result.IsAllowed() || !result.Unlimited {
		t.Fatalf("expected unlimited allowance from feature type, got %+v", result)
	}
}

func TestTenant_Can_UnlimitedFeature_Bad(t *testing.T) {
	cache := NewTenantCache(nil)
	workspace := &Workspace{UUID: "uuid-7", Slug: "acme"}
	feature := &Feature{Code: "pages", Name: "Pages", Type: FeatureTypeUnlimited}
	if err := cache.SetWorkspace(workspace); err != nil {
		t.Fatalf("set workspace: %v", err)
	}
	if err := cache.SetFeature(feature); err != nil {
		t.Fatalf("set feature: %v", err)
	}

	tenant := &Tenant{cache: cache}
	result := tenant.Can(context.Background(), workspace, "pages", 1)
	if result.IsAllowed() {
		t.Fatalf("expected denial without package membership, got %+v", result)
	}
}

func TestTenant_Can_UnlimitedFeature_Ugly(t *testing.T) {
	tenant := &Tenant{cache: NewTenantCache(nil)}
	result := tenant.Can(context.Background(), &Workspace{UUID: "uuid-7"}, "pages", 1)
	if result.IsAllowed() {
		t.Fatalf("expected denial on feature cache miss, got %+v", result)
	}
}

func TestTenant_Can_NonStackablePackagesUseHighestLimit_Good(t *testing.T) {
	cache := NewTenantCache(nil)
	workspace := &Workspace{UUID: "uuid-7", Slug: "acme"}
	feature := &Feature{Code: "pages", Name: "Pages", Type: FeatureTypeLimit}
	packages := []Package{
		{Code: "starter", IsActive: true, Features: []PackageFeature{{FeatureCode: "pages", LimitValue: intPtr(10)}}},
		{Code: "growth", IsActive: true, Features: []PackageFeature{{FeatureCode: "pages", LimitValue: intPtr(20)}}},
		{Code: "addon", IsActive: true, IsStackable: true, Features: []PackageFeature{{FeatureCode: "pages", LimitValue: intPtr(3)}}},
	}
	if err := cache.SetWorkspace(workspace); err != nil {
		t.Fatalf("set workspace: %v", err)
	}
	if err := cache.SetFeature(feature); err != nil {
		t.Fatalf("set feature: %v", err)
	}
	if err := cache.SetPackages(workspace.UUID, packages); err != nil {
		t.Fatalf("set packages: %v", err)
	}
	if err := cache.SetUsage(workspace.UUID, "pages", 24); err != nil {
		t.Fatalf("set usage: %v", err)
	}

	tenant := &Tenant{cache: cache}
	result := tenant.Can(context.Background(), workspace, "pages", 1)
	if result.IsAllowed() {
		t.Fatalf("expected denial once highest non-stackable + stackable limit is exhausted, got %+v", result)
	}
	if result.Limit == nil || *result.Limit != 23 {
		t.Fatalf("expected effective limit=23, got %+v", result.Limit)
	}
}

func TestTenant_Can_BooleanPackage_Good(t *testing.T) {
	cache := NewTenantCache(nil)
	workspace := &Workspace{UUID: "uuid-7", Slug: "acme"}
	feature := &Feature{Code: "api_access", Name: "API Access", Type: FeatureTypeBoolean}
	packages := []Package{{Code: "starter", IsActive: true, Features: []PackageFeature{{FeatureCode: "api_access", LimitValue: nil}}}}
	if err := cache.SetWorkspace(workspace); err != nil {
		t.Fatalf("set workspace: %v", err)
	}
	if err := cache.SetFeature(feature); err != nil {
		t.Fatalf("set feature: %v", err)
	}
	if err := cache.SetPackages(workspace.UUID, packages); err != nil {
		t.Fatalf("set packages: %v", err)
	}

	tenant := &Tenant{cache: cache}
	result := tenant.Can(context.Background(), workspace, "api_access", 1)
	if result.IsDenied() {
		t.Fatalf("expected boolean feature to be allowed by package membership, got %+v", result)
	}
	if result.Limit != nil || result.Used != nil || result.Remaining != nil {
		t.Fatalf("expected boolean feature to have nil usage context, got %+v", result)
	}
}

func TestTenant_Can_BooleanEnableBoost_Good(t *testing.T) {
	cache := NewTenantCache(nil)
	workspace := &Workspace{UUID: "uuid-7", Slug: "acme"}
	feature := &Feature{Code: "api_access", Name: "API Access", Type: FeatureTypeBoolean}
	if err := cache.SetWorkspace(workspace); err != nil {
		t.Fatalf("set workspace: %v", err)
	}
	if err := cache.SetFeature(feature); err != nil {
		t.Fatalf("set feature: %v", err)
	}
	if err := cache.SetBoosts(workspace.UUID, []Boost{{FeatureCode: "api_access", Status: BoostStatusActive, BoostType: BoostTypeEnable}}); err != nil {
		t.Fatalf("set boosts: %v", err)
	}

	tenant := &Tenant{cache: cache}
	result := tenant.Can(context.Background(), workspace, "api_access", 1)
	if result.IsDenied() {
		t.Fatalf("expected boolean feature to be enabled by boost, got %+v", result)
	}
	if result.Limit != nil || result.Used != nil || result.Remaining != nil {
		t.Fatalf("expected boolean feature to have nil usage context, got %+v", result)
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

func TestTenant_RecordUsage_RemoteInvalidatesUsageOnly_Good(t *testing.T) {
	var packagesHits int32
	var boostsHits int32
	var usageHits int32
	serverUsage := 3

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/features/pages":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":1,"code":"pages","name":"Pages","type":"limit","reset_type":"none","is_active":true}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/acme/packages":
			atomic.AddInt32(&packagesHits, 1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":1,"code":"starter","name":"Starter","is_active":true,"features":[{"feature_code":"pages","limit_value":10}]}]`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/acme/boosts":
			atomic.AddInt32(&boostsHits, 1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/acme/usage/pages":
			atomic.AddInt32(&usageHits, 1)
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

	workspace := &Workspace{UUID: "acme", Slug: "acme"}
	tenant := &Tenant{
		client: NewTenantClient(server.URL, "token"),
		cache:  NewTenantCache(nil),
	}

	initial := tenant.Can(context.Background(), workspace, "pages", 0)
	if initial.Used == nil || *initial.Used != 3 {
		t.Fatalf("expected initial usage=3, got %+v", initial.Used)
	}
	if err := tenant.RecordUsage(context.Background(), workspace, "pages", 1, nil, nil); err != nil {
		t.Fatalf("record usage: %v", err)
	}
	refreshed := tenant.Can(context.Background(), workspace, "pages", 0)
	if refreshed.Used == nil || *refreshed.Used != 4 {
		t.Fatalf("expected refreshed usage=4, got %+v", refreshed.Used)
	}
	if atomic.LoadInt32(&packagesHits) != 1 {
		t.Fatalf("expected packages to stay cached, got %d fetches", packagesHits)
	}
	if atomic.LoadInt32(&boostsHits) != 1 {
		t.Fatalf("expected boosts to stay cached, got %d fetches", boostsHits)
	}
	if atomic.LoadInt32(&usageHits) != 2 {
		t.Fatalf("expected usage to be reloaded exactly once, got %d fetches", usageHits)
	}
}

func TestTenant_GetUsageSummary_BooleanEnableBoost_Good(t *testing.T) {
	cache := NewTenantCache(nil)
	workspace := &Workspace{UUID: "uuid-7", Slug: "acme"}
	feature := &Feature{Code: "api_access", Name: "API Access", Type: FeatureTypeBoolean}
	if err := cache.SetWorkspace(workspace); err != nil {
		t.Fatalf("set workspace: %v", err)
	}
	if err := cache.SetFeature(feature); err != nil {
		t.Fatalf("set feature: %v", err)
	}
	if err := cache.SetBoosts(workspace.UUID, []Boost{{FeatureCode: "api_access", Status: BoostStatusActive, BoostType: BoostTypeEnable}}); err != nil {
		t.Fatalf("set boosts: %v", err)
	}

	tenant := &Tenant{cache: cache}
	items, err := tenant.GetUsageSummary(context.Background(), workspace)
	if err != nil {
		t.Fatalf("get usage summary: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one summary item, got %+v", items)
	}
	if items[0].FeatureCode != "api_access" {
		t.Fatalf("unexpected feature code: %+v", items[0])
	}
	if items[0].Limit != nil || items[0].Used != nil || items[0].Remaining != nil {
		t.Fatalf("expected boolean summary item to omit usage counters, got %+v", items[0])
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

func TestTenant_GetUser_Good(t *testing.T) {
	cache := NewTenantCache(nil)
	cached := &User{UUID: "user-7", Email: "cached@example.uk", Name: "Cached"}
	if err := cache.SetUser(cached); err != nil {
		t.Fatalf("set user: %v", err)
	}

	tenant := &Tenant{cache: cache}
	ctx := WithUser(context.Background(), &User{UUID: "user-7"})
	got, err := tenant.GetUser(ctx)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if got == nil || got.Email != "cached@example.uk" {
		t.Fatalf("unexpected user: %+v", got)
	}
}

func TestTenant_GetUser_Bad(t *testing.T) {
	tenant := &Tenant{cache: NewTenantCache(nil)}
	if _, err := tenant.GetUser(context.Background()); !errors.Is(err, ErrNoUserContext) {
		t.Fatalf("expected ErrNoUserContext, got %v", err)
	}
}

func TestTenant_GetUser_Ugly(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/user" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"uuid":"user-9","email":"ada@example.uk","name":"Ada"}`))
	}))
	defer server.Close()

	cache := NewTenantCache(nil)
	tenant := &Tenant{cache: cache, client: NewTenantClient(server.URL, "token")}
	got, err := tenant.GetUser(context.Background())
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if got == nil || got.UUID != "user-9" {
		t.Fatalf("unexpected user: %+v", got)
	}
	cached, ok := cache.GetUser("user-9")
	if !ok || cached == nil || cached.Email != "ada@example.uk" {
		t.Fatalf("expected fetched user to be cached, got %+v ok=%v", cached, ok)
	}
	if hits.Load() != 1 {
		t.Fatalf("expected one api call, got %d", hits.Load())
	}
}

func TestTenant_GetUsageSummary_NonStackablePackagesUseHighestLimit_Good(t *testing.T) {
	cache := NewTenantCache(nil)
	workspace := &Workspace{UUID: "uuid-7", Slug: "acme"}
	feature := &Feature{Code: "pages", Name: "Pages", Type: FeatureTypeLimit}
	packages := []Package{
		{Code: "starter", IsActive: true, Features: []PackageFeature{{FeatureCode: "pages", LimitValue: intPtr(10)}}},
		{Code: "growth", IsActive: true, Features: []PackageFeature{{FeatureCode: "pages", LimitValue: intPtr(20)}}},
		{Code: "addon", IsActive: true, IsStackable: true, Features: []PackageFeature{{FeatureCode: "pages", LimitValue: intPtr(3)}}},
	}
	if err := cache.SetWorkspace(workspace); err != nil {
		t.Fatalf("set workspace: %v", err)
	}
	if err := cache.SetFeature(feature); err != nil {
		t.Fatalf("set feature: %v", err)
	}
	if err := cache.SetPackages(workspace.UUID, packages); err != nil {
		t.Fatalf("set packages: %v", err)
	}
	if err := cache.SetUsage(workspace.UUID, "pages", 7); err != nil {
		t.Fatalf("set usage: %v", err)
	}

	tenant := &Tenant{cache: cache}
	items, err := tenant.GetUsageSummary(context.Background(), workspace)
	if err != nil {
		t.Fatalf("get usage summary: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one summary row, got %+v", items)
	}
	if items[0].Limit == nil || *items[0].Limit != 23 {
		t.Fatalf("expected effective limit=23, got %+v", items[0].Limit)
	}
	if items[0].Remaining == nil || *items[0].Remaining != 16 {
		t.Fatalf("expected remaining=16, got %+v", items[0].Remaining)
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

func TestTenantClient_GetWorkspaceBySlug_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/acme":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":1,"uuid":"uuid-1","slug":"acme","name":"Acme","is_active":true}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &TenantClient{baseURL: server.URL, token: "token"}
	workspace, err := client.GetWorkspaceBySlug(context.Background(), "acme")
	if err != nil {
		t.Fatalf("get workspace: %v", err)
	}
	if workspace == nil || workspace.Slug != "acme" {
		t.Fatalf("unexpected workspace: %+v", workspace)
	}
	if client.httpClient == nil {
		t.Fatal("expected http client to be initialised lazily")
	}
	if client.httpClient.Timeout <= 0 {
		t.Fatalf("expected positive timeout, got %v", client.httpClient.Timeout)
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

func TestTenant_CheckUsageAlerts_PanicHandler_Good(t *testing.T) {
	tenant := &Tenant{}
	workspace := &Workspace{UUID: "uuid-7"}
	var fired []int
	tenant.OnUsageAlert(func(alert UsageAlert) {
		panic("boom")
	})
	tenant.OnUsageAlert(func(alert UsageAlert) {
		fired = append(fired, alert.Threshold)
	})

	limit := 10
	used := 8
	tenant.CheckUsageAlerts(workspace, "pages", Allow("pages", &limit, &used))

	if len(fired) != 1 || fired[0] != AlertThresholdWarning {
		t.Fatalf("expected panic in first handler to not block second handler, got %v", fired)
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

func TestWorkspaceScope_ScopeFunc_Good(t *testing.T) {
	tenant := &Tenant{cache: NewTenantCache(nil)}
	workspace := &Workspace{UUID: "uuid-7", Slug: "acme"}
	if err := tenant.cache.SetWorkspace(workspace); err != nil {
		t.Fatalf("set workspace: %v", err)
	}

	scope := NewWorkspaceScope(tenant)
	err := scope.ScopeFunc(context.Background(), "acme", func(ctx context.Context) error {
		got, err := WorkspaceFromCtx(ctx)
		if err != nil {
			return err
		}
		if got.UUID != "uuid-7" {
			t.Fatalf("unexpected workspace: %+v", got)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scope func: %v", err)
	}
}

func TestWorkspaceScope_ScopeFunc_Bad(t *testing.T) {
	scope := NewWorkspaceScope(&Tenant{cache: NewTenantCache(nil)})
	if err := scope.ScopeFunc(context.Background(), "missing", func(ctx context.Context) error {
		return nil
	}); !errors.Is(err, ErrNoWorkspaceContext) {
		t.Fatalf("expected ErrNoWorkspaceContext, got %v", err)
	}
}

func TestWorkspaceScope_ScopeFunc_Ugly(t *testing.T) {
	tenant := &Tenant{cache: NewTenantCache(nil)}
	workspace := &Workspace{UUID: "uuid-7", Slug: "acme"}
	if err := tenant.cache.SetWorkspace(workspace); err != nil {
		t.Fatalf("set workspace: %v", err)
	}

	scope := NewWorkspaceScope(tenant)
	expected := core.E("tenant.scope", "callback failed", nil)
	if err := scope.ScopeFunc(context.Background(), "acme", func(ctx context.Context) error {
		return expected
	}); !errors.Is(err, expected) {
		t.Fatalf("expected callback error, got %v", err)
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
