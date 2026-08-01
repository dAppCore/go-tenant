// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"context"
	"sort"

	"dappco.re/go"
)

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
	if r.Unlimited || r.Limit == nil || r.Used == nil || *r.Limit <= 0 {
		return nil
	}
	pct := float64(*r.Used) / float64(*r.Limit) * 100
	if pct > 100 {
		pct = 100
	}
	return &pct
}

// IsNearLimit reports whether usage exceeds 80% of the limit.
func (r EntitlementResult) IsNearLimit() bool {
	if pct := r.UsagePercent(); pct != nil {
		return *pct >= 80
	}
	return false
}

// IsAtLimit reports whether remaining capacity is zero.
func (r EntitlementResult) IsAtLimit() bool {
	if r.Unlimited || r.Limit == nil || r.Used == nil {
		return false
	}
	return *r.Used >= *r.Limit
}

// AsError converts to nil (allowed) or ErrEntitlementDenied (denied).
// Convenience for callers that want to treat denial as an error.
//
//	if r := svc.Can(ctx, ws, "pages", 1).AsError(); !r.OK { return r }
func (r EntitlementResult) AsError() core.Result {
	if r.Allowed {
		return core.Ok(nil)
	}
	return core.Fail(ErrEntitlementDenied)
}

// Allow constructs an allowed result with usage context.
//
//	return tenant.Allow("pages", &limit, &used)
func Allow(featureCode string, limit, used *int) EntitlementResult {
	result := EntitlementResult{Allowed: true, FeatureCode: featureCode, Limit: limit, Used: used}
	if limit != nil && used != nil {
		remaining := max(*limit-*used, 0)
		result.Remaining = &remaining
	}
	return result
}

// Deny constructs a denied result with a human-readable reason.
//
//	return tenant.Deny("pages", "Your plan does not include pages.", nil, nil)
func Deny(featureCode, reason string, limit, used *int) EntitlementResult {
	result := EntitlementResult{Allowed: false, FeatureCode: featureCode, Reason: reason, Limit: limit, Used: used}
	if limit != nil && used != nil {
		remaining := max(*limit-*used, 0)
		result.Remaining = &remaining
	}
	return result
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
	RecordUsage(ctx context.Context, ws *Workspace, featureCode string, quantity int, userID *int64, metadata map[string]any) core.Result

	// GetUsageSummary returns all tracked features with current usage for ws.
	//
	//   summary, _ := svc.GetUsageSummary(ctx, ws)
	GetUsageSummary(ctx context.Context, ws *Workspace) core.Result

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
	return &localEntitlementService{cache: cache, client: client}
}

type localEntitlementService struct {
	cache  *TenantCache
	client *TenantClient
}

// featureLimitSnapshot holds the aggregated limit for a feature across all packages.
//
//	snapshot := packageLimitForFeature(packages, "pages")
//	if snapshot.Unlimited { return AllowUnlimited("pages") }
//	if !snapshot.HasFeatureAssignment { return Deny("pages", "not in package", nil, nil) }
type featureLimitSnapshot struct {
	Limit                int
	HasFeatureAssignment bool
	Unlimited            bool
}

func (entitlementService *localEntitlementService) Can(ctx context.Context, ws *Workspace, featureCode string, quantity int) EntitlementResult {
	featureCode = normalizedFeatureCode(featureCode)
	if ws == nil {
		return Deny(featureCode, "no workspace provided", nil, nil)
	}
	if quantity < 0 {
		quantity = 1
	}
	featureResult := entitlementService.loadFeature(ctx, featureCode)
	if !featureResult.OK {
		return Deny(featureCode, featureResult.Error(), nil, nil)
	}
	feature := featureResult.Value.(*Feature)
	poolCode := normalizedFeatureCode(feature.PoolCode())
	packages, _ := entitlementService.loadPackages(ctx, ws.UUID)
	boosts, _ := entitlementService.loadBoosts(ctx, ws.UUID)
	used, _ := entitlementService.loadUsage(ctx, ws.UUID, poolCode)

	packageLimit := packageLimitForFeature(packages, poolCode)
	if packageLimit.Unlimited {
		return AllowUnlimited(featureCode)
	}

	hasBooleanEnableBoost := false
	for _, boost := range boosts {
		boostCode := normalizedFeatureCode(boost.FeatureCode)
		if boostCode != poolCode && boostCode != normalizedFeatureCode(feature.Code) {
			continue
		}
		if !boost.IsUsable() {
			continue
		}
		switch boost.BoostType {
		case BoostTypeUnlimited:
			return AllowUnlimited(featureCode)
		case BoostTypeEnable:
			hasBooleanEnableBoost = true
		default:
			if !packageLimit.HasFeatureAssignment {
				continue
			}
			remaining := boost.Remaining()
			if remaining > 0 {
				packageLimit.Limit += remaining
			}
		}
	}

	if feature.IsUnlimited() {
		if packageLimit.HasFeatureAssignment {
			return AllowUnlimited(featureCode)
		}
		return Deny(featureCode, "feature not in any package", nil, nil)
	}

	if feature.IsBoolean() {
		if packageLimit.HasFeatureAssignment || hasBooleanEnableBoost {
			return Allow(featureCode, nil, nil)
		}
		return Deny(featureCode, "feature not in any package", nil, nil)
	}

	if !packageLimit.HasFeatureAssignment {
		return Deny(featureCode, "feature not in any package", nil, nil)
	}

	if used+quantity > packageLimit.Limit {
		return Deny(featureCode, "limit reached", &packageLimit.Limit, &used)
	}
	return Allow(featureCode, &packageLimit.Limit, &used)
}

