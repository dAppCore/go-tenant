// SPDX-License-Identifier: EUPL-1.2

package tenant

func ExamplePackage_GetFeatureLimit() {
	limit := 10
	pkg := Package{Features: []PackageFeature{{FeatureCode: "pages", LimitValue: &limit}}}
	pkg.GetFeatureLimit("pages")
}
