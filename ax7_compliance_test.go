// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	"dappco.re/go"
)

func ax7Workspace() *Workspace {
	return &Workspace{ID: 7, UUID: "uuid-7", Slug: "acme", Name: "Acme", IsActive: true}
}

func ax7LimitFeature() *Feature {
	return &Feature{Code: "pages", Name: "Pages", Type: FeatureTypeLimit, IsActive: true}
}

func ax7Cache(t *core.T) (*TenantCache, *Workspace) {
	cache := NewTenantCache(nil)
	workspace := ax7Workspace()
	core.RequireNoError(t, cache.SetWorkspace(workspace))
	core.RequireNoError(t, cache.SetFeature(ax7LimitFeature()))
	core.RequireNoError(t, cache.SetPackages(workspace.UUID, []Package{{
		Code: "starter", IsActive: true, Features: []PackageFeature{{FeatureCode: "pages", LimitValue: intPtr(10)}},
	}}))
	core.RequireNoError(t, cache.SetUsage(workspace.UUID, "pages", 3))
	return cache, workspace
}

func ax7Tenant(t *core.T) (*Tenant, *Workspace) {
	cache, workspace := ax7Cache(t)
	tenantService := &Tenant{cache: cache, entitlements: NewLocalEntitlementService(cache, nil)}
	return tenantService, workspace
}

func ax7JSONServer(status int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

func TestAX7More_NewTenantCache_Good(t *core.T) {
	cache := NewTenantCache(nil)
	core.AssertNotNil(t, cache)
	core.AssertFalse(t, cache == nil)
	core.AssertEqual(t, 0, len(cache.workspacesByUUID))
}

func TestAX7More_NewTenantCache_Bad(t *core.T) {
	cache := NewTenantCache(nil)
	got, ok := cache.GetWorkspace("missing")
	core.AssertNil(t, got)
	core.AssertFalse(t, ok)
}

func TestAX7More_NewTenantCache_Ugly(t *core.T) {
	cache := NewTenantCache(nil)
	core.AssertNotNil(t, cache.packages)
	core.AssertNotNil(t, cache.boosts)
	core.AssertNotNil(t, cache.usage)
}

func TestAX7More_TenantCache_SetWorkspace_Good(t *core.T) {
	cache := NewTenantCache(nil)
	err := cache.SetWorkspace(ax7Workspace())
	got, ok := cache.GetWorkspace("uuid-7")
	core.AssertNoError(t, err)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "acme", got.Slug)
}

func TestAX7More_TenantCache_SetWorkspace_Bad(t *core.T) {
	cache := NewTenantCache(nil)
	err := cache.SetWorkspace(nil)
	got, ok := cache.GetWorkspace("uuid-7")
	core.AssertErrorIs(t, err, ErrNoWorkspaceContext)
	core.AssertNil(t, got)
	core.AssertFalse(t, ok)
}

func TestAX7More_TenantCache_SetWorkspace_Ugly(t *core.T) {
	cache := NewTenantCache(nil)
	workspace := ax7Workspace()
	core.RequireNoError(t, cache.SetWorkspace(workspace))
	workspace.Slug = "mutated"
	got, ok := cache.GetWorkspace("uuid-7")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "acme", got.Slug)
}

func TestAX7More_TenantCache_GetWorkspace_Good(t *core.T) {
	cache, workspace := ax7Cache(t)
	got, ok := cache.GetWorkspace(workspace.UUID)
	core.AssertTrue(t, ok)
	core.AssertNotNil(t, got)
	core.AssertEqual(t, workspace.UUID, got.UUID)
}

func TestAX7More_TenantCache_GetWorkspace_Bad(t *core.T) {
	cache := NewTenantCache(nil)
	got, ok := cache.GetWorkspace("missing")
	core.AssertFalse(t, ok)
	core.AssertNil(t, got)
	core.AssertEqual(t, 0, len(cache.workspacesByUUID))
}

func TestAX7More_TenantCache_GetWorkspace_Ugly(t *core.T) {
	cache, workspace := ax7Cache(t)
	got, ok := cache.GetWorkspace(workspace.UUID)
	got.Slug = "changed"
	again, againOK := cache.GetWorkspace(workspace.UUID)
	core.AssertTrue(t, ok && againOK)
	core.AssertEqual(t, "acme", again.Slug)
}

func TestAX7More_TenantCache_GetWorkspaceByID_Good(t *core.T) {
	cache, workspace := ax7Cache(t)
	got, ok := cache.GetWorkspaceByID(workspace.ID)
	core.AssertTrue(t, ok)
	core.AssertNotNil(t, got)
	core.AssertEqual(t, workspace.UUID, got.UUID)
}

func TestAX7More_TenantCache_GetWorkspaceByID_Bad(t *core.T) {
	cache := NewTenantCache(nil)
	got, ok := cache.GetWorkspaceByID(404)
	core.AssertFalse(t, ok)
	core.AssertNil(t, got)
	core.AssertEqual(t, 0, len(cache.workspaceIDs))
}

func TestAX7More_TenantCache_GetWorkspaceByID_Ugly(t *core.T) {
	cache, workspace := ax7Cache(t)
	workspace.ID = 8
	core.RequireNoError(t, cache.SetWorkspace(workspace))
	old, oldOK := cache.GetWorkspaceByID(7)
	got, ok := cache.GetWorkspaceByID(8)
	core.AssertFalse(t, oldOK)
	core.AssertNil(t, old)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, int64(8), got.ID)
}

func TestAX7More_TenantCache_GetWorkspaceBySlug_Good(t *core.T) {
	cache, workspace := ax7Cache(t)
	got, ok := cache.GetWorkspaceBySlug(workspace.Slug)
	core.AssertTrue(t, ok)
	core.AssertNotNil(t, got)
	core.AssertEqual(t, workspace.UUID, got.UUID)
}

func TestAX7More_TenantCache_GetWorkspaceBySlug_Bad(t *core.T) {
	cache := NewTenantCache(nil)
	got, ok := cache.GetWorkspaceBySlug("missing")
	core.AssertFalse(t, ok)
	core.AssertNil(t, got)
	core.AssertEqual(t, 0, len(cache.workspaceSlugs))
}

func TestAX7More_TenantCache_GetWorkspaceBySlug_Ugly(t *core.T) {
	cache, workspace := ax7Cache(t)
	workspace.Slug = "beta"
	core.RequireNoError(t, cache.SetWorkspace(workspace))
	old, oldOK := cache.GetWorkspaceBySlug("acme")
	got, ok := cache.GetWorkspaceBySlug("beta")
	core.AssertFalse(t, oldOK)
	core.AssertNil(t, old)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "beta", got.Slug)
}

func TestAX7More_TenantCache_SetPackages_Good(t *core.T) {
	cache := NewTenantCache(nil)
	err := cache.SetPackages("uuid-7", []Package{{Code: "starter", IsActive: true}})
	got, ok := cache.GetPackages("uuid-7")
	core.AssertNoError(t, err)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "starter", got[0].Code)
}

func TestAX7More_TenantCache_SetPackages_Bad(t *core.T) {
	var cache *TenantCache
	err := cache.SetPackages("uuid-7", nil)
	got, ok := cache.GetPackages("uuid-7")
	core.AssertErrorIs(t, err, ErrNoWorkspaceContext)
	core.AssertNil(t, got)
	core.AssertFalse(t, ok)
}

func TestAX7More_TenantCache_SetPackages_Ugly(t *core.T) {
	cache := NewTenantCache(nil)
	packages := []Package{{Code: "starter", IsActive: true}}
	core.RequireNoError(t, cache.SetPackages("uuid-7", packages))
	packages[0].Code = "mutated"
	got, ok := cache.GetPackages("uuid-7")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "starter", got[0].Code)
}

func TestAX7More_TenantCache_GetPackages_Good(t *core.T) {
	cache, workspace := ax7Cache(t)
	got, ok := cache.GetPackages(workspace.UUID)
	core.AssertTrue(t, ok)
	core.AssertLen(t, got, 1)
	core.AssertEqual(t, "starter", got[0].Code)
}

func TestAX7More_TenantCache_GetPackages_Bad(t *core.T) {
	cache := NewTenantCache(nil)
	got, ok := cache.GetPackages("missing")
	core.AssertFalse(t, ok)
	core.AssertNil(t, got)
	core.AssertEqual(t, 0, len(cache.packages))
}

func TestAX7More_TenantCache_GetPackages_Ugly(t *core.T) {
	cache, workspace := ax7Cache(t)
	got, ok := cache.GetPackages(workspace.UUID)
	got[0].Code = "mutated"
	again, againOK := cache.GetPackages(workspace.UUID)
	core.AssertTrue(t, ok && againOK)
	core.AssertEqual(t, "starter", again[0].Code)
}

