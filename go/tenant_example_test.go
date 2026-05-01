// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"context"

	"dappco.re/go"
)

func ExampleRegister() {
	c := core.New()
	Register(c)
}

func ExampleTenant_GetWorkspace() {
	cache := NewTenantCache(nil)
	cache.SetWorkspace(&Workspace{UUID: "uuid-7", Slug: "acme"})
	tenantService := &Tenant{cache: cache}
	tenantService.GetWorkspace(context.Background(), "acme")
}

func ExampleTenant_GetWorkspaceByUUID() {
	cache := NewTenantCache(nil)
	cache.SetWorkspace(&Workspace{UUID: "uuid-7"})
	tenantService := &Tenant{cache: cache}
	tenantService.GetWorkspaceByUUID(context.Background(), "uuid-7")
}

func ExampleTenant_GetWorkspaceByID() {
	cache := NewTenantCache(nil)
	cache.SetWorkspace(&Workspace{ID: 7, UUID: "uuid-7"})
	tenantService := &Tenant{cache: cache}
	tenantService.GetWorkspaceByID(context.Background(), 7)
}

func ExampleTenant_GetUser() {
	tenantService := &Tenant{cache: NewTenantCache(nil)}
	tenantService.GetUser(WithUser(context.Background(), &User{UUID: "user-9"}))
}

func ExampleTenant_GetWorkspaceBySubdomain() {
	cache := NewTenantCache(nil)
	cache.SetWorkspace(&Workspace{UUID: "uuid-7", Slug: "acme"})
	tenantService := &Tenant{cache: cache}
	tenantService.GetWorkspaceBySubdomain(context.Background(), "acme.host.uk.com")
}

func ExampleTenant_Can() {
	tenantService := &Tenant{cache: NewTenantCache(nil)}
	tenantService.Can(context.Background(), &Workspace{UUID: "uuid-7"}, "pages", 1)
}

func ExampleTenant_RecordUsage() {
	cache := NewTenantCache(nil)
	tenantService := &Tenant{cache: cache, entitlements: NewLocalEntitlementService(cache, nil)}
	tenantService.RecordUsage(context.Background(), &Workspace{UUID: "uuid-7"}, "pages", 1, nil, nil)
}

func ExampleTenant_GetUsageSummary() {
	cache := NewTenantCache(nil)
	tenantService := &Tenant{cache: cache, entitlements: NewLocalEntitlementService(cache, nil)}
	tenantService.GetUsageSummary(context.Background(), &Workspace{UUID: "uuid-7"})
}

func ExampleTenant_InvalidateWorkspace() {
	tenantService := &Tenant{cache: NewTenantCache(nil)}
	tenantService.InvalidateWorkspace("uuid-7")
}

func ExampleTenant_OnUsageAlert() {
	tenantService := &Tenant{}
	tenantService.OnUsageAlert(func(UsageAlert) {})
}

func ExampleTenant_Scope() {
	tenantService := &Tenant{}
	tenantService.Scope()
}

func ExampleTenant_CheckUsageAlerts() {
	tenantService := &Tenant{}
	tenantService.CheckUsageAlerts(&Workspace{UUID: "uuid-7"}, "pages", Allow("pages", nil, nil))
}
