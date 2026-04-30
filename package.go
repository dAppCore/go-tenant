// SPDX-License-Identifier: EUPL-1.2

package tenant

// Package is a bundle of features with defined limits. Workspaces are assigned packages.
// Multiple packages may stack when is_stackable is true.
//
//	pkg.GetFeatureLimit("pages")  // 10 for starter, nil for unlimited
type Package struct {
	ID            int64            `json:"id"`
	Code          string           `json:"code"`
	Name          string           `json:"name"`
	Description   string           `json:"description"`
	IsStackable   bool             `json:"is_stackable"`
	IsBasePackage bool             `json:"is_base_package"`
	IsActive      bool             `json:"is_active"`
	IsPublic      bool             `json:"is_public"`
	Features      []PackageFeature `json:"features"`
}

// PackageFeature is the join between Package and Feature with the limit value.
type PackageFeature struct {
	FeatureCode string `json:"feature_code"`
	LimitValue  *int   `json:"limit_value"` // nil = boolean feature; -1 = unlimited
}

// GetFeatureLimit returns the limit for featureCode in this package.
// Returns nil if the feature is not in this package.
//
//	if lim := pkg.GetFeatureLimit("pages"); lim != nil { total += *lim }
func (p Package) GetFeatureLimit(featureCode string) *int {
	featureCode = normalizedFeatureCode(featureCode)
	for _, f := range p.Features {
		if normalizedFeatureCode(f.FeatureCode) == featureCode {
			return f.LimitValue
		}
	}
	return nil
}

// includesFeature checks whether this package has any assignment for featureCode,
// regardless of the limit value.
//
//	pkg.includesFeature("pages")  // true if "pages" is in pkg.Features
func (p Package) includesFeature(featureCode string) bool {
	featureCode = normalizedFeatureCode(featureCode)
	for _, feature := range p.Features {
		if normalizedFeatureCode(feature.FeatureCode) == featureCode {
			return true
		}
	}
	return false
}
