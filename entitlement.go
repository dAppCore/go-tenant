// SPDX-License-Identifier: EUPL-1.2

package tenant

import "context"

// EntitlementResult is the value object returned by every entitlement check.
// It carries the decision plus usage context for display and logging.
//
//	result := svc.Can(ctx, ws, "pages", 1)
//	if result.IsDenied() { return core.E("tenant", result.Reason, nil) }
type EntitlementResult struct {
	Allowed     bool
	Reason      string // human-readable denial message; empty if allowed
	FeatureCode string
	Limit       *int // nil for boolean features
	Used        *int // nil for boolean features
	Remaining   *int // nil for boolean features
	Unlimited   bool
}

// IsAllowed reports whether the requested quantity can be consumed.
func (r EntitlementResult) IsAllowed() bool { return r.Allowed }

// IsDenied is the inverse of IsAllowed.
func (r EntitlementResult) IsDenied() bool { return !r.Allowed }

// UsagePercent returns 0-100 usage as float64. Returns nil for unlimited/boolean.
//
//	if pct := result.UsagePercent(); pct != nil && *pct >= 80 { warnNearLimit() }
func (r EntitlementResult) UsagePercent() *float64 {
	// TODO: implement
	return nil
}

// IsNearLimit reports whether usage exceeds 80% of the limit.
func (r EntitlementResult) IsNearLimit() bool {
	// TODO: implement
	return false
}

// IsAtLimit reports whether remaining capacity is zero.
func (r EntitlementResult) IsAtLimit() bool {
	// TODO: implement
	return false
}

// AsError converts to nil (allowed) or ErrEntitlementDenied (denied).
// Convenience for callers that want to treat denial as an error.
//
//	if err := svc.Can(ctx, ws, "pages", 1).AsError(); err != nil { return err }
func (r EntitlementResult) AsError() error {
	// TODO: implement
	return nil
}

// Allow constructs an allowed result with usage context.
//
//	return tenant.Allow("pages", &limit, &used)
func Allow(featureCode string, limit, used *int) EntitlementResult {
	// TODO: implement
	return EntitlementResult{Allowed: true, FeatureCode: featureCode}
}

// Deny constructs a denied result with a human-readable reason.
//
//	return tenant.Deny("pages", "Your plan does not include pages.", nil, nil)
func Deny(featureCode, reason string, limit, used *int) EntitlementResult {
	// TODO: implement
	return EntitlementResult{Allowed: false, FeatureCode: featureCode, Reason: reason}
}

// AllowUnlimited constructs an unlimited allowed result.
//
//	return tenant.AllowUnlimited("pages")
func AllowUnlimited(featureCode string) EntitlementResult {
	return EntitlementResult{Allowed: true, FeatureCode: featureCode, Unlimited: true}
}

// EntitlementService is the interface all entitlement checks go through.
// The default implementation calls TenantClient and caches results via TenantCache.
type EntitlementService interface {
	// Can checks whether the workspace can consume quantity units of featureCode.
	// quantity=1 for standard single-resource checks.
	//
	//   result := svc.Can(ctx, ws, "pages", 1)
	//   if result.IsDenied() { return core.E("pages", result.Reason, nil) }
	Can(ctx context.Context, ws *Workspace, featureCode string, quantity int) EntitlementResult

	// RecordUsage records consumption of featureCode after a successful operation.
	// Invalidates the usage cache entry for this workspace+feature.
	//
	//   svc.RecordUsage(ctx, ws, "pages", 1, &userID, map[string]any{"page_id": 42})
	RecordUsage(ctx context.Context, ws *Workspace, featureCode string, quantity int, userID *int64, metadata map[string]any) error

	// GetUsageSummary returns all tracked features with current usage for ws.
	//
	//   summary, _ := svc.GetUsageSummary(ctx, ws)
	GetUsageSummary(ctx context.Context, ws *Workspace) ([]UsageSummaryItem, error)

	// InvalidateWorkspace drops all cached entitlement data for ws.
	//
	//   svc.InvalidateWorkspace(ws.UUID)
	InvalidateWorkspace(wsUUID string)
}

// NewLocalEntitlementService creates the default implementation backed by cache and client.
// client may be nil for unit tests (cache-only mode — no API fallback on miss).
//
//	svc := tenant.NewLocalEntitlementService(cache, client)
//	svc := tenant.NewLocalEntitlementService(cache, nil)  // test mode
func NewLocalEntitlementService(cache *TenantCache, client *TenantClient) EntitlementService {
	// TODO: implement — return localEntitlementService struct
	return nil
}