func (entitlementService *localEntitlementService) RecordUsage(ctx context.Context, ws *Workspace, featureCode string, quantity int, userID *int64, metadata map[string]any) core.Result {
	featureCode = normalizedFeatureCode(featureCode)
	if ws == nil {
		return core.Fail(ErrNoWorkspaceContext)
	}
	if quantity <= 0 {
		quantity = 1
	}
	featureResult := entitlementService.loadFeature(ctx, featureCode)
	if !featureResult.OK {
		return featureResult
	}
	feature := featureResult.Value.(*Feature)
	poolCode := normalizedFeatureCode(feature.PoolCode())
	if entitlementService.client != nil {
		if r := entitlementService.client.RecordUsage(ctx, ws.UUID, featureCode, quantity, userID, metadata); !r.OK {
			return r
		}
	}
	if entitlementService.cache != nil {
		if entitlementService.client != nil {
			if r := entitlementService.cache.invalidateUsage(ws.UUID, poolCode); !r.OK {
				return r
			}
		} else if used, ok := entitlementService.cache.GetUsage(ws.UUID, poolCode); ok {
			if r := entitlementService.cache.SetUsage(ws.UUID, poolCode, used+quantity); !r.OK {
				return r
			}
		} else {
			if r := entitlementService.cache.SetUsage(ws.UUID, poolCode, quantity); !r.OK {
				return r
			}
		}
	}
	return core.Ok(nil)
}

func (entitlementService *localEntitlementService) GetUsageSummary(ctx context.Context, ws *Workspace) core.Result {
	if ws == nil {
		return core.Fail(ErrNoWorkspaceContext)
	}
	codes := map[string]struct{}{}
	packages, _ := entitlementService.loadPackages(ctx, ws.UUID)
	for _, pkg := range packages {
		if !pkg.IsActive {
			continue
		}
		for _, feature := range pkg.Features {
			codes[normalizedFeatureCode(feature.FeatureCode)] = struct{}{}
		}
	}
	boosts, _ := entitlementService.loadBoosts(ctx, ws.UUID)
	for _, boost := range boosts {
		codes[normalizedFeatureCode(boost.FeatureCode)] = struct{}{}
	}

	items := make([]UsageSummaryItem, 0, len(codes))
	for code := range codes {
		feature, _ := core.Cast[*Feature](entitlementService.loadFeature(ctx, code))
		if feature == nil {
			feature = &Feature{Code: code, Name: code}
		}
		limit, unlimited, used := entitlementService.summaryForFeature(ctx, ws.UUID, feature)
		item := UsageSummaryItem{
			FeatureCode: code,
			FeatureName: feature.Name,
			Limit:       limit,
			Used:        used,
			Unlimited:   unlimited,
			ResetType:   feature.ResetType,
		}
		if limit != nil && used != nil {
			remaining := max(*limit-*used, 0)
			item.Remaining = &remaining
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].FeatureCode < items[j].FeatureCode
	})
	return core.Ok(items)
}

func (entitlementService *localEntitlementService) InvalidateWorkspace(wsUUID string) {
	if entitlementService.cache != nil {
		if r := entitlementService.cache.InvalidateWorkspace(wsUUID); !r.OK {
			return
		}
	}
}

func (entitlementService *localEntitlementService) loadFeature(ctx context.Context, code string) core.Result {
	code = normalizedFeatureCode(code)
	if entitlementService.cache != nil {
		if feature, ok := entitlementService.cache.GetFeature(code); ok {
			return core.Ok(feature)
		}
	}
	if entitlementService.client == nil {
		return core.Fail(ErrFeatureNotFound)
	}
	r := entitlementService.client.GetFeature(ctx, code)
	if !r.OK {
		return r
	}
	feature := r.Value.(*Feature)
	if entitlementService.cache != nil {
		if cached := entitlementService.cache.SetFeature(feature); !cached.OK {
			return cached
		}
	}
	return core.Ok(feature)
}

func (entitlementService *localEntitlementService) loadPackages(ctx context.Context, wsUUID string) ([]Package, bool) {
	if entitlementService.cache != nil {
		if packages, ok := entitlementService.cache.GetPackages(wsUUID); ok {
			return packages, true
		}
	}
	if entitlementService.client == nil {
		return nil, false
	}
	r := entitlementService.client.GetPackagesForWorkspace(ctx, wsUUID)
	if !r.OK {
		return nil, false
	}
	packages := r.Value.([]Package)
	if entitlementService.cache != nil {
		if cached := entitlementService.cache.SetPackages(wsUUID, packages); !cached.OK {
			return nil, false
		}
	}
	return packages, true
}

