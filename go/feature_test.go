// SPDX-License-Identifier: EUPL-1.2

package tenant

import "dappco.re/go"

func TestFeature_Feature_IsBoolean_Good(t *core.T) {
	feature := Feature{Type: FeatureTypeBoolean}
	core.AssertTrue(t, feature.IsBoolean())
	core.AssertFalse(t, feature.HasLimit())
}

func TestFeature_Feature_IsBoolean_Bad(t *core.T) {
	feature := Feature{Type: FeatureTypeLimit}
	core.AssertFalse(t, feature.IsBoolean())
	core.AssertTrue(t, feature.HasLimit())
}

func TestFeature_Feature_IsBoolean_Ugly(t *core.T) {
	feature := Feature{}
	core.AssertFalse(t, feature.IsBoolean())
	core.AssertFalse(t, feature.IsUnlimited())
}

func TestFeature_Feature_HasLimit_Good(t *core.T) {
	feature := Feature{Type: FeatureTypeLimit}
	core.AssertTrue(t, feature.HasLimit())
	core.AssertFalse(t, feature.IsBoolean())
}

func TestFeature_Feature_HasLimit_Bad(t *core.T) {
	feature := Feature{Type: FeatureTypeBoolean}
	core.AssertFalse(t, feature.HasLimit())
	core.AssertTrue(t, feature.IsBoolean())
}

func TestFeature_Feature_HasLimit_Ugly(t *core.T) {
	feature := Feature{Type: FeatureTypeUnlimited}
	core.AssertFalse(t, feature.HasLimit())
	core.AssertTrue(t, feature.IsUnlimited())
}

func TestFeature_Feature_IsUnlimited_Good(t *core.T) {
	feature := Feature{Type: FeatureTypeUnlimited}
	core.AssertTrue(t, feature.IsUnlimited())
	core.AssertFalse(t, feature.HasLimit())
}

func TestFeature_Feature_IsUnlimited_Bad(t *core.T) {
	feature := Feature{Type: FeatureTypeLimit}
	core.AssertFalse(t, feature.IsUnlimited())
	core.AssertTrue(t, feature.HasLimit())
}

func TestFeature_Feature_IsUnlimited_Ugly(t *core.T) {
	feature := Feature{}
	core.AssertFalse(t, feature.IsUnlimited())
	core.AssertFalse(t, feature.IsBoolean())
}

func TestFeature_Feature_PoolCode_Good(t *core.T) {
	feature := Feature{Code: "pages"}
	core.AssertEqual(t, "pages", feature.PoolCode())
	core.AssertEqual(t, "pages", feature.Code)
}

func TestFeature_Feature_PoolCode_Bad(t *core.T) {
	parent := "pages"
	feature := Feature{Code: "pages.bio", ParentCode: &parent}
	core.AssertEqual(t, "pages", feature.PoolCode())
	core.AssertEqual(t, "pages.bio", feature.Code)
}

func TestFeature_Feature_PoolCode_Ugly(t *core.T) {
	feature := Feature{Code: ""}
	core.AssertEqual(t, "", feature.PoolCode())
	core.AssertNil(t, feature.ParentCode)
}