func TestAX7More_TenantCache_SetBoosts_Good(t *core.T) {
	cache := NewTenantCache(nil)
	err := cache.SetBoosts("uuid-7", []Boost{{FeatureCode: "pages", Status: BoostStatusActive}})
	got, ok := cache.GetBoosts("uuid-7")
	core.AssertNoError(t, err)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "pages", got[0].FeatureCode)
}

func TestAX7More_TenantCache_SetBoosts_Bad(t *core.T) {
	var cache *TenantCache
	err := cache.SetBoosts("uuid-7", nil)
	got, ok := cache.GetBoosts("uuid-7")
	core.AssertErrorIs(t, err, ErrNoWorkspaceContext)
	core.AssertNil(t, got)
	core.AssertFalse(t, ok)
}

func TestAX7More_TenantCache_SetBoosts_Ugly(t *core.T) {
	cache := NewTenantCache(nil)
	boosts := []Boost{{FeatureCode: "pages", Status: BoostStatusActive}}
	core.RequireNoError(t, cache.SetBoosts("uuid-7", boosts))
	boosts[0].FeatureCode = "mutated"
	got, ok := cache.GetBoosts("uuid-7")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "pages", got[0].FeatureCode)
}

func TestAX7More_TenantCache_GetBoosts_Good(t *core.T) {
	cache, workspace := ax7Cache(t)
	core.RequireNoError(t, cache.SetBoosts(workspace.UUID, []Boost{{FeatureCode: "pages", Status: BoostStatusActive}}))
	got, ok := cache.GetBoosts(workspace.UUID)
	core.AssertTrue(t, ok)
	core.AssertLen(t, got, 1)
	core.AssertEqual(t, "pages", got[0].FeatureCode)
}

func TestAX7More_TenantCache_GetBoosts_Bad(t *core.T) {
	cache := NewTenantCache(nil)
	got, ok := cache.GetBoosts("missing")
	core.AssertFalse(t, ok)
	core.AssertNil(t, got)
	core.AssertEqual(t, 0, len(cache.boosts))
}

func TestAX7More_TenantCache_GetBoosts_Ugly(t *core.T) {
	cache, workspace := ax7Cache(t)
	core.RequireNoError(t, cache.SetBoosts(workspace.UUID, []Boost{{FeatureCode: "pages", Status: BoostStatusActive}}))
	got, ok := cache.GetBoosts(workspace.UUID)
	got[0].FeatureCode = "mutated"
	again, againOK := cache.GetBoosts(workspace.UUID)
	core.AssertTrue(t, ok && againOK)
	core.AssertEqual(t, "pages", again[0].FeatureCode)
}

func TestAX7More_TenantCache_SetUsage_Good(t *core.T) {
	cache := NewTenantCache(nil)
	err := cache.SetUsage("uuid-7", " PaGeS ", 4)
	got, ok := cache.GetUsage("uuid-7", "pages")
	core.AssertNoError(t, err)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, 4, got)
}

func TestAX7More_TenantCache_SetUsage_Bad(t *core.T) {
	var cache *TenantCache
	err := cache.SetUsage("uuid-7", "pages", 1)
	got, ok := cache.GetUsage("uuid-7", "pages")
	core.AssertErrorIs(t, err, ErrNoWorkspaceContext)
	core.AssertEqual(t, 0, got)
	core.AssertFalse(t, ok)
}

func TestAX7More_TenantCache_SetUsage_Ugly(t *core.T) {
	cache := NewTenantCache(nil)
	core.RequireNoError(t, cache.SetUsage("uuid-7", "", -2))
	got, ok := cache.GetUsage("uuid-7", "")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, -2, got)
}

func TestAX7More_TenantCache_GetUsage_Good(t *core.T) {
	cache, workspace := ax7Cache(t)
	got, ok := cache.GetUsage(workspace.UUID, "pages")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, 3, got)
	core.AssertNotEqual(t, 0, got)
}

func TestAX7More_TenantCache_GetUsage_Bad(t *core.T) {
	cache := NewTenantCache(nil)
	got, ok := cache.GetUsage("missing", "pages")
	core.AssertFalse(t, ok)
	core.AssertEqual(t, 0, got)
	core.AssertEqual(t, 0, len(cache.usage))
}

func TestAX7More_TenantCache_GetUsage_Ugly(t *core.T) {
	cache := NewTenantCache(nil)
	core.RequireNoError(t, cache.SetUsage("uuid-7", "PAGES", 9))
	got, ok := cache.GetUsage("uuid-7", " pages ")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, 9, got)
}

func TestAX7More_TenantCache_SetFeature_Good(t *core.T) {
	cache := NewTenantCache(nil)
	err := cache.SetFeature(ax7LimitFeature())
	got, ok := cache.GetFeature("pages")
	core.AssertNoError(t, err)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, FeatureTypeLimit, got.Type)
}

func TestAX7More_TenantCache_SetFeature_Bad(t *core.T) {
	cache := NewTenantCache(nil)
	err := cache.SetFeature(nil)
	got, ok := cache.GetFeature("pages")
	core.AssertErrorIs(t, err, ErrFeatureNotFound)
	core.AssertNil(t, got)
	core.AssertFalse(t, ok)
}

func TestAX7More_TenantCache_SetFeature_Ugly(t *core.T) {
	cache := NewTenantCache(nil)
	feature := ax7LimitFeature()
	core.RequireNoError(t, cache.SetFeature(feature))
	feature.Type = FeatureTypeBoolean
	got, ok := cache.GetFeature("pages")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, FeatureTypeLimit, got.Type)
}

func TestAX7More_TenantCache_GetFeature_Good(t *core.T) {
	cache, _ := ax7Cache(t)
	got, ok := cache.GetFeature("pages")
	core.AssertTrue(t, ok)
	core.AssertNotNil(t, got)
	core.AssertEqual(t, "pages", got.Code)
}

func TestAX7More_TenantCache_GetFeature_Bad(t *core.T) {
	cache := NewTenantCache(nil)
	got, ok := cache.GetFeature("missing")
	core.AssertFalse(t, ok)
	core.AssertNil(t, got)
	core.AssertEqual(t, 0, len(cache.features))
}

func TestAX7More_TenantCache_GetFeature_Ugly(t *core.T) {
	cache, _ := ax7Cache(t)
	got, ok := cache.GetFeature(" PAGES ")
	got.Code = "mutated"
	again, againOK := cache.GetFeature("pages")
	core.AssertTrue(t, ok && againOK)
	core.AssertEqual(t, "pages", again.Code)
}

func TestAX7More_TenantCache_SetUser_Good(t *core.T) {
	cache := NewTenantCache(nil)
	err := cache.SetUser(&User{UUID: "user-7", Email: "ada@example.uk"})
	got, ok := cache.GetUser("user-7")
	core.AssertNoError(t, err)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "ada@example.uk", got.Email)
}

func TestAX7More_TenantCache_SetUser_Bad(t *core.T) {
	cache := NewTenantCache(nil)
	err := cache.SetUser(nil)
	got, ok := cache.GetUser("user-7")
	core.AssertErrorIs(t, err, ErrNoUserContext)
	core.AssertNil(t, got)
	core.AssertFalse(t, ok)
}

func TestAX7More_TenantCache_SetUser_Ugly(t *core.T) {
	cache := NewTenantCache(nil)
	user := &User{UUID: "user-7", Email: "ada@example.uk"}
	core.RequireNoError(t, cache.SetUser(user))
	user.Email = "changed@example.uk"
	got, ok := cache.GetUser("user-7")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "ada@example.uk", got.Email)
}

func TestAX7More_TenantCache_GetUser_Good(t *core.T) {
	cache := NewTenantCache(nil)
	core.RequireNoError(t, cache.SetUser(&User{UUID: "user-7", Email: "ada@example.uk"}))
	got, ok := cache.GetUser("user-7")
	core.AssertTrue(t, ok)
	core.AssertNotNil(t, got)
	core.AssertEqual(t, "ada@example.uk", got.Email)
}

func TestAX7More_TenantCache_GetUser_Bad(t *core.T) {
	cache := NewTenantCache(nil)
	got, ok := cache.GetUser("missing")
	core.AssertFalse(t, ok)
	core.AssertNil(t, got)
	core.AssertEqual(t, 0, len(cache.users))
}

