// SPDX-License-Identifier: EUPL-1.2

package tenant

import "dappco.re/go"

func TestCache_NewTenantCache_Good(t *core.T) {
	cache := NewTenantCache(nil)
	core.AssertNotNil(t, cache)
	core.AssertNotNil(t, cache.workspacesByUUID)
}

func TestCache_NewTenantCache_Bad(t *core.T) {
	cache := NewTenantCache(nil)
	got, ok := cache.GetWorkspace("missing")
	core.AssertFalse(t, ok)
	core.AssertNil(t, got)
}

func TestCache_NewTenantCache_Ugly(t *core.T) {
	cache := NewTenantCache(nil)
	core.AssertNotNil(t, cache.packages)
	core.AssertNotNil(t, cache.usage)
}

func TestCache_TenantCache_SetWorkspace_Good(t *core.T) {
	cache := NewTenantCache(nil)
	result := cache.SetWorkspace(testWorkspace())
	requireResultOK(t, result)
	got, ok := cache.GetWorkspace("uuid-7")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "acme", got.Slug)
}

func TestCache_TenantCache_SetWorkspace_Bad(t *core.T) {
	cache := NewTenantCache(nil)
	result := cache.SetWorkspace(nil)
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoWorkspaceContext, result.Value)
}

func TestCache_TenantCache_SetWorkspace_Ugly(t *core.T) {
	cache := NewTenantCache(nil)
	workspace := testWorkspace()
	requireResultOK(t, cache.SetWorkspace(workspace))
	workspace.Slug = "mutated"
	got, ok := cache.GetWorkspace("uuid-7")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "acme", got.Slug)
}

func TestCache_TenantCache_GetWorkspace_Good(t *core.T) {
	cache, workspace := testCache(t)
	got, ok := cache.GetWorkspace(workspace.UUID)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, workspace.UUID, got.UUID)
}

func TestCache_TenantCache_GetWorkspace_Bad(t *core.T) {
	cache := NewTenantCache(nil)
	got, ok := cache.GetWorkspace("missing")
	core.AssertFalse(t, ok)
	core.AssertNil(t, got)
}

func TestCache_TenantCache_GetWorkspace_Ugly(t *core.T) {
	cache, workspace := testCache(t)
	got, ok := cache.GetWorkspace(workspace.UUID)
	got.Slug = "changed"
	again, againOK := cache.GetWorkspace(workspace.UUID)
	core.AssertTrue(t, ok && againOK)
	core.AssertEqual(t, "acme", again.Slug)
}

func TestCache_TenantCache_GetWorkspaceByID_Good(t *core.T) {
	cache, workspace := testCache(t)
	got, ok := cache.GetWorkspaceByID(workspace.ID)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, workspace.UUID, got.UUID)
}

func TestCache_TenantCache_GetWorkspaceByID_Bad(t *core.T) {
	cache := NewTenantCache(nil)
	got, ok := cache.GetWorkspaceByID(404)
	core.AssertFalse(t, ok)
	core.AssertNil(t, got)
}

func TestCache_TenantCache_GetWorkspaceByID_Ugly(t *core.T) {
	cache, workspace := testCache(t)
	workspace.ID = 8
	requireResultOK(t, cache.SetWorkspace(workspace))
	old, oldOK := cache.GetWorkspaceByID(7)
	got, ok := cache.GetWorkspaceByID(8)
	core.AssertFalse(t, oldOK)
	core.AssertNil(t, old)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, int64(8), got.ID)
}

func TestCache_TenantCache_GetWorkspaceBySlug_Good(t *core.T) {
	cache, workspace := testCache(t)
	got, ok := cache.GetWorkspaceBySlug(workspace.Slug)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, workspace.UUID, got.UUID)
}

func TestCache_TenantCache_GetWorkspaceBySlug_Bad(t *core.T) {
	cache := NewTenantCache(nil)
	got, ok := cache.GetWorkspaceBySlug("missing")
	core.AssertFalse(t, ok)
	core.AssertNil(t, got)
}

func TestCache_TenantCache_GetWorkspaceBySlug_Ugly(t *core.T) {
	cache, workspace := testCache(t)
	workspace.Slug = "beta"
	requireResultOK(t, cache.SetWorkspace(workspace))
	old, oldOK := cache.GetWorkspaceBySlug("acme")
	got, ok := cache.GetWorkspaceBySlug("beta")
	core.AssertFalse(t, oldOK)
	core.AssertNil(t, old)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "beta", got.Slug)
}

func TestCache_TenantCache_SetPackages_Good(t *core.T) {
	cache := NewTenantCache(nil)
	result := cache.SetPackages("uuid-7", []Package{{Code: "starter", IsActive: true}})
	requireResultOK(t, result)
	got, ok := cache.GetPackages("uuid-7")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "starter", got[0].Code)
}

