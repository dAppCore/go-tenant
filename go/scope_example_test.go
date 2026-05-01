// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"context"
	"net/http"

	"dappco.re/go"
)

func ExampleNewWorkspaceScope() {
	scope := NewWorkspaceScope(&Tenant{})
	scope.WithStrict(false)
}

func ExampleWorkspaceScope_WithStrict() {
	scope := NewWorkspaceScope(&Tenant{})
	scope.WithStrict(false)
}

func ExampleWorkspaceScope_Middleware() {
	scope := NewWorkspaceScope(&Tenant{}).WithStrict(false)
	_ = scope.Middleware()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
}

func ExampleWorkspaceScope_RequireWorkspace() {
	scope := NewWorkspaceScope(&Tenant{})
	_ = scope.RequireWorkspace()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
}

func ExampleWorkspaceScope_ScopeFunc() {
	cache := NewTenantCache(nil)
	cache.SetWorkspace(&Workspace{UUID: "uuid-7", Slug: "acme"})
	tenantService := &Tenant{cache: cache}
	scope := NewWorkspaceScope(tenantService)
	scope.ScopeFunc(context.Background(), "acme", func(ctx context.Context) core.Result {
		return WorkspaceFromCtx(ctx)
	})
}