func TestAX7More_TenantCache_GetUser_Ugly(t *core.T) {
	cache := NewTenantCache(nil)
	core.RequireNoError(t, cache.SetUser(&User{UUID: "user-7", Email: "ada@example.uk"}))
	got, ok := cache.GetUser("user-7")
	got.Email = "changed@example.uk"
	again, againOK := cache.GetUser("user-7")
	core.AssertTrue(t, ok && againOK)
	core.AssertEqual(t, "ada@example.uk", again.Email)
}

func TestAX7More_TenantCache_InvalidateWorkspace_Good(t *core.T) {
	cache, workspace := ax7Cache(t)
	err := cache.InvalidateWorkspace(workspace.UUID)
	got, ok := cache.GetWorkspace(workspace.UUID)
	core.AssertNoError(t, err)
	core.AssertNil(t, got)
	core.AssertFalse(t, ok)
}

func TestAX7More_TenantCache_InvalidateWorkspace_Bad(t *core.T) {
	var cache *TenantCache
	err := cache.InvalidateWorkspace("uuid-7")
	got, ok := cache.GetWorkspace("uuid-7")
	core.AssertNoError(t, err)
	core.AssertNil(t, got)
	core.AssertFalse(t, ok)
}

func TestAX7More_TenantCache_InvalidateWorkspace_Ugly(t *core.T) {
	cache, workspace := ax7Cache(t)
	core.RequireNoError(t, cache.SetBoosts(workspace.UUID, []Boost{{FeatureCode: "pages", Status: BoostStatusActive}}))
	core.RequireNoError(t, cache.InvalidateWorkspace(workspace.UUID))
	_, packagesOK := cache.GetPackages(workspace.UUID)
	_, boostsOK := cache.GetBoosts(workspace.UUID)
	core.AssertFalse(t, packagesOK)
	core.AssertFalse(t, boostsOK)
}

func TestAX7More_Allow_Good(t *core.T) {
	result := Allow("pages", intPtr(10), intPtr(3))
	core.AssertTrue(t, result.Allowed)
	core.AssertEqual(t, "pages", result.FeatureCode)
	core.AssertEqual(t, 7, *result.Remaining)
}

func TestAX7More_Allow_Bad(t *core.T) {
	result := Allow("pages", nil, nil)
	core.AssertTrue(t, result.Allowed)
	core.AssertNil(t, result.Remaining)
	core.AssertFalse(t, result.Unlimited)
}

func TestAX7More_Allow_Ugly(t *core.T) {
	result := Allow("pages", intPtr(10), intPtr(12))
	core.AssertTrue(t, result.Allowed)
	core.AssertNotNil(t, result.Remaining)
	core.AssertEqual(t, 0, *result.Remaining)
}

func TestAX7More_Deny_Good(t *core.T) {
	result := Deny("pages", "limit reached", intPtr(10), intPtr(10))
	core.AssertFalse(t, result.Allowed)
	core.AssertEqual(t, "limit reached", result.Reason)
	core.AssertEqual(t, 0, *result.Remaining)
}

func TestAX7More_Deny_Bad(t *core.T) {
	result := Deny("pages", "missing", nil, nil)
	core.AssertFalse(t, result.Allowed)
	core.AssertNil(t, result.Remaining)
	core.AssertEqual(t, "missing", result.Reason)
}

func TestAX7More_Deny_Ugly(t *core.T) {
	result := Deny("", "", intPtr(2), intPtr(5))
	core.AssertFalse(t, result.Allowed)
	core.AssertEqual(t, "", result.FeatureCode)
	core.AssertEqual(t, 0, *result.Remaining)
}

func TestAX7More_AllowUnlimited_Good(t *core.T) {
	result := AllowUnlimited("pages")
	core.AssertTrue(t, result.Allowed)
	core.AssertTrue(t, result.Unlimited)
	core.AssertEqual(t, "pages", result.FeatureCode)
}

func TestAX7More_AllowUnlimited_Bad(t *core.T) {
	result := AllowUnlimited("")
	core.AssertTrue(t, result.Allowed)
	core.AssertTrue(t, result.Unlimited)
	core.AssertEqual(t, "", result.FeatureCode)
}

func TestAX7More_AllowUnlimited_Ugly(t *core.T) {
	result := AllowUnlimited("pages")
	pct := result.UsagePercent()
	core.AssertNil(t, pct)
	core.AssertFalse(t, result.IsAtLimit())
}

func TestAX7More_EntitlementResult_IsAllowed_Good(t *core.T) {
	result := Allow("pages", intPtr(10), intPtr(1))
	got := result.IsAllowed()
	core.AssertTrue(t, got)
	core.AssertFalse(t, result.IsDenied())
}

func TestAX7More_EntitlementResult_IsAllowed_Bad(t *core.T) {
	result := Deny("pages", "limit", intPtr(10), intPtr(10))
	got := result.IsAllowed()
	core.AssertFalse(t, got)
	core.AssertTrue(t, result.IsDenied())
}

func TestAX7More_EntitlementResult_IsAllowed_Ugly(t *core.T) {
	result := EntitlementResult{}
	got := result.IsAllowed()
	core.AssertFalse(t, got)
	core.AssertTrue(t, result.IsDenied())
}

func TestAX7More_EntitlementResult_IsDenied_Good(t *core.T) {
	result := Deny("pages", "limit", nil, nil)
	got := result.IsDenied()
	core.AssertTrue(t, got)
	core.AssertFalse(t, result.IsAllowed())
}

func TestAX7More_EntitlementResult_IsDenied_Bad(t *core.T) {
	result := Allow("pages", nil, nil)
	got := result.IsDenied()
	core.AssertFalse(t, got)
	core.AssertTrue(t, result.IsAllowed())
}

func TestAX7More_EntitlementResult_IsDenied_Ugly(t *core.T) {
	result := EntitlementResult{Allowed: false}
	got := result.IsDenied()
	core.AssertTrue(t, got)
	core.AssertEqual(t, "", result.Reason)
}

func TestAX7More_EntitlementResult_IsNearLimit_Good(t *core.T) {
	result := Allow("pages", intPtr(10), intPtr(8))
	got := result.IsNearLimit()
	core.AssertTrue(t, got)
	core.AssertFalse(t, result.IsAtLimit())
}

func TestAX7More_EntitlementResult_IsNearLimit_Bad(t *core.T) {
	result := Allow("pages", intPtr(10), intPtr(7))
	got := result.IsNearLimit()
	core.AssertFalse(t, got)
	core.AssertFalse(t, result.IsAtLimit())
}

func TestAX7More_EntitlementResult_IsNearLimit_Ugly(t *core.T) {
	result := AllowUnlimited("pages")
	got := result.IsNearLimit()
	core.AssertFalse(t, got)
	core.AssertNil(t, result.UsagePercent())
}

func TestAX7More_EntitlementResult_IsAtLimit_Good(t *core.T) {
	result := Allow("pages", intPtr(10), intPtr(10))
	got := result.IsAtLimit()
	core.AssertTrue(t, got)
	core.AssertTrue(t, result.IsNearLimit())
}

func TestAX7More_EntitlementResult_IsAtLimit_Bad(t *core.T) {
	result := Allow("pages", intPtr(10), intPtr(3))
	got := result.IsAtLimit()
	core.AssertFalse(t, got)
	core.AssertFalse(t, result.IsNearLimit())
}

func TestAX7More_EntitlementResult_IsAtLimit_Ugly(t *core.T) {
	result := AllowUnlimited("pages")
	got := result.IsAtLimit()
	core.AssertFalse(t, got)
	core.AssertNil(t, result.UsagePercent())
}

func TestAX7More_Feature_IsBoolean_Good(t *core.T) {
	feature := Feature{Type: FeatureTypeBoolean}
	got := feature.IsBoolean()
	core.AssertTrue(t, got)
	core.AssertFalse(t, feature.HasLimit())
}

func TestAX7More_Feature_IsBoolean_Bad(t *core.T) {
	feature := Feature{Type: FeatureTypeLimit}
	got := feature.IsBoolean()
	core.AssertFalse(t, got)
	core.AssertTrue(t, feature.HasLimit())
}

func TestAX7More_Feature_IsBoolean_Ugly(t *core.T) {
	feature := Feature{}
	got := feature.IsBoolean()
	core.AssertFalse(t, got)
	core.AssertFalse(t, feature.IsUnlimited())
}

func TestAX7More_Feature_HasLimit_Good(t *core.T) {
	feature := Feature{Type: FeatureTypeLimit}
	got := feature.HasLimit()
	core.AssertTrue(t, got)
	core.AssertFalse(t, feature.IsUnlimited())
}

func TestAX7More_Feature_HasLimit_Bad(t *core.T) {
	feature := Feature{Type: FeatureTypeBoolean}
	got := feature.HasLimit()
	core.AssertFalse(t, got)
	core.AssertTrue(t, feature.IsBoolean())
}

