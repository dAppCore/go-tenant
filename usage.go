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

// UsageSummaryItem is one row in the usage summary dashboard.
//
//	summary, _ := svc.GetUsageSummary(ctx, ws)
//	for _, item := range summary { fmt.Printf("%s: %d/%d\n", item.FeatureCode, item.Used, item.Limit) }
type UsageSummaryItem struct {
	FeatureCode string
	FeatureName string
	Limit       *int
	Used        *int
	Remaining   *int
	Unlimited   bool
	ResetType   ResetType
}
