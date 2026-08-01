// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"context"
	"net/http"
	"net/http/httptest"

	"dappco.re/go"
)

// apiResponses maps request paths to JSON bodies for a routing test server.
// Paths default to 404 (workspace/feature not found) when unmapped.
type apiResponses map[string]string

// testAPIServer returns a server that routes by URL path. A nil body for a
// known path returns 200 with the body; a missing path returns 404.
func testAPIServer(routes apiResponses) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if body, ok := routes[r.URL.Path]; ok {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(body)); err != nil {
				touchError(err)
			}
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
}

// clientBackedService wires a cache-less entitlement service to a test server,
// forcing every load* call through the client fallback.
func clientBackedService(t *core.T, routes apiResponses) (EntitlementService, *TenantCache, *Workspace) {
	t.Helper()
	server := testAPIServer(routes)
	t.Cleanup(server.Close)
	client := NewTenantClient(server.URL, "token")
	cache := NewTenantCache(nil)
	return NewLocalEntitlementService(cache, client), cache, testWorkspace()
}

func TestEntitlementClient_Can_LoadsFromClient_Good(t *core.T) {
	svc, _, ws := clientBackedService(t, apiResponses{
		"/api/v1/features/pages":                `{"code":"pages","type":"limit","is_active":true}`,
		"/api/v1/workspaces/uuid-7/packages":    `{"data":[{"code":"starter","is_active":true,"features":[{"feature_code":"pages","limit_value":10}]}]}`,
		"/api/v1/workspaces/uuid-7/boosts":      `{"data":[]}`,
		"/api/v1/workspaces/uuid-7/usage/pages": `{"ok":true,"count":3}`,
	})
	result := svc.Can(context.Background(), ws, "pages", 1)
	core.AssertTrue(t, result.IsAllowed())
	core.AssertEqual(t, 7, *result.Remaining)
}

func TestEntitlementClient_Can_FeatureNotFound_Bad(t *core.T) {
	svc, _, ws := clientBackedService(t, apiResponses{})
	result := svc.Can(context.Background(), ws, "pages", 1)
	core.AssertTrue(t, result.IsDenied())
	core.AssertContains(t, result.Reason, "not found")
}

func TestEntitlementClient_Can_LimitReached_Ugly(t *core.T) {
	svc, _, ws := clientBackedService(t, apiResponses{
		"/api/v1/features/pages":                `{"code":"pages","type":"limit","is_active":true}`,
		"/api/v1/workspaces/uuid-7/packages":    `{"data":[{"code":"starter","is_active":true,"features":[{"feature_code":"pages","limit_value":5}]}]}`,
		"/api/v1/workspaces/uuid-7/boosts":      `{"data":[]}`,
		"/api/v1/workspaces/uuid-7/usage/pages": `{"ok":true,"count":5}`,
	})
	result := svc.Can(context.Background(), ws, "pages", 1)
	core.AssertTrue(t, result.IsDenied())
	core.AssertEqual(t, "limit reached", result.Reason)
}

func TestEntitlementClient_Can_UnlimitedPackage_Good(t *core.T) {
	svc, _, ws := clientBackedService(t, apiResponses{
		"/api/v1/features/pages":             `{"code":"pages","type":"limit","is_active":true}`,
		"/api/v1/workspaces/uuid-7/packages": `{"data":[{"code":"pro","is_active":true,"features":[{"feature_code":"pages","limit_value":-1}]}]}`,
		"/api/v1/workspaces/uuid-7/boosts":   `{"data":[]}`,
	})
	result := svc.Can(context.Background(), ws, "pages", 100)
	core.AssertTrue(t, result.IsAllowed())
	core.AssertTrue(t, result.Unlimited)
}