func TestAX7More_Feature_HasLimit_Ugly(t *core.T) {
	feature := Feature{}
	got := feature.HasLimit()
	core.AssertFalse(t, got)
	core.AssertEqual(t, FeatureType(""), feature.Type)
}

func TestAX7More_Feature_IsUnlimited_Good(t *core.T) {
	feature := Feature{Type: FeatureTypeUnlimited}
	got := feature.IsUnlimited()
	core.AssertTrue(t, got)
	core.AssertFalse(t, feature.HasLimit())
}

func TestAX7More_Feature_IsUnlimited_Bad(t *core.T) {
	feature := Feature{Type: FeatureTypeLimit}
	got := feature.IsUnlimited()
	core.AssertFalse(t, got)
	core.AssertTrue(t, feature.HasLimit())
}

func TestAX7More_Feature_IsUnlimited_Ugly(t *core.T) {
	feature := Feature{}
	got := feature.IsUnlimited()
	core.AssertFalse(t, got)
	core.AssertFalse(t, feature.IsBoolean())
}

func TestAX7More_Package_GetFeatureLimit_Good(t *core.T) {
	pkg := Package{Features: []PackageFeature{{FeatureCode: "pages", LimitValue: intPtr(10)}}}
	got := pkg.GetFeatureLimit("pages")
	core.AssertNotNil(t, got)
	core.AssertEqual(t, 10, *got)
}

func TestAX7More_Package_GetFeatureLimit_Bad(t *core.T) {
	pkg := Package{Features: []PackageFeature{{FeatureCode: "pages", LimitValue: intPtr(10)}}}
	got := pkg.GetFeatureLimit("uploads")
	core.AssertNil(t, got)
	core.AssertLen(t, pkg.Features, 1)
}

func TestAX7More_Package_GetFeatureLimit_Ugly(t *core.T) {
	pkg := Package{Features: []PackageFeature{{FeatureCode: " PaGeS ", LimitValue: nil}}}
	got := pkg.GetFeatureLimit("pages")
	core.AssertNil(t, got)
	core.AssertTrue(t, pkg.includesFeature(" pages "))
}

func TestAX7More_Boost_Remaining_Good(t *core.T) {
	boost := Boost{BoostType: BoostTypeAddLimit, LimitValue: 10, ConsumedQuantity: 3}
	got := boost.Remaining()
	core.AssertEqual(t, 7, got)
	core.AssertTrue(t, got > 0)
}

func TestAX7More_Boost_Remaining_Bad(t *core.T) {
	boost := Boost{BoostType: BoostTypeAddLimit, LimitValue: 10, ConsumedQuantity: 12}
	got := boost.Remaining()
	core.AssertEqual(t, 0, got)
	core.AssertFalse(t, got < 0)
}

func TestAX7More_Boost_Remaining_Ugly(t *core.T) {
	boost := Boost{BoostType: BoostTypeUnlimited}
	got := boost.Remaining()
	core.AssertEqual(t, -1, got)
	core.AssertTrue(t, boost.IsUsable() == false)
}

func TestAX7More_UserTier_MaxWorkspaces_Good(t *core.T) {
	got := TierApollo.MaxWorkspaces()
	core.AssertEqual(t, 5, got)
	core.AssertTrue(t, got > 1)
}

func TestAX7More_UserTier_MaxWorkspaces_Bad(t *core.T) {
	got := TierFree.MaxWorkspaces()
	core.AssertEqual(t, 1, got)
	core.AssertFalse(t, got < 0)
}

func TestAX7More_UserTier_MaxWorkspaces_Ugly(t *core.T) {
	got := TierHades.MaxWorkspaces()
	core.AssertEqual(t, -1, got)
	core.AssertTrue(t, got < 0)
}

func TestAX7More_UserTier_HasFeature_Good(t *core.T) {
	got := TierApollo.HasFeature(" api_access ")
	core.AssertTrue(t, got)
	core.AssertFalse(t, TierApollo.HasFeature("uploads"))
}

func TestAX7More_UserTier_HasFeature_Bad(t *core.T) {
	got := TierFree.HasFeature("api_access")
	core.AssertFalse(t, got)
	core.AssertEqual(t, 1, TierFree.MaxWorkspaces())
}

func TestAX7More_UserTier_HasFeature_Ugly(t *core.T) {
	got := TierHades.HasFeature("")
	core.AssertTrue(t, got)
	core.AssertEqual(t, -1, TierHades.MaxWorkspaces())
}

func TestAX7More_WithWorkspace_Good(t *core.T) {
	ctx := WithWorkspace(context.Background(), ax7Workspace())
	got, err := WorkspaceFromCtx(ctx)
	core.AssertNoError(t, err)
	core.AssertNotNil(t, got)
	core.AssertEqual(t, "uuid-7", got.UUID)
}

func TestAX7More_WithWorkspace_Bad(t *core.T) {
	ctx := WithWorkspace(context.Background(), nil)
	got, err := WorkspaceFromCtx(ctx)
	core.AssertErrorIs(t, err, ErrNoWorkspaceContext)
	core.AssertNil(t, got)
	core.AssertNotNil(t, ctx)
}

func TestAX7More_WithWorkspace_Ugly(t *core.T) {
	ctx := WithWorkspace(nil, ax7Workspace())
	got, err := WorkspaceFromCtx(ctx)
	core.AssertNoError(t, err)
	core.AssertNotNil(t, ctx)
	core.AssertEqual(t, "acme", got.Slug)
}

func TestAX7More_WorkspaceFromCtx_Good(t *core.T) {
	ctx := WithWorkspace(context.Background(), ax7Workspace())
	got, err := WorkspaceFromCtx(ctx)
	core.AssertNoError(t, err)
	core.AssertNotNil(t, got)
	core.AssertEqual(t, int64(7), got.ID)
}

func TestAX7More_WorkspaceFromCtx_Bad(t *core.T) {
	got, err := WorkspaceFromCtx(context.Background())
	core.AssertErrorIs(t, err, ErrNoWorkspaceContext)
	core.AssertNil(t, got)
	core.AssertNotNil(t, err)
}

func TestAX7More_WorkspaceFromCtx_Ugly(t *core.T) {
	got, err := WorkspaceFromCtx(nil)
	core.AssertErrorIs(t, err, ErrNoWorkspaceContext)
	core.AssertNil(t, got)
	core.AssertNotNil(t, err)
}

func TestAX7More_WithUser_Good(t *core.T) {
	ctx := WithUser(context.Background(), &User{UUID: "user-7", Email: "ada@example.uk"})
	got, err := UserFromCtx(ctx)
	core.AssertNoError(t, err)
	core.AssertNotNil(t, got)
	core.AssertEqual(t, "user-7", got.UUID)
}

func TestAX7More_WithUser_Bad(t *core.T) {
	ctx := WithUser(context.Background(), nil)
	got, err := UserFromCtx(ctx)
	core.AssertErrorIs(t, err, ErrNoUserContext)
	core.AssertNil(t, got)
	core.AssertNotNil(t, ctx)
}

func TestAX7More_WithUser_Ugly(t *core.T) {
	ctx := WithUser(nil, &User{UUID: "user-7"})
	got, err := UserFromCtx(ctx)
	core.AssertNoError(t, err)
	core.AssertNotNil(t, ctx)
	core.AssertEqual(t, "user-7", got.UUID)
}

func TestAX7More_UserFromCtx_Good(t *core.T) {
	ctx := WithUser(context.Background(), &User{UUID: "user-7"})
	got, err := UserFromCtx(ctx)
	core.AssertNoError(t, err)
	core.AssertNotNil(t, got)
	core.AssertEqual(t, "user-7", got.UUID)
}

func TestAX7More_UserFromCtx_Bad(t *core.T) {
	got, err := UserFromCtx(context.Background())
	core.AssertErrorIs(t, err, ErrNoUserContext)
	core.AssertNil(t, got)
	core.AssertNotNil(t, err)
}

func TestAX7More_UserFromCtx_Ugly(t *core.T) {
	got, err := UserFromCtx(nil)
	core.AssertErrorIs(t, err, ErrNoUserContext)
	core.AssertNil(t, got)
	core.AssertNotNil(t, err)
}

func TestAX7More_WorkspaceContext_WithWorkspace_Good(t *core.T) {
	holder := WorkspaceContext{Context: context.Background()}.WithWorkspace(ax7Workspace())
	got, err := holder.Workspace()
	core.AssertNoError(t, err)
	core.AssertNotNil(t, got)
	core.AssertEqual(t, "uuid-7", got.UUID)
}

