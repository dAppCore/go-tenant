// SPDX-License-Identifier: EUPL-1.2

package tenant

import "time"

// Boost is a temporary or permanent addition to a feature limit.
// Assigned to a workspace outside the base package.
//
//	if boost.IsUsable() { limit += boost.Remaining() }
type Boost struct {
	ID               int64       `json:"id"`
	WorkspaceID      *int64      `json:"workspace_id"`
	FeatureCode      string      `json:"feature_code"`
	BoostType        BoostType   `json:"boost_type"`
	DurationType     string      `json:"duration_type"`
	LimitValue       int         `json:"limit_value"`
	ConsumedQuantity int         `json:"consumed_quantity"`
	Status           BoostStatus `json:"status"`
	StartsAt         *time.Time  `json:"starts_at"`
	ExpiresAt        *time.Time  `json:"expires_at"`
}

// BoostType controls how the boost contributes to the effective limit.
type BoostType string

const (
	BoostTypeAddLimit  BoostType = "add_limit" // adds N to the package limit
	BoostTypeEnable    BoostType = "enable"    // boolean boost marker in the PHP model
	BoostTypeUnlimited BoostType = "unlimited" // removes the cap entirely
)

// BoostStatus reflects the current lifecycle state.
type BoostStatus string

const (
	BoostStatusActive    BoostStatus = "active"
	BoostStatusExhausted BoostStatus = "exhausted"
	BoostStatusExpired   BoostStatus = "expired"
	BoostStatusCancelled BoostStatus = "cancelled"
)

// IsUsable reports whether this boost can contribute to a limit check now.
//
//	for _, b := range boosts { if b.IsUsable() { effectiveLimit += b.Remaining() } }
func (b Boost) IsUsable() bool {
	if b.Status != BoostStatusActive {
		return false
	}
	if b.StartsAt != nil && time.Now().Before(*b.StartsAt) {
		return false
	}
	if b.ExpiresAt != nil && time.Now().After(*b.ExpiresAt) {
		return false
	}
	if b.BoostType == BoostTypeAddLimit && b.Remaining() <= 0 {
		return false
	}
	return true
}

// Remaining returns the unconsumed portion of an add_limit boost.
// Returns -1 for unlimited boosts. Returns 0 for exhausted boosts.
//
//	remaining := boost.Remaining()  // 7 if limit_value=10 consumed=3
func (b Boost) Remaining() int {
	switch b.BoostType {
	case BoostTypeUnlimited:
		return -1
	case BoostTypeEnable:
		return 0
	default:
		remaining := b.LimitValue - b.ConsumedQuantity
		if remaining < 0 {
			return 0
		}
		return remaining
	}
}