func TestCache_TenantCache_SetPackages_Bad(t *core.T) {
	var cache *TenantCache
	result := cache.SetPackages("uuid-7", nil)
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoWorkspaceContext, result.Value)
}

func TestCache_TenantCache_SetPackages_Ugly(t *core.T) {
	cache := NewTenantCache(nil)
	packages := []Package{{Code: "starter", IsActive: true}}
	requireResultOK(t, cache.SetPackages("uuid-7", packages))
	packages[0].Code = "mutated"
	got, ok := cache.GetPackages("uuid-7")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "starter", got[0].Code)
}

func TestCache_TenantCache_GetPackages_Good(t *core.T) {
	cache, workspace := testCache(t)
	got, ok := cache.GetPackages(workspace.UUID)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "starter", got[0].Code)
}

func TestCache_TenantCache_GetPackages_Bad(t *core.T) {
	cache := NewTenantCache(nil)
	got, ok := cache.GetPackages("missing")
	core.AssertFalse(t, ok)
	core.AssertNil(t, got)
}

func TestCache_TenantCache_GetPackages_Ugly(t *core.T) {
	cache, workspace := testCache(t)
	got, ok := cache.GetPackages(workspace.UUID)
	got[0].Code = "mutated"
	again, againOK := cache.GetPackages(workspace.UUID)
	core.AssertTrue(t, ok && againOK)
	core.AssertEqual(t, "starter", again[0].Code)
}

func TestCache_TenantCache_SetBoosts_Good(t *core.T) {
	cache := NewTenantCache(nil)
	result := cache.SetBoosts("uuid-7", []Boost{{FeatureCode: "pages", Status: BoostStatusActive}})
	requireResultOK(t, result)
	got, ok := cache.GetBoosts("uuid-7")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "pages", got[0].FeatureCode)
}

func TestCache_TenantCache_SetBoosts_Bad(t *core.T) {
	var cache *TenantCache
	result := cache.SetBoosts("uuid-7", nil)
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoWorkspaceContext, result.Value)
}

func TestCache_TenantCache_SetBoosts_Ugly(t *core.T) {
	cache := NewTenantCache(nil)
	boosts := []Boost{{FeatureCode: "pages", Status: BoostStatusActive}}
	requireResultOK(t, cache.SetBoosts("uuid-7", boosts))
	boosts[0].FeatureCode = "mutated"
	got, ok := cache.GetBoosts("uuid-7")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "pages", got[0].FeatureCode)
}

func TestCache_TenantCache_GetBoosts_Good(t *core.T) {
	cache, workspace := testCache(t)
	requireResultOK(t, cache.SetBoosts(workspace.UUID, []Boost{{FeatureCode: "pages", Status: BoostStatusActive}}))
	got, ok := cache.GetBoosts(workspace.UUID)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "pages", got[0].FeatureCode)
}

func TestCache_TenantCache_GetBoosts_Bad(t *core.T) {
	cache := NewTenantCache(nil)
	got, ok := cache.GetBoosts("missing")
	core.AssertFalse(t, ok)
	core.AssertNil(t, got)
}

func TestCache_TenantCache_GetBoosts_Ugly(t *core.T) {
	cache, workspace := testCache(t)
	boosts := []Boost{{FeatureCode: "pages", Status: BoostStatusActive}}
	requireResultOK(t, cache.SetBoosts(workspace.UUID, boosts))
	boosts[0].Status = BoostStatusExpired
	got, ok := cache.GetBoosts(workspace.UUID)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, BoostStatusActive, got[0].Status)
}

func TestCache_TenantCache_SetUsage_Good(t *core.T) {
	cache := NewTenantCache(nil)
	result := cache.SetUsage("uuid-7", "Pages", 4)
	requireResultOK(t, result)
	got, ok := cache.GetUsage("uuid-7", "pages")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, 4, got)
}

func TestCache_TenantCache_SetUsage_Bad(t *core.T) {
	var cache *TenantCache
	result := cache.SetUsage("uuid-7", "pages", 1)
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoWorkspaceContext, result.Value)
}

func TestCache_TenantCache_SetUsage_Ugly(t *core.T) {
	cache := NewTenantCache(nil)
	requireResultOK(t, cache.SetUsage("uuid-7", " Pages ", 2))
	got, ok := cache.GetUsage("uuid-7", "pages")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, 2, got)
}

func TestCache_TenantCache_GetUsage_Good(t *core.T) {
	cache, workspace := testCache(t)
	got, ok := cache.GetUsage(workspace.UUID, "pages")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, 3, got)
}