func TestAX7More_WorkspaceContext_WithWorkspace_Bad(t *core.T) {
	holder := WorkspaceContext{Context: context.Background()}.WithWorkspace(nil)
	got, err := holder.Workspace()
	core.AssertErrorIs(t, err, ErrNoWorkspaceContext)
	core.AssertNil(t, got)
	core.AssertNotNil(t, holder.Context)
}

func TestAX7More_WorkspaceContext_WithWorkspace_Ugly(t *core.T) {
	holder := WorkspaceContext{}.WithWorkspace(ax7Workspace())
	got, err := holder.Workspace()
	core.AssertNoError(t, err)
	core.AssertNotNil(t, holder.Context)
	core.AssertEqual(t, "acme", got.Slug)
}

func TestAX7More_WorkspaceContext_WithUser_Good(t *core.T) {
	holder := WorkspaceContext{Context: context.Background()}.WithUser(&User{UUID: "user-7"})
	got, err := holder.User()
	core.AssertNoError(t, err)
	core.AssertNotNil(t, got)
	core.AssertEqual(t, "user-7", got.UUID)
}

func TestAX7More_WorkspaceContext_WithUser_Bad(t *core.T) {
	holder := WorkspaceContext{Context: context.Background()}.WithUser(nil)
	got, err := holder.User()
	core.AssertErrorIs(t, err, ErrNoUserContext)
	core.AssertNil(t, got)
	core.AssertNotNil(t, holder.Context)
}

func TestAX7More_WorkspaceContext_WithUser_Ugly(t *core.T) {
	holder := WorkspaceContext{}.WithUser(&User{UUID: "user-7"})
	got, err := holder.User()
	core.AssertNoError(t, err)
	core.AssertNotNil(t, holder.Context)
	core.AssertEqual(t, "user-7", got.UUID)
}

func TestAX7More_WorkspaceContext_Workspace_Good(t *core.T) {
	holder := WorkspaceContext{Context: WithWorkspace(context.Background(), ax7Workspace())}
	got, err := holder.Workspace()
	core.AssertNoError(t, err)
	core.AssertNotNil(t, got)
	core.AssertEqual(t, "uuid-7", got.UUID)
}

func TestAX7More_WorkspaceContext_Workspace_Bad(t *core.T) {
	holder := WorkspaceContext{Context: context.Background()}
	got, err := holder.Workspace()
	core.AssertErrorIs(t, err, ErrNoWorkspaceContext)
	core.AssertNil(t, got)
	core.AssertNotNil(t, err)
}

func TestAX7More_WorkspaceContext_Workspace_Ugly(t *core.T) {
	holder := WorkspaceContext{}
	got, err := holder.Workspace()
	core.AssertErrorIs(t, err, ErrNoWorkspaceContext)
	core.AssertNil(t, got)
	core.AssertNotNil(t, err)
}

func TestAX7More_WorkspaceContext_User_Good(t *core.T) {
	holder := WorkspaceContext{Context: WithUser(context.Background(), &User{UUID: "user-7"})}
	got, err := holder.User()
	core.AssertNoError(t, err)
	core.AssertNotNil(t, got)
	core.AssertEqual(t, "user-7", got.UUID)
}

func TestAX7More_WorkspaceContext_User_Bad(t *core.T) {
	holder := WorkspaceContext{Context: context.Background()}
	got, err := holder.User()
	core.AssertErrorIs(t, err, ErrNoUserContext)
	core.AssertNil(t, got)
	core.AssertNotNil(t, err)
}

func TestAX7More_WorkspaceContext_User_Ugly(t *core.T) {
	holder := WorkspaceContext{}
	got, err := holder.User()
	core.AssertErrorIs(t, err, ErrNoUserContext)
	core.AssertNil(t, got)
	core.AssertNotNil(t, err)
}

func TestAX7More_NewLocalEntitlementService_Good(t *core.T) {
	cache, _ := ax7Cache(t)
	svc := NewLocalEntitlementService(cache, nil)
	result := svc.Can(context.Background(), ax7Workspace(), "pages", 1)
	core.AssertNotNil(t, svc)
	core.AssertTrue(t, result.IsAllowed())
}

func TestAX7More_NewLocalEntitlementService_Bad(t *core.T) {
	svc := NewLocalEntitlementService(nil, nil)
	result := svc.Can(context.Background(), ax7Workspace(), "pages", 1)
	core.AssertNotNil(t, svc)
	core.AssertTrue(t, result.IsDenied())
}

func TestAX7More_NewLocalEntitlementService_Ugly(t *core.T) {
	svc := NewLocalEntitlementService(nil, nil)
	err := svc.RecordUsage(context.Background(), nil, "pages", 1, nil, nil)
	core.AssertNotNil(t, svc)
	core.AssertErrorIs(t, err, ErrNoWorkspaceContext)
}

func TestAX7More_EntitlementService_Can_Good(t *core.T) {
	cache, workspace := ax7Cache(t)
	svc := NewLocalEntitlementService(cache, nil)
	result := svc.Can(context.Background(), workspace, "pages", 1)
	core.AssertTrue(t, result.IsAllowed())
	core.AssertEqual(t, 10, *result.Limit)
	core.AssertEqual(t, 3, *result.Used)
}

func TestAX7More_EntitlementService_Can_Bad(t *core.T) {
	cache, _ := ax7Cache(t)
	svc := NewLocalEntitlementService(cache, nil)
	result := svc.Can(context.Background(), nil, "pages", 1)
	core.AssertTrue(t, result.IsDenied())
	core.AssertEqual(t, "no workspace provided", result.Reason)
	core.AssertEqual(t, "pages", result.FeatureCode)
}

func TestAX7More_EntitlementService_Can_Ugly(t *core.T) {
	cache, workspace := ax7Cache(t)
	svc := NewLocalEntitlementService(cache, nil)
	result := svc.Can(context.Background(), workspace, " PAGES ", -1)
	core.AssertTrue(t, result.IsAllowed())
	core.AssertEqual(t, "pages", result.FeatureCode)
	core.AssertEqual(t, 7, *result.Remaining)
}

func TestAX7More_EntitlementService_RecordUsage_Good(t *core.T) {
	cache, workspace := ax7Cache(t)
	svc := NewLocalEntitlementService(cache, nil)
	err := svc.RecordUsage(context.Background(), workspace, "pages", 2, nil, nil)
	used, ok := cache.GetUsage(workspace.UUID, "pages")
	core.AssertNoError(t, err)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, 5, used)
}

func TestAX7More_EntitlementService_RecordUsage_Bad(t *core.T) {
	cache, _ := ax7Cache(t)
	svc := NewLocalEntitlementService(cache, nil)
	err := svc.RecordUsage(context.Background(), nil, "pages", 1, nil, nil)
	core.AssertErrorIs(t, err, ErrNoWorkspaceContext)
	core.AssertNotNil(t, svc)
	core.AssertNotNil(t, cache)
}

func TestAX7More_EntitlementService_RecordUsage_Ugly(t *core.T) {
	cache, workspace := ax7Cache(t)
	svc := NewLocalEntitlementService(cache, nil)
	err := svc.RecordUsage(context.Background(), workspace, "pages", 0, nil, map[string]any{"source": "test"})
	used, ok := cache.GetUsage(workspace.UUID, "pages")
	core.AssertNoError(t, err)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, 4, used)
}

func TestAX7More_EntitlementService_GetUsageSummary_Good(t *core.T) {
	cache, workspace := ax7Cache(t)
	svc := NewLocalEntitlementService(cache, nil)
	items, err := svc.GetUsageSummary(context.Background(), workspace)
	core.AssertNoError(t, err)
	core.AssertLen(t, items, 1)
	core.AssertEqual(t, "pages", items[0].FeatureCode)
}

func TestAX7More_EntitlementService_GetUsageSummary_Bad(t *core.T) {
	cache, _ := ax7Cache(t)
	svc := NewLocalEntitlementService(cache, nil)
	items, err := svc.GetUsageSummary(context.Background(), nil)
	core.AssertErrorIs(t, err, ErrNoWorkspaceContext)
	core.AssertNil(t, items)
	core.AssertNotNil(t, svc)
}

func TestAX7More_EntitlementService_GetUsageSummary_Ugly(t *core.T) {
	cache, workspace := ax7Cache(t)
	core.RequireNoError(t, cache.SetBoosts(workspace.UUID, []Boost{{FeatureCode: "uploads", Status: BoostStatusActive, BoostType: BoostTypeEnable}}))
	svc := NewLocalEntitlementService(cache, nil)
	items, err := svc.GetUsageSummary(context.Background(), workspace)
	core.AssertNoError(t, err)
	core.AssertLen(t, items, 2)
}

