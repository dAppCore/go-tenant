// SPDX-License-Identifier: EUPL-1.2

package tenant

import "time"

// UsageRecord tracks consumption of a limit-based feature.
// Written after a successful operation — never before.
//
//	svc.RecordUsage(ctx, ws, "pages", 1, userID, nil)
type UsageRecord struct {
	ID          int64          `json:"id"`
	WorkspaceID int64          `json:"workspace_id"`
	FeatureCode string         `json:"feature_code"`
	Quantity    int            `json:"quantity"`
	UserID      *int64         `json:"user_id"`
	Metadata    map[string]any `json:"metadata"`
	RecordedAt  time.Time      `json:"recorded_at"`
}

// newUsageRecord builds a record with normalised values for API payloads.
//
//	record := newUsageRecord(7, "pages", 1, &userID, map[string]any{"page_id": 42})
func newUsageRecord(workspaceID int64, featureCode string, quantity int, userID *int64, metadata map[string]any) UsageRecord {
	if quantity <= 0 {
		quantity = 1
	}
	record := UsageRecord{
		WorkspaceID: workspaceID,
		FeatureCode: normalizedFeatureCode(featureCode),
		Quantity:    quantity,
		RecordedAt:  time.Now().UTC(),
	}
	if userID != nil {
		record.UserID = cloneInt64(userID)
	}
	if len(metadata) > 0 {
		record.Metadata = cloneMetadata(metadata)
	}
	return record
}

// addQuantity returns a new record with extra quantity accumulated.
//
//	record := newUsageRecord(7, "pages", 1, nil, nil).addQuantity(2)
func (r UsageRecord) addQuantity(quantity int) UsageRecord {
	if quantity <= 0 {
		return r
	}
	r.Quantity += quantity
	return r
}

func (r UsageRecord) payload() map[string]any {
	payload := map[string]any{
		"feature_code": r.FeatureCode,
		"quantity":     r.Quantity,
		"metadata":     r.Metadata,
	}
	if r.UserID != nil {
		payload["user_id"] = *r.UserID
	}
	return payload
}

// cloneInt64 returns a copy of the pointer to prevent shared mutation.
//
//	safe := cloneInt64(&userID)
func cloneInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

// cloneMetadata returns a shallow copy of the metadata map to prevent shared mutation.
//
//	safe := cloneMetadata(map[string]any{"page_id": 42})
func cloneMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return nil
	}
	clone := make(map[string]any, len(metadata))
	for key, value := range metadata {
		clone[key] = value
	}
	return clone
}

// UsageSummaryItem is one row in the usage summary dashboard.
//
//	summary, _ := svc.GetUsageSummary(ctx, ws)
//	for _, item := range summary { core.Println(item.FeatureCode, *item.Used, "/", *item.Limit) }
type UsageSummaryItem struct {
	FeatureCode string
	FeatureName string
	Limit       *int
	Used        *int
	Remaining   *int
	Unlimited   bool
	ResetType   ResetType
}
