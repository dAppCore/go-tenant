// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"context"
	"time"

	"dappco.re/go"
)

// expire forces every in-memory cache entry to be treated as stale so the
// next Get falls through the expiry-delete branch. With a nil store there is
// no rehydration, so the lookup then reports a miss.
func expire(cache *TenantCache) {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	past := time.Now().Add(-time.Hour)
	for key, entry := range cache.workspacesByUUID {
		entry.expiresAt = past
		cache.workspacesByUUID[key] = entry
	}
	for key, entry := range cache.packages {
		entry.expiresAt = past
		cache.packages[key] = entry
	}
	for key, entry := range cache.boosts {
		entry.expiresAt = past
		cache.boosts[key] = entry
	}
	for key, entry := range cache.usage {
		entry.expiresAt = past
		cache.usage[key] = entry
	}
	for key, entry := range cache.features {
		entry.expiresAt = past
		cache.features[key] = entry
	}
	for key, entry := range cache.users {
		entry.expiresAt = past
		cache.users[key] = entry
	}
}

func TestCacheExpiry_GetWorkspace_Expired_Ugly(t *core.T) {
	cache, _ := testCache(t)
	expire(cache)
	workspace, ok := cache.GetWorkspace("uuid-7")
	core.AssertFalse(t, ok)
	core.AssertNil(t, workspace)
}

func TestCacheExpiry_GetPackages_Expired_Ugly(t *core.T) {
	cache, _ := testCache(t)
	expire(cache)
	packages, ok := cache.GetPackages("uuid-7")
	core.AssertFalse(t, ok)
	core.AssertNil(t, packages)
}

func TestCacheExpiry_GetBoosts_Expired_Ugly(t *core.T) {
	cache, workspace := testCache(t)
	requireResultOK(t, cache.SetBoosts(workspace.UUID, []Boost{{FeatureCode: "pages"}}))
	expire(cache)
	boosts, ok := cache.GetBoosts(workspace.UUID)
	core.AssertFalse(t, ok)
	core.AssertNil(t, boosts)
}

func TestCacheExpiry_GetUsage_Expired_Ugly(t *core.T) {
	cache, _ := testCache(t)
	expire(cache)
	used, ok := cache.GetUsage("uuid-7", "pages")
	core.AssertFalse(t, ok)
	core.AssertEqual(t, 0, used)
}

func TestCacheExpiry_GetFeature_Expired_Ugly(t *core.T) {
	cache, _ := testCache(t)
	expire(cache)
	feature, ok := cache.GetFeature("pages")
	core.AssertFalse(t, ok)
	core.AssertNil(t, feature)
}

func TestCacheExpiry_GetUser_Expired_Ugly(t *core.T) {
	cache, _ := testCache(t)
	requireResultOK(t, cache.SetUser(testUser()))
	expire(cache)
	user, ok := cache.GetUser("user-9")
	core.AssertFalse(t, ok)
	core.AssertNil(t, user)
}

func TestCacheExpiry_SummaryUnlimitedFeatureType_Ugly(t *core.T) {
	// An unlimited-type feature assigned to a package reports no limit and an
	// observed usage count.
	cache, workspace := testCache(t)
	requireResultOK(t, cache.SetFeature(&Feature{Code: "seats", Name: "Seats", Type: FeatureTypeUnlimited, IsActive: true}))
	requireResultOK(t, cache.SetPackages(workspace.UUID, []Package{{
		Code: "pro", IsActive: true, Features: []PackageFeature{{FeatureCode: "seats", LimitValue: testInt(0)}},
	}}))
	requireResultOK(t, cache.SetUsage(workspace.UUID, "seats", 2))
	svc := NewLocalEntitlementService(cache, nil)
	result := svc.GetUsageSummary(context.Background(), workspace)
	requireResultOK(t, result)
	for _, item := range result.Value.([]UsageSummaryItem) {
		if item.FeatureCode == "seats" {
			core.AssertTrue(t, item.Unlimited)
			core.AssertNil(t, item.Limit)
			core.AssertEqual(t, 2, *item.Used)
		}
	}
}

func TestCacheExpiry_CanBooleanFeatureNotInPackage_Bad(t *core.T) {
	cache, workspace := testCache(t)
	requireResultOK(t, cache.SetFeature(&Feature{Code: "sso", Name: "SSO", Type: FeatureTypeBoolean, IsActive: true}))
	svc := NewLocalEntitlementService(cache, nil)
	result := svc.Can(context.Background(), workspace, "sso", 1)
	core.AssertTrue(t, result.IsDenied())
	core.AssertEqual(t, "feature not in any package", result.Reason)
}

func TestCacheExpiry_CanUnlimitedFeatureInPackage_Good(t *core.T) {
	cache, workspace := testCache(t)
	requireResultOK(t, cache.SetFeature(&Feature{Code: "seats", Name: "Seats", Type: FeatureTypeUnlimited, IsActive: true}))
	requireResultOK(t, cache.SetPackages(workspace.UUID, []Package{{
		Code: "pro", IsActive: true, Features: []PackageFeature{{FeatureCode: "seats", LimitValue: testInt(0)}},
	}}))
	svc := NewLocalEntitlementService(cache, nil)
	result := svc.Can(context.Background(), workspace, "seats", 1)
	core.AssertTrue(t, result.IsAllowed())
	core.AssertTrue(t, result.Unlimited)
}