func TestAX7More_EntitlementService_InvalidateWorkspace_Good(t *core.T) {
	cache, workspace := ax7Cache(t)
	svc := NewLocalEntitlementService(cache, nil)
	svc.InvalidateWorkspace(workspace.UUID)
	got, ok := cache.GetWorkspace(workspace.UUID)
	core.AssertNil(t, got)
	core.AssertFalse(t, ok)
}

func TestAX7More_EntitlementService_InvalidateWorkspace_Bad(t *core.T) {
	svc := NewLocalEntitlementService(nil, nil)
	core.AssertNotPanics(t, func() { svc.InvalidateWorkspace("uuid-7") })
	core.AssertNotNil(t, svc)
	core.AssertTrue(t, true)
}

func TestAX7More_EntitlementService_InvalidateWorkspace_Ugly(t *core.T) {
	cache, _ := ax7Cache(t)
	svc := NewLocalEntitlementService(cache, nil)
	svc.InvalidateWorkspace("")
	core.AssertNotNil(t, svc)
	core.AssertNotNil(t, cache)
}

func TestAX7More_Tenant_GetWorkspace_Good(t *core.T) {
	tenantService, workspace := ax7Tenant(t)
	got, err := tenantService.GetWorkspace(context.Background(), workspace.Slug)
	core.AssertNoError(t, err)
	core.AssertNotNil(t, got)
	core.AssertEqual(t, workspace.UUID, got.UUID)
}

func TestAX7More_Tenant_GetWorkspace_Bad(t *core.T) {
	tenantService := &Tenant{}
	got, err := tenantService.GetWorkspace(context.Background(), "missing")
	core.AssertErrorIs(t, err, ErrWorkspaceNotFound)
	core.AssertNil(t, got)
	core.AssertNotNil(t, tenantService)
}

func TestAX7More_Tenant_GetWorkspace_Ugly(t *core.T) {
	var tenantService *Tenant
	got, err := tenantService.GetWorkspace(context.Background(), "acme")
	core.AssertErrorIs(t, err, ErrWorkspaceNotFound)
	core.AssertNil(t, got)
	core.AssertNil(t, tenantService)
}

func TestAX7More_Tenant_GetWorkspaceByUUID_Good(t *core.T) {
	tenantService, workspace := ax7Tenant(t)
	got, err := tenantService.GetWorkspaceByUUID(context.Background(), workspace.UUID)
	core.AssertNoError(t, err)
	core.AssertNotNil(t, got)
	core.AssertEqual(t, workspace.Slug, got.Slug)
}

func TestAX7More_Tenant_GetWorkspaceByUUID_Bad(t *core.T) {
	tenantService := &Tenant{}
	got, err := tenantService.GetWorkspaceByUUID(context.Background(), "missing")
	core.AssertErrorIs(t, err, ErrWorkspaceNotFound)
	core.AssertNil(t, got)
	core.AssertNotNil(t, tenantService)
}

func TestAX7More_Tenant_GetWorkspaceByUUID_Ugly(t *core.T) {
	var tenantService *Tenant
	got, err := tenantService.GetWorkspaceByUUID(context.Background(), "uuid-7")
	core.AssertErrorIs(t, err, ErrWorkspaceNotFound)
	core.AssertNil(t, got)
	core.AssertNil(t, tenantService)
}

func TestAX7More_Tenant_GetWorkspaceBySubdomain_Good(t *core.T) {
	tenantService, workspace := ax7Tenant(t)
	got, err := tenantService.GetWorkspaceBySubdomain(context.Background(), "acme.example.uk")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, got)
	core.AssertEqual(t, workspace.UUID, got.UUID)
}

func TestAX7More_Tenant_GetWorkspaceBySubdomain_Bad(t *core.T) {
	tenantService := &Tenant{}
	got, err := tenantService.GetWorkspaceBySubdomain(context.Background(), "missing.example.uk")
	core.AssertErrorIs(t, err, ErrWorkspaceNotFound)
	core.AssertNil(t, got)
	core.AssertNotNil(t, tenantService)
}

func TestAX7More_Tenant_GetWorkspaceBySubdomain_Ugly(t *core.T) {
	var tenantService *Tenant
	got, err := tenantService.GetWorkspaceBySubdomain(context.Background(), "")
	core.AssertErrorIs(t, err, ErrWorkspaceNotFound)
	core.AssertNil(t, got)
	core.AssertNil(t, tenantService)
}

func TestAX7More_Tenant_GetUsageSummary_Good(t *core.T) {
	tenantService, workspace := ax7Tenant(t)
	items, err := tenantService.GetUsageSummary(context.Background(), workspace)
	core.AssertNoError(t, err)
	core.AssertLen(t, items, 1)
	core.AssertEqual(t, "pages", items[0].FeatureCode)
}

func TestAX7More_Tenant_GetUsageSummary_Bad(t *core.T) {
	tenantService := &Tenant{}
	items, err := tenantService.GetUsageSummary(context.Background(), nil)
	core.AssertErrorIs(t, err, ErrNoWorkspaceContext)
	core.AssertNil(t, items)
	core.AssertNotNil(t, tenantService)
}

func TestAX7More_Tenant_GetUsageSummary_Ugly(t *core.T) {
	var tenantService *Tenant
	items, err := tenantService.GetUsageSummary(context.Background(), ax7Workspace())
	core.AssertErrorIs(t, err, ErrNoWorkspaceContext)
	core.AssertNil(t, items)
	core.AssertNil(t, tenantService)
}

func TestAX7More_Tenant_InvalidateWorkspace_Good(t *core.T) {
	tenantService, workspace := ax7Tenant(t)
	tenantService.alertState = map[string]usageAlertTracker{alertStateKey(workspace.UUID, "pages"): {lastUsedCount: 8}}
	tenantService.InvalidateWorkspace(workspace.UUID)
	_, ok := tenantService.cache.GetWorkspace(workspace.UUID)
	core.AssertFalse(t, ok)
	core.AssertEqual(t, 0, len(tenantService.alertState))
}

func TestAX7More_Tenant_InvalidateWorkspace_Bad(t *core.T) {
	var tenantService *Tenant
	core.AssertNotPanics(t, func() { tenantService.InvalidateWorkspace("uuid-7") })
	core.AssertNil(t, tenantService)
	core.AssertTrue(t, true)
}

func TestAX7More_Tenant_InvalidateWorkspace_Ugly(t *core.T) {
	tenantService, _ := ax7Tenant(t)
	tenantService.alertState = map[string]usageAlertTracker{"other\x00pages": {lastUsedCount: 1}}
	tenantService.InvalidateWorkspace("uuid-7")
	core.AssertEqual(t, 1, len(tenantService.alertState))
	core.AssertNotNil(t, tenantService.cache)
}

func TestAX7More_Tenant_OnUsageAlert_Good(t *core.T) {
	tenantService := &Tenant{}
	tenantService.OnUsageAlert(func(UsageAlert) {})
	core.AssertLen(t, tenantService.alertHandlers, 1)
	core.AssertNotNil(t, tenantService.alertHandlers[0])
}

func TestAX7More_Tenant_OnUsageAlert_Bad(t *core.T) {
	tenantService := &Tenant{}
	tenantService.OnUsageAlert(nil)
	core.AssertLen(t, tenantService.alertHandlers, 0)
	core.AssertNotNil(t, tenantService)
}

func TestAX7More_Tenant_OnUsageAlert_Ugly(t *core.T) {
	var tenantService *Tenant
	core.AssertNotPanics(t, func() { tenantService.OnUsageAlert(func(UsageAlert) {}) })
	core.AssertNil(t, tenantService)
	core.AssertTrue(t, true)
}

func TestAX7More_Tenant_Scope_Good(t *core.T) {
	tenantService, _ := ax7Tenant(t)
	scope := tenantService.Scope()
	core.AssertNotNil(t, scope)
	core.AssertEqual(t, tenantService, scope.tenant)
	core.AssertTrue(t, scope.strict)
}

func TestAX7More_Tenant_Scope_Bad(t *core.T) {
	var tenantService *Tenant
	scope := tenantService.Scope()
	core.AssertNotNil(t, scope)
	core.AssertNil(t, scope.tenant)
	core.AssertTrue(t, scope.strict)
}

func TestAX7More_Tenant_Scope_Ugly(t *core.T) {
	tenantService := &Tenant{}
	scope := tenantService.Scope().WithStrict(false)
	core.AssertNotNil(t, scope)
	core.AssertFalse(t, scope.strict)
	core.AssertEqual(t, tenantService, scope.tenant)
}

