// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"context"
	"net/http"
)

// WorkspaceScope resolves and injects workspace context for HTTP handlers.
// Resolution order (matches PHP's RequireWorkspaceContext.resolveWorkspace):
//  1. X-Workspace-ID header (integer workspace ID)
//  2. X-Workspace-Slug header (slug string)
//  3. Host header subdomain (e.g., "acme" from "acme.host.uk.com")
//  4. ?workspace= query parameter (slug)
//
//	router.Use(tenant.NewWorkspaceScope(tenantSvc).Middleware())
type WorkspaceScope struct {
	tenant *Tenant
	strict bool
}

// NewWorkspaceScope creates a new scope resolver. Strict mode is enabled by default.
//
//	scope := tenant.NewWorkspaceScope(ten)
func NewWorkspaceScope(t *Tenant) *WorkspaceScope {
	return &WorkspaceScope{tenant: t, strict: true}
}

// WithStrict enables or disables strict mode.
// Strict (default true): requests without a resolvable workspace are rejected 401.
// Non-strict: middleware continues without workspace — handler receives empty context.
//
//	scope := tenant.NewWorkspaceScope(t).WithStrict(false)  // public routes
func (s *WorkspaceScope) WithStrict(strict bool) *WorkspaceScope {
	s.strict = strict
	return s
}

// Middleware returns an http.Handler middleware for use with net/http or Gin.
//
//	router.Use(scope.Middleware())
func (s *WorkspaceScope) Middleware() func(http.Handler) http.Handler {
	// TODO: implement — resolve workspace from request, inject into context
	return func(next http.Handler) http.Handler {
		return next
	}
}

// RequireWorkspace is a second-stage middleware that aborts 401 if no workspace is
// in context. For use after Middleware() when strict=false on the outer scope.
//
//	authGroup.Use(scope.Middleware(), scope.RequireWorkspace())
func (s *WorkspaceScope) RequireWorkspace() func(http.Handler) http.Handler {
	// TODO: implement
	return func(next http.Handler) http.Handler {
		return next
	}
}

// ScopeFunc is a helper for non-HTTP contexts (agents, background jobs).
// It resolves a workspace by slug, injects it into ctx, then runs fn.
// Returns ErrNoWorkspaceContext if the slug cannot be resolved.
//
//	err := scope.ScopeFunc(ctx, "acme", func(ctx context.Context) error {
//	    ws, _ := tenant.WorkspaceFromCtx(ctx)
//	    return ten.Can(ctx, ws, "pages", 1).AsError()
//	})
func (s *WorkspaceScope) ScopeFunc(ctx context.Context, slug string, fn func(context.Context) error) error {
	// TODO: implement — resolve workspace, inject, call fn
	return ErrNoWorkspaceContext
}
