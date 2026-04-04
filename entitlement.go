// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"context"
	"sort"
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
//	if err := svc.Can(ctx, ws, "pages", 1).AsError(); err != nil { return err }
func (r EntitlementResult) AsError() error {
	if r.Allowed {
		return nil
	}
	return ErrEntitlementDenied
}

// Allow constructs an allowed result with usage context.
//
//	return tenant.Allow("pages", &limit, &used)
func Allow(featureCode string, limit, used *int) EntitlementResult {
	result := EntitlementResult{Allowed: true, FeatureCode: featureCode, Limit: limit, Used: used}
	if limit != nil && used != nil {
		remaining := *limit - *used
		if remaining < 0 {
			remaining = 0
		}
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
		remaining := *limit - *used
		if remaining < 0 {
			remaining = 0
		}
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
	return &localEntitlementService{cache: cache, client: client}
}

type localEntitlementService struct {
	cache  *TenantCache
	client *TenantClient
}

type packageLimitSnapshot struct {
	Limit                int
	HasFeatureAssignment bool
	Unlimited            bool
}

func (s *localEntitlementService) Can(ctx context.Context, ws *Workspace, featureCode string, quantity int) EntitlementResult {
	featureCode = normalizedFeatureCode(featureCode)
	if ws == nil {
		return Deny(featureCode, "no workspace provided", nil, nil)
	}
	if quantity < 0 {
		quantity = 1
	}
	feature, err := s.loadFeature(ctx, featureCode)
	if err != nil {
		return Deny(featureCode, err.Error(), nil, nil)
	}
	poolCode := normalizedFeatureCode(feature.PoolCode())
	packages, _ := s.loadPackages(ctx, ws.UUID)
	boosts, _ := s.loadBoosts(ctx, ws.UUID)
	used, _ := s.loadUsage(ctx, ws.UUID, poolCode)

	packageLimit := packageLimitForFeature(packages, poolCode)
	if packageLimit.Unlimited {
		return AllowUnlimited(featureCode)
	}

	hasEnableBoost := false
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
			hasEnableBoost = true
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
		if packageLimit.HasFeatureAssignment || hasEnableBoost {
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

func (s *localEntitlementService) RecordUsage(ctx context.Context, ws *Workspace, featureCode string, quantity int, userID *int64, metadata map[string]any) error {
	featureCode = normalizedFeatureCode(featureCode)
	if ws == nil {
		return ErrNoWorkspaceContext
	}
	if quantity <= 0 {
		quantity = 1
	}
	feature, err := s.loadFeature(ctx, featureCode)
	if err != nil {
		return err
	}
	poolCode := normalizedFeatureCode(feature.PoolCode())
	if s.client != nil {
		if err := s.client.RecordUsage(ctx, ws.UUID, featureCode, quantity, userID, metadata); err != nil {
			return err
		}
	}
	if s.cache != nil {
		if s.client != nil {
			_ = s.cache.InvalidateWorkspace(ws.UUID)
		} else if used, ok := s.cache.GetUsage(ws.UUID, poolCode); ok {
			_ = s.cache.SetUsage(ws.UUID, poolCode, used+quantity)
		} else {
			_ = s.cache.SetUsage(ws.UUID, poolCode, quantity)
		}
	}
	return nil
}

func (s *localEntitlementService) GetUsageSummary(ctx context.Context, ws *Workspace) ([]UsageSummaryItem, error) {
	if ws == nil {
		return nil, ErrNoWorkspaceContext
	}
	codes := map[string]struct{}{}
	packages, _ := s.loadPackages(ctx, ws.UUID)
	for _, pkg := range packages {
		if !pkg.IsActive {
			continue
		}
		for _, feature := range pkg.Features {
			codes[normalizedFeatureCode(feature.FeatureCode)] = struct{}{}
		}
	}
	boosts, _ := s.loadBoosts(ctx, ws.UUID)
	for _, boost := range boosts {
		codes[normalizedFeatureCode(boost.FeatureCode)] = struct{}{}
	}

	items := make([]UsageSummaryItem, 0, len(codes))
	for code := range codes {
		feature, _ := s.loadFeature(ctx, code)
		if feature == nil {
			feature = &Feature{Code: code, Name: code}
		}
		limit, unlimited, used := s.summaryForFeature(ctx, ws.UUID, feature)
		item := UsageSummaryItem{
			FeatureCode: code,
			FeatureName: feature.Name,
			Limit:       limit,
			Used:        used,
			Unlimited:   unlimited,
			ResetType:   feature.ResetType,
		}
		if limit != nil && used != nil {
			remaining := *limit - *used
			if remaining < 0 {
				remaining = 0
			}
			item.Remaining = &remaining
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].FeatureCode < items[j].FeatureCode
	})
	return items, nil
}

func (s *localEntitlementService) InvalidateWorkspace(wsUUID string) {
	if s.cache != nil {
		_ = s.cache.InvalidateWorkspace(wsUUID)
	}
}

func (s *localEntitlementService) loadFeature(ctx context.Context, code string) (*Feature, error) {
	code = normalizedFeatureCode(code)
	if s.cache != nil {
		if feature, ok := s.cache.GetFeature(code); ok {
			return feature, nil
		}
	}
	if s.client == nil {
		return nil, ErrFeatureNotFound
	}
	feature, err := s.client.GetFeature(ctx, code)
	if err != nil {
		return nil, err
	}
	if s.cache != nil {
		_ = s.cache.SetFeature(feature)
	}
	return feature, nil
}

func (s *localEntitlementService) loadPackages(ctx context.Context, wsUUID string) ([]Package, bool) {
	if s.cache != nil {
		if packages, ok := s.cache.GetPackages(wsUUID); ok {
			return packages, true
		}
	}
	if s.client == nil {
		return nil, false
	}
	packages, err := s.client.GetPackagesForWorkspace(ctx, wsUUID)
	if err != nil {
		return nil, false
	}
	if s.cache != nil {
		_ = s.cache.SetPackages(wsUUID, packages)
	}
	return packages, true
}

func (s *localEntitlementService) loadBoosts(ctx context.Context, wsUUID string) ([]Boost, bool) {
	if s.cache != nil {
		if boosts, ok := s.cache.GetBoosts(wsUUID); ok {
			return boosts, true
		}
	}
	if s.client == nil {
		return nil, false
	}
	boosts, err := s.client.GetBoostsForWorkspace(ctx, wsUUID)
	if err != nil {
		return nil, false
	}
	if s.cache != nil {
		_ = s.cache.SetBoosts(wsUUID, boosts)
	}
	return boosts, true
}

func (s *localEntitlementService) loadUsage(ctx context.Context, wsUUID, featureCode string) (int, bool) {
	featureCode = normalizedFeatureCode(featureCode)
	if s.cache != nil {
		if used, ok := s.cache.GetUsage(wsUUID, featureCode); ok {
			return used, true
		}
	}
	if s.client == nil {
		return 0, false
	}
	used, err := s.client.GetCurrentUsage(ctx, wsUUID, featureCode)
	if err != nil {
		return 0, false
	}
	if s.cache != nil {
		_ = s.cache.SetUsage(wsUUID, featureCode, used)
	}
	return used, true
}

func (s *localEntitlementService) summaryForFeature(ctx context.Context, wsUUID string, feature *Feature) (*int, bool, *int) {
	poolCode := normalizedFeatureCode(feature.PoolCode())
	packages, _ := s.loadPackages(ctx, wsUUID)
	boosts, _ := s.loadBoosts(ctx, wsUUID)

	packageLimit := packageLimitForFeature(packages, poolCode)
	if packageLimit.Unlimited {
		return nil, true, s.loadUsedCount(ctx, wsUUID, poolCode)
	}

	hasEnableBoost := false
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
			return nil, true, s.loadUsedCount(ctx, wsUUID, poolCode)
		case BoostTypeEnable:
			hasEnableBoost = true
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
			return nil, true, s.loadUsedCount(ctx, wsUUID, poolCode)
		}
		return nil, false, s.loadUsedCount(ctx, wsUUID, poolCode)
	}
	if feature.IsBoolean() {
		if packageLimit.HasFeatureAssignment || hasEnableBoost {
			return nil, false, nil
		}
		return nil, false, nil
	}
	if !packageLimit.HasFeatureAssignment {
		return nil, false, s.loadUsedCount(ctx, wsUUID, poolCode)
	}
	used, _ := s.loadUsage(ctx, wsUUID, poolCode)
	return &packageLimit.Limit, false, &used
}

func (s *localEntitlementService) loadUsedCount(ctx context.Context, wsUUID, featureCode string) *int {
	used, _ := s.loadUsage(ctx, wsUUID, featureCode)
	return &used
}

func packageLimitForFeature(packages []Package, featureCode string) packageLimitSnapshot {
	featureCode = normalizedFeatureCode(featureCode)
	snapshot := packageLimitSnapshot{}
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
