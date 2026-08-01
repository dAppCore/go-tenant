// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"context"
	"time"

	"dappco.re/go"
)

// clientBackedTenant wires a Tenant with an empty cache to a routing test
// server, forcing the client fallback on every resolver miss.
func clientBackedTenant(t *core.T, routes apiResponses) *Tenant {
	t.Helper()
	server := testAPIServer(routes)
	t.Cleanup(server.Close)
	cache := NewTenantCache(nil)
	client := NewTenantClient(server.URL, "token")
	return &Tenant{
		cache:        cache,
		client:       client,
		entitlements: NewLocalEntitlementService(cache, client),
		alertState:   map[string]usageAlertTracker{},
	}
}

func TestTenantClient_GetWorkspace_FromClient_Good(t *core.T) {
	ten := clientBackedTenant(t, apiResponses{
		"/api/v1/workspaces/acme": `{"id":7,"uuid":"uuid-7","slug":"acme"}`,
	})
	result := ten.GetWorkspace(context.Background(), "acme")
	requireResultOK(t, result)
	core.AssertEqual(t, "uuid-7", result.Value.(*Workspace).UUID)
	// Second call must hit the now-populated cache, not the client.
	cached, ok := ten.cache.GetWorkspaceBySlug("acme")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "uuid-7", cached.UUID)
}

func TestTenantClient_GetWorkspace_NotFound_Bad(t *core.T) {
	ten := clientBackedTenant(t, apiResponses{})
	result := ten.GetWorkspace(context.Background(), "missing")
	requireResultFail(t, result)
	core.AssertEqual(t, ErrWorkspaceNotFound, result.Value)
}

func TestTenantClient_GetWorkspaceByUUID_FromClient_Good(t *core.T) {
	ten := clientBackedTenant(t, apiResponses{
		"/api/v1/workspaces/uuid/uuid-7": `{"id":7,"uuid":"uuid-7","slug":"acme"}`,
	})
	result := ten.GetWorkspaceByUUID(context.Background(), "uuid-7")
	requireResultOK(t, result)
	core.AssertEqual(t, "acme", result.Value.(*Workspace).Slug)
}

func TestTenantClient_GetWorkspaceByUUID_NotFound_Bad(t *core.T) {
	ten := clientBackedTenant(t, apiResponses{})
	result := ten.GetWorkspaceByUUID(context.Background(), "ghost")
	requireResultFail(t, result)
	core.AssertEqual(t, ErrWorkspaceNotFound, result.Value)
}

func TestTenantClient_GetWorkspaceByID_FromClient_Good(t *core.T) {
	ten := clientBackedTenant(t, apiResponses{
		"/api/v1/workspaces/id/7": `{"id":7,"uuid":"uuid-7","slug":"acme"}`,
	})
	result := ten.GetWorkspaceByID(context.Background(), 7)
	requireResultOK(t, result)
	core.AssertEqual(t, "uuid-7", result.Value.(*Workspace).UUID)
}

func TestTenantClient_GetWorkspaceByID_NotFound_Bad(t *core.T) {
	ten := clientBackedTenant(t, apiResponses{})
	result := ten.GetWorkspaceByID(context.Background(), 404)
	requireResultFail(t, result)
	core.AssertEqual(t, ErrWorkspaceNotFound, result.Value)
}

func TestTenantClient_GetWorkspaceBySubdomain_FromClient_Good(t *core.T) {
	// Slug lookup misses (404), so it falls back to the subdomain endpoint.
	ten := clientBackedTenant(t, apiResponses{
		"/api/v1/workspaces/subdomain/acme.host.uk.com": `{"id":7,"uuid":"uuid-7","slug":"acme"}`,
	})
	result := ten.GetWorkspaceBySubdomain(context.Background(), "acme.host.uk.com")
	requireResultOK(t, result)
	core.AssertEqual(t, "uuid-7", result.Value.(*Workspace).UUID)
}

func TestTenantClient_GetWorkspaceBySubdomain_SlugHit_Ugly(t *core.T) {
	// The slug derived from the host resolves directly; the subdomain
	// endpoint is never consulted.
	ten := clientBackedTenant(t, apiResponses{
		"/api/v1/workspaces/acme": `{"id":7,"uuid":"uuid-7","slug":"acme"}`,
	})
	result := ten.GetWorkspaceBySubdomain(context.Background(), "acme.host.uk.com")
	requireResultOK(t, result)
	core.AssertEqual(t, "acme", result.Value.(*Workspace).Slug)
}

func TestTenantClient_GetWorkspaceBySubdomain_NotFound_Bad(t *core.T) {
	ten := clientBackedTenant(t, apiResponses{})
	result := ten.GetWorkspaceBySubdomain(context.Background(), "ghost.host.uk.com")
	requireResultFail(t, result)
	core.AssertEqual(t, ErrWorkspaceNotFound, result.Value)
}

func TestTenantClient_GetUser_FromClient_Good(t *core.T) {
	// No user in ctx, so the client /api/v1/user endpoint is queried.
	ten := clientBackedTenant(t, apiResponses{
		"/api/v1/user": `{"uuid":"user-9","email":"ada@example.uk"}`,
	})
	result := ten.GetUser(context.Background())
	requireResultOK(t, result)
	core.AssertEqual(t, "user-9", result.Value.(*User).UUID)
}

func TestTenantClient_GetUser_FromContextCachesOnce_Ugly(t *core.T) {
	ten := clientBackedTenant(t, apiResponses{})
	ctx := WithUser(context.Background(), testUser())
	first := ten.GetUser(ctx)
	requireResultOK(t, first)
	// Second resolution returns the cached copy without re-storing.
	second := ten.GetUser(ctx)
	requireResultOK(t, second)
	core.AssertEqual(t, "user-9", second.Value.(*User).UUID)
}

