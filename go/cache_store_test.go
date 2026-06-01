// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"dappco.re/go"
	"dappco.re/go/store"
)

// testStore opens an in-memory go-store for the store-backed cache paths.
// The second return is a fresh cache sharing the same store, used to force
// reads through go-store instead of the in-memory hot path.
func testStore(t *core.T) *store.Store {
	t.Helper()
	result := store.New(":memory:")
	requireResultOK(t, result)
	return result.Value.(*store.Store)
}

func TestCacheStore_SetWorkspace_PersistsAndRehydrates_Good(t *core.T) {
	st := testStore(t)
	writer := NewTenantCache(st)
	requireResultOK(t, writer.SetWorkspace(testWorkspace()))

	reader := NewTenantCache(st)
	workspace, ok := reader.GetWorkspace("uuid-7")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "acme", workspace.Slug)
}

func TestCacheStore_GetWorkspaceByID_RehydratesFromStore_Good(t *core.T) {
	st := testStore(t)
	requireResultOK(t, NewTenantCache(st).SetWorkspace(testWorkspace()))

	reader := NewTenantCache(st)
	workspace, ok := reader.GetWorkspaceByID(7)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "uuid-7", workspace.UUID)
}

func TestCacheStore_GetWorkspaceBySlug_RehydratesFromStore_Good(t *core.T) {
	st := testStore(t)
	requireResultOK(t, NewTenantCache(st).SetWorkspace(testWorkspace()))

	reader := NewTenantCache(st)
	workspace, ok := reader.GetWorkspaceBySlug("acme")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "uuid-7", workspace.UUID)
}

func TestCacheStore_GetWorkspace_Miss_Bad(t *core.T) {
	reader := NewTenantCache(testStore(t))
	workspace, ok := reader.GetWorkspace("never-stored")
	core.AssertFalse(t, ok)
	core.AssertNil(t, workspace)
}

func TestCacheStore_GetWorkspaceByID_Miss_Bad(t *core.T) {
	reader := NewTenantCache(testStore(t))
	workspace, ok := reader.GetWorkspaceByID(999)
	core.AssertFalse(t, ok)
	core.AssertNil(t, workspace)
}

func TestCacheStore_GetWorkspaceBySlug_Miss_Bad(t *core.T) {
	reader := NewTenantCache(testStore(t))
	workspace, ok := reader.GetWorkspaceBySlug("ghost")
	core.AssertFalse(t, ok)
	core.AssertNil(t, workspace)
}

func TestCacheStore_Packages_PersistsAndRehydrates_Good(t *core.T) {
	st := testStore(t)
	writer := NewTenantCache(st)
	requireResultOK(t, writer.SetPackages("uuid-7", []Package{{Code: "starter", IsActive: true}}))

	reader := NewTenantCache(st)
	packages, ok := reader.GetPackages("uuid-7")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "starter", packages[0].Code)
}

func TestCacheStore_Packages_Miss_Bad(t *core.T) {
	reader := NewTenantCache(testStore(t))
	packages, ok := reader.GetPackages("uuid-7")
	core.AssertFalse(t, ok)
	core.AssertNil(t, packages)
}

func TestCacheStore_Boosts_PersistsAndRehydrates_Good(t *core.T) {
	st := testStore(t)
	writer := NewTenantCache(st)
	requireResultOK(t, writer.SetBoosts("uuid-7", []Boost{{FeatureCode: "pages"}}))

	reader := NewTenantCache(st)
	boosts, ok := reader.GetBoosts("uuid-7")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "pages", boosts[0].FeatureCode)
}

func TestCacheStore_Boosts_Miss_Bad(t *core.T) {
	reader := NewTenantCache(testStore(t))
	boosts, ok := reader.GetBoosts("uuid-7")
	core.AssertFalse(t, ok)
	core.AssertNil(t, boosts)
}

func TestCacheStore_Usage_PersistsAndRehydrates_Good(t *core.T) {
	st := testStore(t)
	writer := NewTenantCache(st)
	requireResultOK(t, writer.SetUsage("uuid-7", "pages", 4))

	reader := NewTenantCache(st)
	used, ok := reader.GetUsage("uuid-7", "pages")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, 4, used)
}

func TestCacheStore_Usage_Miss_Bad(t *core.T) {
	reader := NewTenantCache(testStore(t))
	used, ok := reader.GetUsage("uuid-7", "pages")
	core.AssertFalse(t, ok)
	core.AssertEqual(t, 0, used)
}

func TestCacheStore_Feature_PersistsAndRehydrates_Good(t *core.T) {
	st := testStore(t)
	writer := NewTenantCache(st)
	requireResultOK(t, writer.SetFeature(testLimitFeature()))

	reader := NewTenantCache(st)
	feature, ok := reader.GetFeature("pages")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "Pages", feature.Name)
}

func TestCacheStore_Feature_Miss_Bad(t *core.T) {
	reader := NewTenantCache(testStore(t))
	feature, ok := reader.GetFeature("pages")
	core.AssertFalse(t, ok)
	core.AssertNil(t, feature)
}

func TestCacheStore_User_PersistsAndRehydrates_Good(t *core.T) {
	st := testStore(t)
	writer := NewTenantCache(st)
	requireResultOK(t, writer.SetUser(testUser()))

	reader := NewTenantCache(st)
	user, ok := reader.GetUser("user-9")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "ada@example.uk", user.Email)
}

func TestCacheStore_User_Miss_Bad(t *core.T) {
	reader := NewTenantCache(testStore(t))
	user, ok := reader.GetUser("user-9")
	core.AssertFalse(t, ok)
	core.AssertNil(t, user)
}

func TestCacheStore_InvalidateWorkspace_ClearsStore_Ugly(t *core.T) {
	st := testStore(t)
	writer := NewTenantCache(st)
	requireResultOK(t, writer.SetWorkspace(testWorkspace()))
	requireResultOK(t, writer.SetPackages("uuid-7", []Package{{Code: "starter", IsActive: true}}))
	requireResultOK(t, writer.SetBoosts("uuid-7", []Boost{{FeatureCode: "pages"}}))
	requireResultOK(t, writer.SetUsage("uuid-7", "pages", 4))
	requireResultOK(t, writer.InvalidateWorkspace("uuid-7"))

	reader := NewTenantCache(st)
	_, wsOK := reader.GetWorkspace("uuid-7")
	core.AssertFalse(t, wsOK)
	_, pkgOK := reader.GetPackages("uuid-7")
	core.AssertFalse(t, pkgOK)
	_, boostOK := reader.GetBoosts("uuid-7")
	core.AssertFalse(t, boostOK)
	_, usageOK := reader.GetUsage("uuid-7", "pages")
	core.AssertFalse(t, usageOK)
}

func TestCacheStore_invalidateUsage_ClearsStore_Ugly(t *core.T) {
	st := testStore(t)
	writer := NewTenantCache(st)
	requireResultOK(t, writer.SetUsage("uuid-7", "pages", 4))
	requireResultOK(t, writer.invalidateUsage("uuid-7", "pages"))

	reader := NewTenantCache(st)
	_, ok := reader.GetUsage("uuid-7", "pages")
	core.AssertFalse(t, ok)
}