func TestCache_TenantCache_GetUsage_Bad(t *core.T) {
	cache := NewTenantCache(nil)
	got, ok := cache.GetUsage("missing", "pages")
	core.AssertFalse(t, ok)
	core.AssertEqual(t, 0, got)
}

func TestCache_TenantCache_GetUsage_Ugly(t *core.T) {
	cache := NewTenantCache(nil)
	requireResultOK(t, cache.SetUsage("uuid-7", " Pages ", 9))
	got, ok := cache.GetUsage("uuid-7", "pages")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, 9, got)
}

func TestCache_TenantCache_InvalidateWorkspace_Good(t *core.T) {
	cache, workspace := testCache(t)
	requireResultOK(t, cache.InvalidateWorkspace(workspace.UUID))
	got, ok := cache.GetWorkspace(workspace.UUID)
	core.AssertFalse(t, ok)
	core.AssertNil(t, got)
}

func TestCache_TenantCache_InvalidateWorkspace_Bad(t *core.T) {
	var cache *TenantCache
	result := cache.InvalidateWorkspace("missing")
	requireResultOK(t, result)
	core.AssertNil(t, cache)
}

func TestCache_TenantCache_InvalidateWorkspace_Ugly(t *core.T) {
	cache, workspace := testCache(t)
	requireResultOK(t, cache.SetBoosts(workspace.UUID, []Boost{{FeatureCode: "pages"}}))
	requireResultOK(t, cache.InvalidateWorkspace(workspace.UUID))
	_, ok := cache.GetBoosts(workspace.UUID)
	core.AssertFalse(t, ok)
}

func TestCache_TenantCache_SetFeature_Good(t *core.T) {
	cache := NewTenantCache(nil)
	result := cache.SetFeature(testLimitFeature())
	requireResultOK(t, result)
	got, ok := cache.GetFeature("pages")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, FeatureTypeLimit, got.Type)
}

func TestCache_TenantCache_SetFeature_Bad(t *core.T) {
	var cache *TenantCache
	result := cache.SetFeature(testLimitFeature())
	requireResultFail(t, result)
	core.AssertEqual(t, ErrFeatureNotFound, result.Value)
}

func TestCache_TenantCache_SetFeature_Ugly(t *core.T) {
	cache := NewTenantCache(nil)
	feature := testLimitFeature()
	requireResultOK(t, cache.SetFeature(feature))
	feature.Type = FeatureTypeBoolean
	got, ok := cache.GetFeature("pages")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, FeatureTypeLimit, got.Type)
}

func TestCache_TenantCache_GetFeature_Good(t *core.T) {
	cache, _ := testCache(t)
	got, ok := cache.GetFeature("pages")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "pages", got.Code)
}

func TestCache_TenantCache_GetFeature_Bad(t *core.T) {
	cache := NewTenantCache(nil)
	got, ok := cache.GetFeature("missing")
	core.AssertFalse(t, ok)
	core.AssertNil(t, got)
}

func TestCache_TenantCache_GetFeature_Ugly(t *core.T) {
	cache, _ := testCache(t)
	got, ok := cache.GetFeature(" Pages ")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "pages", got.Code)
}

func TestCache_TenantCache_SetUser_Good(t *core.T) {
	cache := NewTenantCache(nil)
	result := cache.SetUser(testUser())
	requireResultOK(t, result)
	got, ok := cache.GetUser("user-9")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "ada@example.uk", got.Email)
}

func TestCache_TenantCache_SetUser_Bad(t *core.T) {
	cache := NewTenantCache(nil)
	result := cache.SetUser(nil)
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoUserContext, result.Value)
}

func TestCache_TenantCache_SetUser_Ugly(t *core.T) {
	cache := NewTenantCache(nil)
	result := cache.SetUser(&User{Name: "Ada"})
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoUserContext, result.Value)
}

func TestCache_TenantCache_GetUser_Good(t *core.T) {
	cache := NewTenantCache(nil)
	requireResultOK(t, cache.SetUser(testUser()))
	got, ok := cache.GetUser("user-9")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "ada@example.uk", got.Email)
}

func TestCache_TenantCache_GetUser_Bad(t *core.T) {
	cache := NewTenantCache(nil)
	got, ok := cache.GetUser("missing")
	core.AssertFalse(t, ok)
	core.AssertNil(t, got)
}

func TestCache_TenantCache_GetUser_Ugly(t *core.T) {
	cache := NewTenantCache(nil)
	user := testUser()
	requireResultOK(t, cache.SetUser(user))
	user.Email = "mutated@example.uk"
	got, ok := cache.GetUser("user-9")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "ada@example.uk", got.Email)
}
