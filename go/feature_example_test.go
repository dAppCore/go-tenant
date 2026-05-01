// SPDX-License-Identifier: EUPL-1.2

package tenant

func ExampleFeature_IsBoolean() {
	feature := Feature{Type: FeatureTypeBoolean}
	feature.IsBoolean()
}

func ExampleFeature_HasLimit() {
	feature := Feature{Type: FeatureTypeLimit}
	feature.HasLimit()
}

func ExampleFeature_IsUnlimited() {
	feature := Feature{Type: FeatureTypeUnlimited}
	feature.IsUnlimited()
}

func ExampleFeature_PoolCode() {
	parent := "pages"
	feature := Feature{Code: "pages.bio", ParentCode: &parent}
	feature.PoolCode()
}