func TestTenantClient_GetUser_NotFound_Bad(t *core.T) {
	// With no user in ctx the client is queried; a 404 on /api/v1/user maps
	// to ErrWorkspaceNotFound (statusError only treats /features/ paths as
	// feature-not-found, everything else as workspace-not-found).
	ten := clientBackedTenant(t, apiResponses{})
	result := ten.GetUser(context.Background())
	requireResultFail(t, result)
	core.AssertEqual(t, ErrWorkspaceNotFound, result.Value)
}

func TestTenantClient_GetUser_NoClient_Bad(t *core.T) {
	// No ctx user and no client at all returns ErrNoUserContext.
	ten := &Tenant{cache: NewTenantCache(nil)}
	result := ten.GetUser(context.Background())
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoUserContext, result.Value)
}

func TestTenantClient_RecordUsage_FiresAlert_Good(t *core.T) {
	ten := clientBackedTenant(t, apiResponses{
		"/api/v1/features/pages":                `{"code":"pages","type":"limit","is_active":true}`,
		"/api/v1/workspaces/uuid-7/packages":    `{"data":[{"code":"starter","is_active":true,"features":[{"feature_code":"pages","limit_value":10}]}]}`,
		"/api/v1/workspaces/uuid-7/boosts":      `{"data":[]}`,
		"/api/v1/workspaces/uuid-7/usage/pages": `{"ok":true,"count":9}`,
		"/api/v1/workspaces/uuid-7/usage":       `{"ok":true}`,
	})
	alerts := 0
	ten.OnUsageAlert(func(UsageAlert) { alerts++ })
	result := ten.RecordUsage(context.Background(), testWorkspace(), "pages", 1, nil, nil)
	requireResultOK(t, result)
	core.AssertTrue(t, alerts > 0)
}

func TestTenantClient_parseInt64_Good(t *core.T) {
	result := parseInt64("42")
	requireResultOK(t, result)
	core.AssertEqual(t, int64(42), result.Value.(int64))
}

func TestTenantClient_parseInt64_Bad(t *core.T) {
	result := parseInt64("bogus")
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoWorkspaceContext, result.Value)
}

func TestTenantClient_parseInt64_Ugly(t *core.T) {
	negative := parseInt64("-7")
	requireResultFail(t, negative)
	empty := parseInt64("")
	requireResultOK(t, empty)
	core.AssertEqual(t, int64(0), empty.Value.(int64))
}

func TestTenantClient_coreConfigStringValue_Good(t *core.T) {
	c := core.New()
	c.Config().Set("tenant.api_url", "https://api.host.uk.com")
	core.AssertEqual(t, "https://api.host.uk.com", coreConfigStringValue(c, "api_url", "tenant.api_url"))
}

func TestTenantClient_coreConfigStringValue_Bad(t *core.T) {
	var c *core.Core
	core.AssertEqual(t, "", coreConfigStringValue(c, "api_url"))
}

func TestTenantClient_coreConfigStringValue_Ugly(t *core.T) {
	c := core.New()
	core.AssertEqual(t, "", coreConfigStringValue(c, "api_url", "tenant.api_url"))
}

func TestTenantClient_coreConfigDurationValue_Good(t *core.T) {
	c := core.New()
	c.Config().Set("timeout", 5*time.Second)
	core.AssertEqual(t, 5*time.Second, coreConfigDurationValue(c, "timeout"))
}

func TestTenantClient_coreConfigDurationValue_Bad(t *core.T) {
	var c *core.Core
	core.AssertEqual(t, time.Duration(0), coreConfigDurationValue(c, "timeout"))
}

func TestTenantClient_coreConfigDurationValue_Ugly(t *core.T) {
	c := core.New()
	c.Config().Set("string_timeout", "3s")
	core.AssertEqual(t, 3*time.Second, coreConfigDurationValue(c, "string_timeout"))
	c.Config().Set("int_timeout", 4)
	core.AssertEqual(t, 4*time.Second, coreConfigDurationValue(c, "int_timeout"))
	c.Config().Set("int64_timeout", int64(6))
	core.AssertEqual(t, 6*time.Second, coreConfigDurationValue(c, "int64_timeout"))
	c.Config().Set("float_timeout", float64(7))
	core.AssertEqual(t, 7*time.Second, coreConfigDurationValue(c, "float_timeout"))
}

func TestTenantClient_tenantOptionsFromCoreConfig_Good(t *core.T) {
	c := core.New()
	c.Config().Set("tenant.api_url", "https://api.host.uk.com")
	c.Config().Set("tenant.api_token", "bearer")
	c.Config().Set("tenant.timeout", 5*time.Second)
	opts := tenantOptionsFromCoreConfig(c)
	core.AssertEqual(t, "https://api.host.uk.com", opts.APIURL)
	core.AssertEqual(t, "bearer", opts.APIToken)
	core.AssertEqual(t, 5*time.Second, opts.Timeout)
}

func TestTenantClient_tenantOptionsFromCoreConfig_Bad(t *core.T) {
	var c *core.Core
	opts := tenantOptionsFromCoreConfig(c)
	core.AssertEqual(t, 10*time.Second, opts.Timeout)
	core.AssertEqual(t, "", opts.APIURL)
}

func TestTenantClient_tenantOptionsFromCoreConfig_Ugly(t *core.T) {
	c := core.New()
	opts := tenantOptionsFromCoreConfig(c)
	core.AssertEqual(t, 10*time.Second, opts.Timeout)
	core.AssertEqual(t, "", opts.APIToken)
}
