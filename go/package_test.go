// SPDX-License-Identifier: EUPL-1.2

package tenant

import "dappco.re/go"

func TestPackage_Package_GetFeatureLimit_Good(t *core.T) {
	pkg := Package{Features: []PackageFeature{{FeatureCode: "pages", LimitValue: new(10)}}}
	limit := pkg.GetFeatureLimit("pages")
	core.AssertNotNil(t, limit)
	core.AssertEqual(t, 10, *limit)
}

func TestPackage_Package_GetFeatureLimit_Bad(t *core.T) {
	pkg := Package{Features: []PackageFeature{{FeatureCode: "pages", LimitValue: new(10)}}}
	limit := pkg.GetFeatureLimit("missing")
	core.AssertNil(t, limit)
	core.AssertEqual(t, 1, len(pkg.Features))
}

func TestPackage_Package_GetFeatureLimit_Ugly(t *core.T) {
	pkg := Package{Features: []PackageFeature{{FeatureCode: " Pages ", LimitValue: nil}}}
	limit := pkg.GetFeatureLimit("pages")
	core.AssertNil(t, limit)
	core.AssertTrue(t, pkg.includesFeature("pages"))
}
