// SPDX-License-Identifier: EUPL-1.2

package tenant

import "time"

// AlertThreshold constants for usage percentage triggers.
const (
	AlertThresholdWarning  = 80  // 80% of limit consumed
	AlertThresholdCritical = 90  // 90% of limit consumed
	AlertThresholdLimit    = 100 // limit reached
)

// UsageAlert represents a triggered threshold condition for a workspace+feature.
//
//	ten.OnUsageAlert(func(alert tenant.UsageAlert) { notifyUsageTeam(alert.WorkspaceUUID, alert.Threshold) })
type UsageAlert struct {
	WorkspaceUUID string
	FeatureCode   string
	Threshold     int // 80, 90, or 100
	Used          int
	Limit         int
	Percentage    float64
	TriggeredAt   time.Time
}

// AlertHandler is called when a usage threshold is crossed.
//
//	tenantService.OnUsageAlert(func(alert tenant.UsageAlert) { logUsageAlert(alert) })
type AlertHandler func(UsageAlert)