func TestAX7More_NewWorkspaceScope_Good(t *core.T) {
	tenantService, _ := ax7Tenant(t)
	scope := NewWorkspaceScope(tenantService)
	core.AssertNotNil(t, scope)
	core.AssertEqual(t, tenantService, scope.tenant)
	core.AssertTrue(t, scope.strict)
}

func TestAX7More_NewWorkspaceScope_Bad(t *core.T) {
	scope := NewWorkspaceScope(nil)
	core.AssertNotNil(t, scope)
	core.AssertNil(t, scope.tenant)
	core.AssertTrue(t, scope.strict)
}

func TestAX7More_NewWorkspaceScope_Ugly(t *core.T) {
	scope := NewWorkspaceScope(&Tenant{}).WithStrict(false)
	core.AssertNotNil(t, scope)
	core.AssertFalse(t, scope.strict)
	core.AssertNotNil(t, scope.tenant)
}

func TestAX7More_WorkspaceScope_WithStrict_Good(t *core.T) {
	scope := NewWorkspaceScope(&Tenant{})
	got := scope.WithStrict(false)
	core.AssertEqual(t, scope, got)
	core.AssertFalse(t, scope.strict)
}

func TestAX7More_WorkspaceScope_WithStrict_Bad(t *core.T) {
	scope := NewWorkspaceScope(&Tenant{}).WithStrict(false)
	got := scope.WithStrict(true)
	core.AssertEqual(t, scope, got)
	core.AssertTrue(t, scope.strict)
}

func TestAX7More_WorkspaceScope_WithStrict_Ugly(t *core.T) {
	var scope *WorkspaceScope
	core.AssertPanics(t, func() { scope.WithStrict(false) })
	core.AssertNil(t, scope)
}

func TestAX7More_WorkspaceScope_Middleware_Good(t *core.T) {
	tenantService, _ := ax7Tenant(t)
	scope := NewWorkspaceScope(tenantService)
	called := false
	handler := scope.Middleware()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, err := WorkspaceFromCtx(r.Context())
		called = err == nil
	}))
	req := httptest.NewRequest(http.MethodGet, "http://acme.example.uk/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	core.AssertTrue(t, called)
	core.AssertEqual(t, http.StatusOK, rec.Code)
}

