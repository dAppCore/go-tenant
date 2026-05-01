// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"context"
	"net/http"
	"net/http/httptest"

	"dappco.re/go"
)

func TestScope_NewWorkspaceScope_Good(t *core.T) {
	tenantService, _ := testTenant(t)
	scope := NewWorkspaceScope(tenantService)
	core.AssertNotNil(t, scope)
	core.AssertTrue(t, scope.strict)
}

func TestScope_NewWorkspaceScope_Bad(t *core.T) {
	scope := NewWorkspaceScope(nil)
	core.AssertNotNil(t, scope)
	core.AssertNil(t, scope.tenant)
}

func TestScope_NewWorkspaceScope_Ugly(t *core.T) {
	scope := NewWorkspaceScope(&Tenant{})
	core.AssertNotNil(t, scope.tenant)
	core.AssertTrue(t, scope.strict)
}

func TestScope_WorkspaceScope_WithStrict_Good(t *core.T) {
	scope := NewWorkspaceScope(nil).WithStrict(false)
	core.AssertFalse(t, scope.strict)
	core.AssertNotNil(t, scope)
}

func TestScope_WorkspaceScope_WithStrict_Bad(t *core.T) {
	scope := NewWorkspaceScope(nil).WithStrict(true)
	core.AssertTrue(t, scope.strict)
	core.AssertNil(t, scope.tenant)
}

func TestScope_WorkspaceScope_WithStrict_Ugly(t *core.T) {
	scope := NewWorkspaceScope(nil)
	again := scope.WithStrict(false).WithStrict(true)
	core.AssertEqual(t, scope, again)
	core.AssertTrue(t, again.strict)
}

func TestScope_WorkspaceScope_Middleware_Good(t *core.T) {
	tenantService, _ := testTenant(t)
	scope := NewWorkspaceScope(tenantService)
	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set("X-Workspace-Slug", "acme")
	rec := httptest.NewRecorder()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if result := WorkspaceFromCtx(r.Context()); result.OK {
			w.WriteHeader(http.StatusAccepted)
		}
	})
	scope.Middleware()(next).ServeHTTP(rec, req)
	core.AssertEqual(t, http.StatusAccepted, rec.Code)
}

func TestScope_WorkspaceScope_Middleware_Bad(t *core.T) {
	scope := NewWorkspaceScope(&Tenant{}).WithStrict(true)
	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	rec := httptest.NewRecorder()
	scope.Middleware()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(rec, req)
	core.AssertEqual(t, http.StatusUnauthorized, rec.Code)
}

func TestScope_WorkspaceScope_Middleware_Ugly(t *core.T) {
	scope := NewWorkspaceScope(&Tenant{}).WithStrict(false)
	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	rec := httptest.NewRecorder()
	scope.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(rec, req)
	core.AssertEqual(t, http.StatusNoContent, rec.Code)
}

func TestScope_WorkspaceScope_RequireWorkspace_Good(t *core.T) {
	scope := NewWorkspaceScope(nil)
	req := httptest.NewRequest(http.MethodGet, "/resource", nil).WithContext(WithWorkspace(context.Background(), testWorkspace()))
	rec := httptest.NewRecorder()
	scope.RequireWorkspace()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})).ServeHTTP(rec, req)
	core.AssertEqual(t, http.StatusAccepted, rec.Code)
}

func TestScope_WorkspaceScope_RequireWorkspace_Bad(t *core.T) {
	scope := NewWorkspaceScope(nil)
	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	rec := httptest.NewRecorder()
	scope.RequireWorkspace()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(rec, req)
	core.AssertEqual(t, http.StatusUnauthorized, rec.Code)
}

func TestScope_WorkspaceScope_RequireWorkspace_Ugly(t *core.T) {
	scope := NewWorkspaceScope(nil)
	req := httptest.NewRequest(http.MethodGet, "/resource", nil).WithContext(WithWorkspace(nil, testWorkspace()))
	rec := httptest.NewRecorder()
	scope.RequireWorkspace()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(rec, req)
	core.AssertEqual(t, http.StatusNoContent, rec.Code)
}

func TestScope_WorkspaceScope_ScopeFunc_Good(t *core.T) {
	tenantService, _ := testTenant(t)
	scope := NewWorkspaceScope(tenantService)
	result := scope.ScopeFunc(context.Background(), "acme", func(ctx context.Context) core.Result {
		return WorkspaceFromCtx(ctx)
	})
	requireResultOK(t, result)
	core.AssertEqual(t, "uuid-7", result.Value.(*Workspace).UUID)
}

func TestScope_WorkspaceScope_ScopeFunc_Bad(t *core.T) {
	var scope *WorkspaceScope
	result := scope.ScopeFunc(context.Background(), "acme", func(context.Context) core.Result { return core.Ok(nil) })
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoWorkspaceContext, result.Value)
}

func TestScope_WorkspaceScope_ScopeFunc_Ugly(t *core.T) {
	scope := NewWorkspaceScope(&Tenant{})
	result := scope.ScopeFunc(context.Background(), "missing", func(context.Context) core.Result { return core.Ok(nil) })
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoWorkspaceContext, result.Value)
}
