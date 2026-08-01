// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"dappco.re/go"
)

func TestEntitlement_EntitlementResult_IsAllowed_Good(t *core.T) {
	result := Allow("pages", new(10), new(3))
	core.AssertTrue(t, result.IsAllowed())
	core.AssertFalse(t, result.IsDenied())
}

func TestEntitlement_EntitlementResult_IsAllowed_Bad(t *core.T) {
	result := Deny("pages", "limit reached", new(10), new(10))
	core.AssertFalse(t, result.IsAllowed())
	core.AssertTrue(t, result.IsDenied())
}

func TestEntitlement_EntitlementResult_IsAllowed_Ugly(t *core.T) {
	result := AllowUnlimited("pages")
	core.AssertTrue(t, result.IsAllowed())
	core.AssertTrue(t, result.Unlimited)
}

func TestEntitlement_EntitlementResult_IsDenied_Good(t *core.T) {
	result := Deny("pages", "limit reached", new(10), new(10))
	core.AssertTrue(t, result.IsDenied())
	core.AssertEqual(t, "limit reached", result.Reason)
}

func TestEntitlement_EntitlementResult_IsDenied_Bad(t *core.T) {
	result := Allow("pages", new(10), new(3))
	core.AssertFalse(t, result.IsDenied())
	core.AssertEqual(t, "pages", result.FeatureCode)
}

func TestEntitlement_EntitlementResult_IsDenied_Ugly(t *core.T) {
	result := AllowUnlimited("pages")
	core.AssertFalse(t, result.IsDenied())
	core.AssertTrue(t, result.Unlimited)
}

func TestEntitlement_EntitlementResult_UsagePercent_Good(t *core.T) {
	result := Allow("pages", new(10), new(3))
	pct := result.UsagePercent()
	core.AssertNotNil(t, pct)
	core.AssertEqual(t, float64(30), *pct)
}

func TestEntitlement_EntitlementResult_UsagePercent_Bad(t *core.T) {
	result := AllowUnlimited("pages")
	pct := result.UsagePercent()
	core.AssertNil(t, pct)
	core.AssertTrue(t, result.Unlimited)
}

func TestEntitlement_EntitlementResult_UsagePercent_Ugly(t *core.T) {
	result := Allow("pages", new(10), new(15))
	pct := result.UsagePercent()
	core.AssertNotNil(t, pct)
	core.AssertEqual(t, float64(100), *pct)
}

func TestEntitlement_EntitlementResult_IsNearLimit_Good(t *core.T) {
	result := Allow("pages", new(10), new(8))
	core.AssertTrue(t, result.IsNearLimit())
	core.AssertFalse(t, result.IsAtLimit())
}

func TestEntitlement_EntitlementResult_IsNearLimit_Bad(t *core.T) {
	result := Allow("pages", new(10), new(3))
	core.AssertFalse(t, result.IsNearLimit())
	core.AssertFalse(t, result.IsAtLimit())
}

func TestEntitlement_EntitlementResult_IsNearLimit_Ugly(t *core.T) {
	result := AllowUnlimited("pages")
	core.AssertFalse(t, result.IsNearLimit())
	core.AssertTrue(t, result.Unlimited)
}

func TestEntitlement_EntitlementResult_IsAtLimit_Good(t *core.T) {
	result := Allow("pages", new(10), new(10))
	core.AssertTrue(t, result.IsAtLimit())
	core.AssertEqual(t, 0, *result.Remaining)
}

func TestEntitlement_EntitlementResult_IsAtLimit_Bad(t *core.T) {
	result := Allow("pages", new(10), new(9))
	core.AssertFalse(t, result.IsAtLimit())
	core.AssertEqual(t, 1, *result.Remaining)
}

func TestEntitlement_EntitlementResult_IsAtLimit_Ugly(t *core.T) {
	result := AllowUnlimited("pages")
	core.AssertFalse(t, result.IsAtLimit())
	core.AssertTrue(t, result.Unlimited)
}

func TestEntitlement_EntitlementResult_AsError_Good(t *core.T) {
	result := Allow("pages", new(10), new(3)).AsError()
	requireResultOK(t, result)
	core.AssertNil(t, result.Value)
}

