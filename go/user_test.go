// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"dappco.re/go"
)

func TestUser_UserTier_MaxWorkspaces_Good(t *core.T) {
	core.AssertEqual(t, 5, TierApollo.MaxWorkspaces())
	core.AssertEqual(t, TierApollo, UserTier("apollo"))
	core.AssertGreater(t, TierApollo.MaxWorkspaces(), TierFree.MaxWorkspaces())
}

func TestUser_UserTier_MaxWorkspaces_Bad(t *core.T) {
	core.AssertEqual(t, 1, TierFree.MaxWorkspaces())
	core.AssertEqual(t, 1, UserTier("unknown").MaxWorkspaces())
	core.AssertFalse(t, TierFree.MaxWorkspaces() < 1)
}

func TestUser_UserTier_MaxWorkspaces_Ugly(t *core.T) {
	core.AssertEqual(t, -1, TierHades.MaxWorkspaces())
	core.AssertEqual(t, TierHades, UserTier("hades"))
	core.AssertLess(t, TierHades.MaxWorkspaces(), TierFree.MaxWorkspaces())
}

func TestUser_UserTier_HasFeature_Good(t *core.T) {
	core.AssertTrue(t, TierApollo.HasFeature("api_access"))
	core.AssertFalse(t, TierApollo.HasFeature("pages"))
	core.AssertTrue(t, UserTier("apollo").HasFeature(" api_access "))
}

func TestUser_UserTier_HasFeature_Bad(t *core.T) {
	core.AssertFalse(t, TierFree.HasFeature("api_access"))
	core.AssertEqual(t, TierFree, UserTier("free"))
	core.AssertFalse(t, UserTier("unknown").HasFeature("api_access"))
}

func TestUser_UserTier_HasFeature_Ugly(t *core.T) {
	core.AssertTrue(t, TierHades.HasFeature("anything"))
	core.AssertTrue(t, TierApollo.HasFeature(" API_ACCESS "))
	core.AssertFalse(t, TierFree.HasFeature("anything"))
}

func TestUser_UserFromCtx_Good(t *core.T) {
	result := UserFromCtx(WithUser(t.Context(), testUser()))
	requireResultOK(t, result)
	core.AssertEqual(t, "user-9", result.Value.(*User).UUID)
}

func TestUser_UserFromCtx_Bad(t *core.T) {
	result := UserFromCtx(t.Context())
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoUserContext, result.Value)
}

func TestUser_UserFromCtx_Ugly(t *core.T) {
	result := UserFromCtx(nil)
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoUserContext, result.Value)
}

func TestUser_WithUser_Good(t *core.T) {
	ctx := WithUser(t.Context(), testUser())
	result := UserFromCtx(ctx)
	requireResultOK(t, result)
	core.AssertEqual(t, "user-9", result.Value.(*User).UUID)
}

func TestUser_WithUser_Bad(t *core.T) {
	ctx := WithUser(t.Context(), nil)
	result := UserFromCtx(ctx)
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoUserContext, result.Value)
}

func TestUser_WithUser_Ugly(t *core.T) {
	ctx := WithUser(nil, testUser())
	result := UserFromCtx(ctx)
	requireResultOK(t, result)
	core.AssertEqual(t, "ada@example.uk", result.Value.(*User).Email)
}
