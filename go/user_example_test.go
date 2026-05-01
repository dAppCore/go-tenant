// SPDX-License-Identifier: EUPL-1.2

package tenant

import "context"

func ExampleUserTier_MaxWorkspaces() {
	TierApollo.MaxWorkspaces()
}

func ExampleUserTier_HasFeature() {
	TierApollo.HasFeature("api_access")
}

func ExampleUserFromCtx() {
	ctx := WithUser(context.Background(), &User{UUID: "user-9"})
	UserFromCtx(ctx)
}

func ExampleWithUser() {
	ctx := WithUser(context.Background(), &User{UUID: "user-9"})
	UserFromCtx(ctx)
}