func TestEntitlement_EntitlementResult_AsError_Bad(t *core.T) {
	result := Deny("pages", "limit reached", new(10), new(10)).AsError()
	requireResultFail(t, result)
	core.AssertEqual(t, ErrEntitlementDenied, result.Value)
}

func TestEntitlement_EntitlementResult_AsError_Ugly(t *core.T) {
	result := AllowUnlimited("pages").AsError()
	requireResultOK(t, result)
	core.AssertNil(t, result.Value)
}

func TestEntitlement_Allow_Good(t *core.T) {
	result := Allow("pages", new(10), new(3))
	core.AssertTrue(t, result.Allowed)
	core.AssertEqual(t, 7, *result.Remaining)
}

func TestEntitlement_Allow_Bad(t *core.T) {
	result := Allow("pages", nil, nil)
	core.AssertTrue(t, result.Allowed)
	core.AssertNil(t, result.Remaining)
}

func TestEntitlement_Allow_Ugly(t *core.T) {
	result := Allow("pages", new(10), new(12))
	core.AssertTrue(t, result.Allowed)
	core.AssertEqual(t, 0, *result.Remaining)
}

func TestEntitlement_Deny_Good(t *core.T) {
	result := Deny("pages", "limit reached", new(10), new(10))
	core.AssertFalse(t, result.Allowed)
	core.AssertEqual(t, "limit reached", result.Reason)
}

func TestEntitlement_Deny_Bad(t *core.T) {
	result := Deny("pages", "missing", nil, nil)
	core.AssertFalse(t, result.Allowed)
	core.AssertNil(t, result.Remaining)
}

func TestEntitlement_Deny_Ugly(t *core.T) {
	result := Deny("pages", "over", new(10), new(12))
	core.AssertFalse(t, result.Allowed)
	core.AssertEqual(t, 0, *result.Remaining)
}

func TestEntitlement_AllowUnlimited_Good(t *core.T) {
	result := AllowUnlimited("pages")
	core.AssertTrue(t, result.Allowed)
	core.AssertTrue(t, result.Unlimited)
}

func TestEntitlement_AllowUnlimited_Bad(t *core.T) {
	result := AllowUnlimited("")
	core.AssertTrue(t, result.Allowed)
	core.AssertEqual(t, "", result.FeatureCode)
}

func TestEntitlement_AllowUnlimited_Ugly(t *core.T) {
	result := AllowUnlimited(" Pages ")
	core.AssertTrue(t, result.Unlimited)
	core.AssertEqual(t, " Pages ", result.FeatureCode)
}

func TestEntitlement_NewLocalEntitlementService_Good(t *core.T) {
	cache, _ := testCache(t)
	svc := NewLocalEntitlementService(cache, nil)
	core.AssertNotNil(t, svc)
	core.AssertTrue(t, svc.Can(t.Context(), testWorkspace(), "pages", 1).IsAllowed())
}

func TestEntitlement_NewLocalEntitlementService_Bad(t *core.T) {
	svc := NewLocalEntitlementService(nil, nil)
	result := svc.Can(t.Context(), testWorkspace(), "pages", 1)
	core.AssertTrue(t, result.IsDenied())
	core.AssertEqual(t, ErrFeatureNotFound.Error(), result.Reason)
}

func TestEntitlement_NewLocalEntitlementService_Ugly(t *core.T) {
	cache := NewTenantCache(nil)
	svc := NewLocalEntitlementService(cache, nil)
	result := svc.GetUsageSummary(t.Context(), testWorkspace())
	requireResultOK(t, result)
	core.AssertEqual(t, 0, len(result.Value.([]UsageSummaryItem)))
}

func TestEntitlement_EntitlementService_Can_Good(t *core.T) {
	cache, workspace := testCache(t)
	svc := NewLocalEntitlementService(cache, nil)
	result := svc.Can(t.Context(), workspace, "pages", 1)
	core.AssertTrue(t, result.IsAllowed())
	core.AssertEqual(t, 7, *result.Remaining)
}