func (entitlementService *localEntitlementService) loadBoosts(ctx context.Context, wsUUID string) ([]Boost, bool) {
	if entitlementService.cache != nil {
		if boosts, ok := entitlementService.cache.GetBoosts(wsUUID); ok {
			return boosts, true
		}
	}
	if entitlementService.client == nil {
		return nil, false
	}
	r := entitlementService.client.GetBoostsForWorkspace(ctx, wsUUID)
	if !r.OK {
		return nil, false
	}
	boosts := r.Value.([]Boost)
	if entitlementService.cache != nil {
		if cached := entitlementService.cache.SetBoosts(wsUUID, boosts); !cached.OK {
			return nil, false
		}
	}
	return boosts, true
}

func (entitlementService *localEntitlementService) loadUsage(ctx context.Context, wsUUID, featureCode string) (int, bool) {
	featureCode = normalizedFeatureCode(featureCode)
	if entitlementService.cache != nil {
		if used, ok := entitlementService.cache.GetUsage(wsUUID, featureCode); ok {
			return used, true
		}
	}
	if entitlementService.client == nil {
		return 0, false
	}
	r := entitlementService.client.GetCurrentUsage(ctx, wsUUID, featureCode)
	if !r.OK {
		return 0, false
	}
	used := r.Value.(int)
	if entitlementService.cache != nil {
		if cached := entitlementService.cache.SetUsage(wsUUID, featureCode, used); !cached.OK {
			return 0, false
		}
	}
	return used, true
}

func (entitlementService *localEntitlementService) summaryForFeature(ctx context.Context, wsUUID string, feature *Feature) (*int, bool, *int) {
	poolCode := normalizedFeatureCode(feature.PoolCode())
	packages, _ := entitlementService.loadPackages(ctx, wsUUID)
	boosts, _ := entitlementService.loadBoosts(ctx, wsUUID)

	packageLimit := packageLimitForFeature(packages, poolCode)
	if packageLimit.Unlimited {
		return nil, true, entitlementService.loadUsageCount(ctx, wsUUID, poolCode)
	}

	hasBooleanEnableBoost := false
	for _, boost := range boosts {
		boostCode := normalizedFeatureCode(boost.FeatureCode)
		if boostCode != poolCode && boostCode != normalizedFeatureCode(feature.Code) {
			continue
		}
		if !boost.IsUsable() {
			continue
		}
		switch boost.BoostType {
		case BoostTypeUnlimited:
			return nil, true, entitlementService.loadUsageCount(ctx, wsUUID, poolCode)
		case BoostTypeEnable:
			hasBooleanEnableBoost = true
		case BoostTypeAddLimit:
			if !packageLimit.HasFeatureAssignment {
				continue
			}
			remaining := boost.Remaining()
			if remaining > 0 {
				packageLimit.Limit += remaining
			}
		}
	}

	if feature.IsUnlimited() {
		if packageLimit.HasFeatureAssignment {
			return nil, true, entitlementService.loadUsageCount(ctx, wsUUID, poolCode)
		}
		return nil, false, entitlementService.loadUsageCount(ctx, wsUUID, poolCode)
	}
	if feature.IsBoolean() {
		if packageLimit.HasFeatureAssignment || hasBooleanEnableBoost {
			return nil, false, nil
		}
		return nil, false, nil
	}
	if !packageLimit.HasFeatureAssignment {
		return nil, false, entitlementService.loadUsageCount(ctx, wsUUID, poolCode)
	}
	used, _ := entitlementService.loadUsage(ctx, wsUUID, poolCode)
	return &packageLimit.Limit, false, &used
}

func (entitlementService *localEntitlementService) loadUsageCount(ctx context.Context, wsUUID, featureCode string) *int {
	used, _ := entitlementService.loadUsage(ctx, wsUUID, featureCode)
	return &used
}

// packageLimitForFeature aggregates the effective limit from all packages for a feature.
// Non-stackable packages use the highest limit; stackable packages are summed on top.
//
//	snapshot := packageLimitForFeature(packages, "pages")
//	if snapshot.Unlimited { return AllowUnlimited(code) }
func packageLimitForFeature(packages []Package, featureCode string) featureLimitSnapshot {
	featureCode = normalizedFeatureCode(featureCode)
	snapshot := featureLimitSnapshot{}
	highestNonStackableLimit := 0
	hasNonStackableLimit := false

	for _, pkg := range packages {
		if !pkg.IsActive || !pkg.includesFeature(featureCode) {
			continue
		}
		snapshot.HasFeatureAssignment = true

		limitValue := pkg.GetFeatureLimit(featureCode)
		if limitValue == nil {
			continue
		}
		if *limitValue == -1 {
			snapshot.Unlimited = true
			snapshot.Limit = 0
			return snapshot
		}
		if pkg.IsStackable {
			snapshot.Limit += *limitValue
			continue
		}
		if !hasNonStackableLimit || *limitValue > highestNonStackableLimit {
			highestNonStackableLimit = *limitValue
			hasNonStackableLimit = true
		}
	}

	if hasNonStackableLimit {
		snapshot.Limit += highestNonStackableLimit
	}
	return snapshot
}
