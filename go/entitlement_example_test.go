// SPDX-License-Identifier: EUPL-1.2

package tenant

import "context"

func ExampleEntitlementResult_IsAllowed() {
	Allow("pages", nil, nil).IsAllowed()
}

func ExampleEntitlementResult_IsDenied() {
	Deny("pages", "limit reached", nil, nil).IsDenied()
}

func ExampleEntitlementResult_UsagePercent() {
	limit, used := 10, 8
	Allow("pages", &limit, &used).UsagePercent()
}

func ExampleEntitlementResult_IsNearLimit() {
	limit, used := 10, 8
	Allow("pages", &limit, &used).IsNearLimit()
}

func ExampleEntitlementResult_IsAtLimit() {
	limit, used := 10, 10
	Allow("pages", &limit, &used).IsAtLimit()
}

func ExampleEntitlementResult_AsError() {
	Deny("pages", "limit reached", nil, nil).AsError()
}

func ExampleAllow() {
	limit, used := 10, 3
	Allow("pages", &limit, &used)
}

func ExampleDeny() {
	Deny("pages", "limit reached", nil, nil)
}

func ExampleAllowUnlimited() {
	AllowUnlimited("pages")
}

func ExampleNewLocalEntitlementService() {
	cache := NewTenantCache(nil)
	svc := NewLocalEntitlementService(cache, nil)
	svc.GetUsageSummary(context.Background(), &Workspace{UUID: "uuid-7"})
}

func ExampleEntitlementService_Can() {
	cache := NewTenantCache(nil)
	svc := NewLocalEntitlementService(cache, nil)
	svc.Can(context.Background(), &Workspace{UUID: "uuid-7"}, "pages", 1)
}

func ExampleEntitlementService_RecordUsage() {
	cache := NewTenantCache(nil)
	svc := NewLocalEntitlementService(cache, nil)
	svc.RecordUsage(context.Background(), &Workspace{UUID: "uuid-7"}, "pages", 1, nil, nil)
}

func ExampleEntitlementService_GetUsageSummary() {
	cache := NewTenantCache(nil)
	svc := NewLocalEntitlementService(cache, nil)
	svc.GetUsageSummary(context.Background(), &Workspace{UUID: "uuid-7"})
}

func ExampleEntitlementService_InvalidateWorkspace() {
	cache := NewTenantCache(nil)
	svc := NewLocalEntitlementService(cache, nil)
	svc.InvalidateWorkspace("uuid-7")
}
