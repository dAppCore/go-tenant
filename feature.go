// SPDX-License-Identifier: EUPL-1.2

package tenant

// Feature defines a single capability that can be enabled, limited, or unlimited.
//
//	f := tenant.Feature{Code: "pages", Type: tenant.FeatureTypeLimit}
//	if f.IsBoolean() { ... }
type Feature struct {
	ID                int64       `json:"id"`
	Code              string      `json:"code"`
	Name              string      `json:"name"`
	Description       string      `json:"description"`
	Category          string      `json:"category"`
	Type              FeatureType `json:"type"`
	ResetType         ResetType   `json:"reset_type"`
	RollingWindowDays int         `json:"rolling_window_days"`
	ParentFeatureID   *int64      `json:"parent_feature_id"`
	ParentCode        *string     `json:"parent_code"`
	IsActive          bool        `json:"is_active"`
}

// FeatureType controls how a feature is evaluated.
//
//	feature := tenant.Feature{Code: "pages", Type: tenant.FeatureTypeLimit}
type FeatureType string

const (
	FeatureTypeBoolean   FeatureType = "boolean"
	FeatureTypeLimit     FeatureType = "limit"
	FeatureTypeUnlimited FeatureType = "unlimited"
)

// ResetType controls when usage counters reset.
//
//	feature := tenant.Feature{Code: "pages", ResetType: tenant.ResetMonthly}
type ResetType string

const (
	ResetNone    ResetType = "none"
	ResetMonthly ResetType = "monthly"
	ResetRolling ResetType = "rolling"
)

// IsBoolean reports whether this feature is a boolean toggle.
func (f Feature) IsBoolean() bool { return f.Type == FeatureTypeBoolean }

// HasLimit reports whether this feature has a numeric cap.
func (f Feature) HasLimit() bool { return f.Type == FeatureTypeLimit }

// IsUnlimited reports whether this feature has no cap.
func (f Feature) IsUnlimited() bool { return f.Type == FeatureTypeUnlimited }

// PoolCode returns the feature code to use for usage pooling.
// Child features pool against their parent's usage counter.
//
//	code := f.PoolCode()  // "pages" for child feature "pages.bio"
func (f Feature) PoolCode() string {
	if f.ParentCode != nil {
		return *f.ParentCode
	}
	return f.Code
}