func TestEntitlement_EntitlementService_Can_Bad(t *core.T) {
	cache, _ := testCache(t)
	svc := NewLocalEntitlementService(cache, nil)
	result := svc.Can(t.Context(), nil, "pages", 1)
	core.AssertTrue(t, result.IsDenied())
	core.AssertEqual(t, "no workspace provided", result.Reason)
}

func TestEntitlement_EntitlementService_Can_Ugly(t *core.T) {
	cache, workspace := testCache(t)
	svc := NewLocalEntitlementService(cache, nil)
	result := svc.Can(t.Context(), workspace, "pages", 99)
	core.AssertTrue(t, result.IsDenied())
	core.AssertEqual(t, "limit reached", result.Reason)
}

func TestEntitlement_EntitlementService_RecordUsage_Good(t *core.T) {
	cache, workspace := testCache(t)
	svc := NewLocalEntitlementService(cache, nil)
	result := svc.RecordUsage(t.Context(), workspace, "pages", 2, nil, nil)
	requireResultOK(t, result)
	used, ok := cache.GetUsage(workspace.UUID, "pages")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, 5, used)
}

func TestEntitlement_EntitlementService_RecordUsage_Bad(t *core.T) {
	cache, _ := testCache(t)
	svc := NewLocalEntitlementService(cache, nil)
	result := svc.RecordUsage(t.Context(), nil, "pages", 1, nil, nil)
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoWorkspaceContext, result.Value)
}

func TestEntitlement_EntitlementService_RecordUsage_Ugly(t *core.T) {
	cache, workspace := testCache(t)
	svc := NewLocalEntitlementService(cache, nil)
	result := svc.RecordUsage(t.Context(), workspace, "missing", 1, nil, nil)
	requireResultFail(t, result)
	core.AssertEqual(t, ErrFeatureNotFound, result.Value)
}

func TestEntitlement_EntitlementService_GetUsageSummary_Good(t *core.T) {
	cache, workspace := testCache(t)
	svc := NewLocalEntitlementService(cache, nil)
	result := svc.GetUsageSummary(t.Context(), workspace)
	requireResultOK(t, result)
	core.AssertEqual(t, "pages", result.Value.([]UsageSummaryItem)[0].FeatureCode)
}

func TestEntitlement_EntitlementService_GetUsageSummary_Bad(t *core.T) {
	cache, _ := testCache(t)
	svc := NewLocalEntitlementService(cache, nil)
	result := svc.GetUsageSummary(t.Context(), nil)
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoWorkspaceContext, result.Value)
}

func TestEntitlement_EntitlementService_GetUsageSummary_Ugly(t *core.T) {
	svc := NewLocalEntitlementService(NewTenantCache(nil), nil)
	result := svc.GetUsageSummary(t.Context(), testWorkspace())
	requireResultOK(t, result)
	core.AssertEqual(t, 0, len(result.Value.([]UsageSummaryItem)))
}

func TestEntitlement_EntitlementService_InvalidateWorkspace_Good(t *core.T) {
	cache, workspace := testCache(t)
	svc := NewLocalEntitlementService(cache, nil)
	svc.InvalidateWorkspace(workspace.UUID)
	got, ok := cache.GetWorkspace(workspace.UUID)
	core.AssertFalse(t, ok)
	core.AssertNil(t, got)
}

func TestEntitlement_EntitlementService_InvalidateWorkspace_Bad(t *core.T) {
	svc := NewLocalEntitlementService(nil, nil)
	svc.InvalidateWorkspace("missing")
	core.AssertNotNil(t, svc)
}

func TestEntitlement_EntitlementService_InvalidateWorkspace_Ugly(t *core.T) {
	cache, workspace := testCache(t)
	requireResultOK(t, cache.SetBoosts(workspace.UUID, []Boost{{FeatureCode: "pages"}}))
	svc := NewLocalEntitlementService(cache, nil)
	svc.InvalidateWorkspace(workspace.UUID)
	_, ok := cache.GetBoosts(workspace.UUID)
	core.AssertFalse(t, ok)
}