func TestAX7More_WorkspaceScope_Middleware_Bad(t *core.T) {
	scope := NewWorkspaceScope(&Tenant{}).WithStrict(true)
	called := false
	handler := scope.Middleware()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	req := httptest.NewRequest(http.MethodGet, "http://missing.example.uk/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	core.AssertFalse(t, called)
	core.AssertEqual(t, http.StatusUnauthorized, rec.Code)
}

func TestAX7More_WorkspaceScope_Middleware_Ugly(t *core.T) {
	scope := NewWorkspaceScope(&Tenant{}).WithStrict(false)
	called := false
	handler := scope.Middleware()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	req := httptest.NewRequest(http.MethodGet, "http://missing.example.uk/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	core.AssertTrue(t, called)
	core.AssertEqual(t, http.StatusOK, rec.Code)
}

func TestAX7More_WorkspaceScope_RequireWorkspace_Good(t *core.T) {
	scope := NewWorkspaceScope(&Tenant{})
	called := false
	handler := scope.RequireWorkspace()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(WithWorkspace(context.Background(), ax7Workspace()))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	core.AssertTrue(t, called)
	core.AssertEqual(t, http.StatusOK, rec.Code)
}

func TestAX7More_WorkspaceScope_RequireWorkspace_Bad(t *core.T) {
	scope := NewWorkspaceScope(&Tenant{})
	called := false
	handler := scope.RequireWorkspace()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	core.AssertFalse(t, called)
	core.AssertEqual(t, http.StatusUnauthorized, rec.Code)
}

func TestAX7More_WorkspaceScope_RequireWorkspace_Ugly(t *core.T) {
	var scope *WorkspaceScope
	handler := scope.RequireWorkspace()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	core.AssertEqual(t, http.StatusUnauthorized, rec.Code)
	core.AssertNil(t, scope)
}

func TestAX7More_Register_Good(t *core.T) {
	result := Register(core.New())
	tenantService, ok := result.Value.(*Tenant)
	core.AssertTrue(t, result.OK)
	core.AssertTrue(t, ok)
	core.AssertNotNil(t, tenantService.cache)
}

func TestAX7More_Register_Bad(t *core.T) {
	result := Register(nil)
	core.AssertFalse(t, result.OK)
	core.AssertNotNil(t, result.Value)
	core.AssertContains(t, result.Error(), "core is nil")
}

func TestAX7More_Register_Ugly(t *core.T) {
	c := core.New()
	result := Register(c)
	tenantService := result.Value.(*Tenant)
	core.AssertTrue(t, result.OK)
	core.AssertNotNil(t, tenantService.ServiceRuntime)
	core.AssertNotNil(t, tenantService.entitlements)
}

func TestAX7More_NewTenantClient_Good(t *core.T) {
	client := NewTenantClient("http://example.test///", "token")
	core.AssertNotNil(t, client)
	core.AssertEqual(t, "http://example.test", client.baseURL)
	core.AssertEqual(t, 10*time.Second, client.timeout)
}

func TestAX7More_NewTenantClient_Bad(t *core.T) {
	client := NewTenantClient("", "")
	core.AssertNotNil(t, client)
	core.AssertEqual(t, "", client.baseURL)
	core.AssertNotNil(t, client.httpClient)
}

func TestAX7More_NewTenantClient_Ugly(t *core.T) {
	client := NewTenantClient("http://example.test", "token", WithTimeout(2*time.Second))
	core.AssertNotNil(t, client)
	core.AssertEqual(t, 2*time.Second, client.timeout)
	core.AssertEqual(t, 2*time.Second, client.httpClient.Timeout)
}

func TestAX7More_WithTimeout_Good(t *core.T) {
	client := NewTenantClient("http://example.test", "token", WithTimeout(2*time.Second))
	core.AssertEqual(t, 2*time.Second, client.timeout)
	core.AssertNotNil(t, client.httpClient)
	core.AssertEqual(t, 2*time.Second, client.httpClient.Timeout)
}

func TestAX7More_WithTimeout_Bad(t *core.T) {
	client := NewTenantClient("http://example.test", "token", WithTimeout(0))
	core.AssertEqual(t, 10*time.Second, client.timeout)
	core.AssertNotNil(t, client.httpClient)
	core.AssertEqual(t, 10*time.Second, client.httpClient.Timeout)
}

func TestAX7More_WithTimeout_Ugly(t *core.T) {
	client := NewTenantClient("http://example.test", "token", WithTimeout(-1*time.Second))
	core.AssertEqual(t, 10*time.Second, client.timeout)
	core.AssertNotNil(t, client.httpClient)
	core.AssertEqual(t, 10*time.Second, client.httpClient.Timeout)
}

func TestAX7More_TenantClient_GetWorkspaceBySlug_Good(t *core.T) {
	server := ax7JSONServer(http.StatusOK, `{"id":7,"uuid":"uuid-7","slug":"acme","name":"Acme"}`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	got, err := client.GetWorkspaceBySlug(context.Background(), "acme")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "uuid-7", got.UUID)
}

func TestAX7More_TenantClient_GetWorkspaceBySlug_Bad(t *core.T) {
	server := ax7JSONServer(http.StatusNotFound, `{"error":"missing"}`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	got, err := client.GetWorkspaceBySlug(context.Background(), "missing")
	core.AssertErrorIs(t, err, ErrWorkspaceNotFound)
	core.AssertNil(t, got)
}

func TestAX7More_TenantClient_GetWorkspaceBySlug_Ugly(t *core.T) {
	server := ax7JSONServer(http.StatusOK, `{bad json`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	got, err := client.GetWorkspaceBySlug(context.Background(), "acme")
	core.AssertError(t, err)
	core.AssertNil(t, got)
}

func TestAX7More_TenantClient_GetWorkspaceByUUID_Good(t *core.T) {
	server := ax7JSONServer(http.StatusOK, `{"id":7,"uuid":"uuid-7","slug":"acme"}`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	got, err := client.GetWorkspaceByUUID(context.Background(), "uuid-7")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "acme", got.Slug)
}

func TestAX7More_TenantClient_GetWorkspaceByUUID_Bad(t *core.T) {
	server := ax7JSONServer(http.StatusNotFound, `{"error":"missing"}`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	got, err := client.GetWorkspaceByUUID(context.Background(), "missing")
	core.AssertErrorIs(t, err, ErrWorkspaceNotFound)
	core.AssertNil(t, got)
}

func TestAX7More_TenantClient_GetWorkspaceByUUID_Ugly(t *core.T) {
	server := ax7JSONServer(http.StatusOK, ``)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	got, err := client.GetWorkspaceByUUID(context.Background(), "uuid-7")
	core.AssertErrorIs(t, err, core.EOF)
	core.AssertNil(t, got)
}

func TestAX7More_TenantClient_GetWorkspaceByID_Good(t *core.T) {
	server := ax7JSONServer(http.StatusOK, `{"id":7,"uuid":"uuid-7","slug":"acme"}`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	got, err := client.GetWorkspaceByID(context.Background(), 7)
	core.AssertNoError(t, err)
	core.AssertEqual(t, int64(7), got.ID)
}

func TestAX7More_TenantClient_GetWorkspaceByID_Bad(t *core.T) {
	server := ax7JSONServer(http.StatusNotFound, `{"error":"missing"}`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	got, err := client.GetWorkspaceByID(context.Background(), 99)
	core.AssertErrorIs(t, err, ErrWorkspaceNotFound)
	core.AssertNil(t, got)
}

func TestAX7More_TenantClient_GetWorkspaceByID_Ugly(t *core.T) {
	server := ax7JSONServer(http.StatusOK, `{bad json`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	got, err := client.GetWorkspaceByID(context.Background(), 7)
	core.AssertError(t, err)
	core.AssertNil(t, got)
}

func TestAX7More_TenantClient_GetWorkspaceBySubdomain_Good(t *core.T) {
	server := ax7JSONServer(http.StatusOK, `{"id":7,"uuid":"uuid-7","slug":"acme"}`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	got, err := client.GetWorkspaceBySubdomain(context.Background(), "acme.example.uk")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "acme", got.Slug)
}

func TestAX7More_TenantClient_GetWorkspaceBySubdomain_Bad(t *core.T) {
	server := ax7JSONServer(http.StatusNotFound, `{"error":"missing"}`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	got, err := client.GetWorkspaceBySubdomain(context.Background(), "missing.example.uk")
	core.AssertErrorIs(t, err, ErrWorkspaceNotFound)
	core.AssertNil(t, got)
}

func TestAX7More_TenantClient_GetWorkspaceBySubdomain_Ugly(t *core.T) {
	server := ax7JSONServer(http.StatusOK, `{bad json`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	got, err := client.GetWorkspaceBySubdomain(context.Background(), "acme.example.uk")
	core.AssertError(t, err)
	core.AssertNil(t, got)
}

func TestAX7More_TenantClient_GetUser_Good(t *core.T) {
	server := ax7JSONServer(http.StatusOK, `{"uuid":"user-7","email":"ada@example.uk"}`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	got, err := client.GetUser(context.Background())
	core.AssertNoError(t, err)
	core.AssertEqual(t, "user-7", got.UUID)
}

func TestAX7More_TenantClient_GetUser_Bad(t *core.T) {
	server := ax7JSONServer(http.StatusUnauthorized, `{"error":"unauthorized"}`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	got, err := client.GetUser(context.Background())
	core.AssertError(t, err)
	core.AssertNil(t, got)
}

func TestAX7More_TenantClient_GetUser_Ugly(t *core.T) {
	server := ax7JSONServer(http.StatusOK, ``)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	got, err := client.GetUser(context.Background())
	core.AssertErrorIs(t, err, core.EOF)
	core.AssertNil(t, got)
}

func TestAX7More_TenantClient_GetPackagesForWorkspace_Good(t *core.T) {
	server := ax7JSONServer(http.StatusOK, `[{"code":"starter","is_active":true}]`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	got, err := client.GetPackagesForWorkspace(context.Background(), "uuid-7")
	core.AssertNoError(t, err)
	core.AssertLen(t, got, 1)
}

func TestAX7More_TenantClient_GetPackagesForWorkspace_Bad(t *core.T) {
	server := ax7JSONServer(http.StatusNotFound, `{"error":"missing"}`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	got, err := client.GetPackagesForWorkspace(context.Background(), "missing")
	core.AssertError(t, err)
	core.AssertNil(t, got)
}

func TestAX7More_TenantClient_GetPackagesForWorkspace_Ugly(t *core.T) {
	server := ax7JSONServer(http.StatusOK, `{bad json`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	got, err := client.GetPackagesForWorkspace(context.Background(), "uuid-7")
	core.AssertError(t, err)
	core.AssertNil(t, got)
}

func TestAX7More_TenantClient_GetBoostsForWorkspace_Good(t *core.T) {
	server := ax7JSONServer(http.StatusOK, `[{"feature_code":"pages","status":"active","boost_type":"add_limit","limit_value":5}]`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	got, err := client.GetBoostsForWorkspace(context.Background(), "uuid-7")
	core.AssertNoError(t, err)
	core.AssertLen(t, got, 1)
}

func TestAX7More_TenantClient_GetBoostsForWorkspace_Bad(t *core.T) {
	server := ax7JSONServer(http.StatusNotFound, `{"error":"missing"}`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	got, err := client.GetBoostsForWorkspace(context.Background(), "missing")
	core.AssertError(t, err)
	core.AssertNil(t, got)
}

func TestAX7More_TenantClient_GetBoostsForWorkspace_Ugly(t *core.T) {
	server := ax7JSONServer(http.StatusOK, `{bad json`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	got, err := client.GetBoostsForWorkspace(context.Background(), "uuid-7")
	core.AssertError(t, err)
	core.AssertNil(t, got)
}

func TestAX7More_TenantClient_GetCurrentUsage_Good(t *core.T) {
	server := ax7JSONServer(http.StatusOK, `{"ok":true,"count":4}`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	got, err := client.GetCurrentUsage(context.Background(), "uuid-7", "pages")
	core.AssertNoError(t, err)
	core.AssertEqual(t, 4, got)
}

func TestAX7More_TenantClient_GetCurrentUsage_Bad(t *core.T) {
	server := ax7JSONServer(http.StatusNotFound, `{"error":"missing"}`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	got, err := client.GetCurrentUsage(context.Background(), "missing", "pages")
	core.AssertError(t, err)
	core.AssertEqual(t, 0, got)
}

func TestAX7More_TenantClient_GetCurrentUsage_Ugly(t *core.T) {
	server := ax7JSONServer(http.StatusOK, `9`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	got, err := client.GetCurrentUsage(context.Background(), "uuid-7", " PAGES ")
	core.AssertNoError(t, err)
	core.AssertEqual(t, 9, got)
}

func TestAX7More_TenantClient_RecordUsage_Good(t *core.T) {
	server := ax7JSONServer(http.StatusOK, `{"ok":true}`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	err := client.RecordUsage(context.Background(), "uuid-7", "pages", 2, nil, nil)
	core.AssertNoError(t, err)
	core.AssertNotNil(t, client.httpClient)
}

func TestAX7More_TenantClient_RecordUsage_Bad(t *core.T) {
	server := ax7JSONServer(http.StatusInternalServerError, `{"error":"failed"}`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	err := client.RecordUsage(context.Background(), "uuid-7", "pages", 1, nil, nil)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed")
}

func TestAX7More_TenantClient_RecordUsage_Ugly(t *core.T) {
	server := ax7JSONServer(http.StatusOK, `{"ok":true}`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	userID := int64(42)
	err := client.RecordUsage(context.Background(), "uuid-7", " PAGES ", 0, &userID, map[string]any{"source": "test"})
	core.AssertNoError(t, err)
	core.AssertNotNil(t, client.httpClient)
}

func TestAX7More_TenantClient_GetFeature_Good(t *core.T) {
	server := ax7JSONServer(http.StatusOK, `{"code":"pages","name":"Pages","type":"limit","is_active":true}`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	got, err := client.GetFeature(context.Background(), "pages")
	core.AssertNoError(t, err)
	core.AssertEqual(t, FeatureTypeLimit, got.Type)
}

func TestAX7More_TenantClient_GetFeature_Bad(t *core.T) {
	server := ax7JSONServer(http.StatusNotFound, `{"error":"missing"}`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	got, err := client.GetFeature(context.Background(), "missing")
	core.AssertErrorIs(t, err, ErrFeatureNotFound)
	core.AssertNil(t, got)
}

func TestAX7More_TenantClient_GetFeature_Ugly(t *core.T) {
	server := ax7JSONServer(http.StatusOK, `{bad json`)
	defer server.Close()
	client := NewTenantClient(server.URL, "token")
	got, err := client.GetFeature(context.Background(), "pages")
	core.AssertError(t, err)
	core.AssertNil(t, got)
}