func TestEntitlementClient_GetUsageSummary_FromClient_Good(t *core.T) {
	svc, _, ws := clientBackedService(t, apiResponses{
		"/api/v1/features/pages":                `{"code":"pages","name":"Pages","type":"limit","is_active":true}`,
		"/api/v1/workspaces/uuid-7/packages":    `{"data":[{"code":"starter","is_active":true,"features":[{"feature_code":"pages","limit_value":10}]}]}`,
		"/api/v1/workspaces/uuid-7/boosts":      `{"data":[]}`,
		"/api/v1/workspaces/uuid-7/usage/pages": `{"ok":true,"count":4}`,
	})
	result := svc.GetUsageSummary(context.Background(), ws)
	requireResultOK(t, result)
	items := result.Value.([]UsageSummaryItem)
	core.AssertEqual(t, 1, len(items))
	core.AssertEqual(t, "pages", items[0].FeatureCode)
	core.AssertEqual(t, 10, *items[0].Limit)
	core.AssertEqual(t, 4, *items[0].Used)
	core.AssertEqual(t, 6, *items[0].Remaining)
}

func TestEntitlementClient_GetUsageSummary_UnlimitedFeature_Ugly(t *core.T) {
	svc, _, ws := clientBackedService(t, apiResponses{
		"/api/v1/features/pages":                `{"code":"pages","name":"Pages","type":"limit","is_active":true}`,
		"/api/v1/workspaces/uuid-7/packages":    `{"data":[{"code":"pro","is_active":true,"features":[{"feature_code":"pages","limit_value":-1}]}]}`,
		"/api/v1/workspaces/uuid-7/boosts":      `{"data":[]}`,
		"/api/v1/workspaces/uuid-7/usage/pages": `{"ok":true,"count":12}`,
	})
	result := svc.GetUsageSummary(context.Background(), ws)
	requireResultOK(t, result)
	items := result.Value.([]UsageSummaryItem)
	core.AssertEqual(t, 1, len(items))
	core.AssertTrue(t, items[0].Unlimited)
	core.AssertNil(t, items[0].Limit)
	core.AssertEqual(t, 12, *items[0].Used)
}

func TestEntitlementClient_RecordUsage_InvalidatesCache_Good(t *core.T) {
	svc, cache, ws := clientBackedService(t, apiResponses{
		"/api/v1/features/pages":          `{"code":"pages","type":"limit","is_active":true}`,
		"/api/v1/workspaces/uuid-7/usage": `{"ok":true}`,
	})
	requireResultOK(t, cache.SetUsage(ws.UUID, "pages", 3))
	result := svc.RecordUsage(context.Background(), ws, "pages", 2, testInt64(9), map[string]any{"page_id": 42})
	requireResultOK(t, result)
	// Client-backed RecordUsage invalidates the cache rather than incrementing it.
	_, ok := cache.GetUsage(ws.UUID, "pages")
	core.AssertFalse(t, ok)
}

func TestEntitlementClient_RecordUsage_ClientError_Bad(t *core.T) {
	svc, _, ws := clientBackedService(t, apiResponses{
		"/api/v1/features/pages": `{"code":"pages","type":"limit","is_active":true}`,
		// usage POST path unmapped -> 404 -> workspace not found
	})
	result := svc.RecordUsage(context.Background(), ws, "pages", 1, nil, nil)
	requireResultFail(t, result)
	core.AssertEqual(t, ErrWorkspaceNotFound, result.Value)
}

func TestEntitlementClient_BooleanFeatureEnableBoost_Ugly(t *core.T) {
	svc, _, ws := clientBackedService(t, apiResponses{
		"/api/v1/features/api_access":        `{"code":"api_access","type":"boolean","is_active":true}`,
		"/api/v1/workspaces/uuid-7/packages": `{"data":[]}`,
		"/api/v1/workspaces/uuid-7/boosts":   `{"data":[{"feature_code":"api_access","boost_type":"enable","status":"active"}]}`,
	})
	result := svc.Can(context.Background(), ws, "api_access", 1)
	core.AssertTrue(t, result.IsAllowed())
}
