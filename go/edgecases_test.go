// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	"dappco.re/go"
)

func TestEdge_cloneWorkspace_Good(t *core.T) {
	original := &Workspace{UUID: "uuid-7", Settings: map[string]any{"theme": "dark"}}
	clone := cloneWorkspace(original)
	clone.Settings["theme"] = "light"
	core.AssertEqual(t, "dark", original.Settings["theme"])
}

func TestEdge_cloneWorkspace_Bad(t *core.T) {
	core.AssertNil(t, cloneWorkspace(nil))
}

func TestEdge_cloneWorkspace_Ugly(t *core.T) {
	clone := cloneWorkspace(&Workspace{UUID: "uuid-7"})
	core.AssertNil(t, clone.Settings)
	core.AssertEqual(t, "uuid-7", clone.UUID)
}

func TestEdge_cloneFeature_Bad(t *core.T) {
	core.AssertNil(t, cloneFeature(nil))
}

func TestEdge_cloneUser_Bad(t *core.T) {
	core.AssertNil(t, cloneUser(nil))
}

func TestEdge_hasUsagePrefix_Good(t *core.T) {
	core.AssertTrue(t, hasUsagePrefix("uuid-7\x00pages", "uuid-7"))
}

func TestEdge_hasUsagePrefix_Bad(t *core.T) {
	core.AssertFalse(t, hasUsagePrefix("uuid-9\x00pages", "uuid-7"))
}

func TestEdge_hasUsagePrefix_Ugly(t *core.T) {
	// Key shorter than the workspace UUID plus separator.
	core.AssertFalse(t, hasUsagePrefix("u", "uuid-7"))
}

func TestEdge_hasAlertPrefix_Good(t *core.T) {
	core.AssertTrue(t, hasAlertPrefix("uuid-7\x00pages", "uuid-7"))
}

func TestEdge_hasAlertPrefix_Bad(t *core.T) {
	core.AssertFalse(t, hasAlertPrefix("uuid-9\x00pages", "uuid-7"))
}

func TestEdge_hasAlertPrefix_Ugly(t *core.T) {
	core.AssertFalse(t, hasAlertPrefix("u", "uuid-7"))
}

func TestEdge_BoostIsUsable_Good(t *core.T) {
	boost := Boost{BoostType: BoostTypeAddLimit, Status: BoostStatusActive, LimitValue: 10, ConsumedQuantity: 3}
	core.AssertTrue(t, boost.IsUsable())
}

func TestEdge_BoostIsUsable_Bad(t *core.T) {
	exhausted := Boost{BoostType: BoostTypeAddLimit, Status: BoostStatusActive, LimitValue: 5, ConsumedQuantity: 5}
	core.AssertFalse(t, exhausted.IsUsable())
	inactive := Boost{BoostType: BoostTypeUnlimited, Status: BoostStatusExpired}
	core.AssertFalse(t, inactive.IsUsable())
}

func TestEdge_BoostIsUsable_Ugly(t *core.T) {
	future := time.Now().Add(time.Hour)
	notStarted := Boost{BoostType: BoostTypeUnlimited, Status: BoostStatusActive, StartsAt: &future}
	core.AssertFalse(t, notStarted.IsUsable())
	past := time.Now().Add(-time.Hour)
	expired := Boost{BoostType: BoostTypeUnlimited, Status: BoostStatusActive, ExpiresAt: &past}
	core.AssertFalse(t, expired.IsUsable())
}

func TestEdge_ScopeResolveByID_Good(t *core.T) {
	tenantService, _ := testTenant(t)
	scope := NewWorkspaceScope(tenantService)
	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set("X-Workspace-ID", "7")
	rec := httptest.NewRecorder()
	scope.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if WorkspaceFromCtx(r.Context()).OK {
			w.WriteHeader(http.StatusAccepted)
		}
	})).ServeHTTP(rec, req)
	core.AssertEqual(t, http.StatusAccepted, rec.Code)
}

func TestEdge_ScopeResolveByID_Bad(t *core.T) {
	tenantService, _ := testTenant(t)
	scope := NewWorkspaceScope(tenantService)
	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set("X-Workspace-ID", "not-a-number")
	rec := httptest.NewRecorder()
	scope.Middleware()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(rec, req)
	core.AssertEqual(t, http.StatusUnauthorized, rec.Code)
}

func TestEdge_ScopeResolveByQueryParam_Ugly(t *core.T) {
	tenantService, _ := testTenant(t)
	scope := NewWorkspaceScope(tenantService)
	req := httptest.NewRequest(http.MethodGet, "/resource?workspace=acme", nil)
	req.Host = ""
	rec := httptest.NewRecorder()
	scope.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if WorkspaceFromCtx(r.Context()).OK {
			w.WriteHeader(http.StatusAccepted)
		}
	})).ServeHTTP(rec, req)
	core.AssertEqual(t, http.StatusAccepted, rec.Code)
}

func TestEdge_CanWithAddLimitBoost_Good(t *core.T) {
	// Package limit 5 + add_limit boost of 10 (remaining 10) = 15 effective.
	cache, workspace := testCache(t)
	requireResultOK(t, cache.SetBoosts(workspace.UUID, []Boost{{
		FeatureCode: "pages", BoostType: BoostTypeAddLimit, Status: BoostStatusActive, LimitValue: 10,
	}}))
	requireResultOK(t, cache.SetPackages(workspace.UUID, []Package{{
		Code: "starter", IsActive: true, Features: []PackageFeature{{FeatureCode: "pages", LimitValue: testInt(5)}},
	}}))
	requireResultOK(t, cache.SetUsage(workspace.UUID, "pages", 7))
	svc := NewLocalEntitlementService(cache, nil)
	result := svc.Can(context.Background(), workspace, "pages", 1)
	core.AssertTrue(t, result.IsAllowed())
	core.AssertEqual(t, 15, *result.Limit)
}

func TestEdge_SummaryWithAddLimitBoost_Ugly(t *core.T) {
	cache, workspace := testCache(t)
	requireResultOK(t, cache.SetBoosts(workspace.UUID, []Boost{{
		FeatureCode: "pages", BoostType: BoostTypeAddLimit, Status: BoostStatusActive, LimitValue: 20,
	}}))
	svc := NewLocalEntitlementService(cache, nil)
	result := svc.GetUsageSummary(context.Background(), workspace)
	requireResultOK(t, result)
	items := result.Value.([]UsageSummaryItem)
	core.AssertTrue(t, len(items) >= 1)
	for _, item := range items {
		if item.FeatureCode == "pages" {
			core.AssertNotNil(t, item.Limit)
			core.AssertEqual(t, 30, *item.Limit)
		}
	}
}
