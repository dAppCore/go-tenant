// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"context"
	"net/http"

	"dappco.re/go"
)

// WorkspaceScope resolves and injects workspace context for HTTP handlers.
// Resolution order (matches PHP's RequireWorkspaceContext.resolveWorkspace):
//
//  1. X-Workspace-ID header (integer workspace ID)
//
//  2. X-Workspace-Slug header (slug string)
//
//  3. Host header subdomain (e.g., "acme" from "acme.host.uk.com")
//
//  4. ?workspace= query parameter (slug)
//
//     router.Use(tenant.NewWorkspaceScope(tenantSvc).Middleware())
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
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			result := s.resolveWorkspace(r)
			workspace, _ := core.Cast[*Workspace](result)
			if workspace == nil {
				if s.strict {
					http.Error(w, "workspace required", http.StatusUnauthorized)
					return
				}
				next.ServeHTTP(w, r)
				return
			}
			requestContext := WorkspaceContext{Context: r.Context()}.WithWorkspace(workspace)
			next.ServeHTTP(w, r.WithContext(requestContext.Context))
		})
	}
}

// RequireWorkspace is a second-stage middleware that aborts 401 if no workspace is
// in context. For use after Middleware() when strict=false on the outer scope.
//
//	authGroup.Use(scope.Middleware(), scope.RequireWorkspace())
func (s *WorkspaceScope) RequireWorkspace() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if result := WorkspaceFromCtx(r.Context()); !result.OK {
				http.Error(w, "workspace required", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ScopeFunc is a helper for non-HTTP contexts (agents, background jobs).
// It resolves a workspace by slug, injects it into ctx, then runs fn.
// Returns ErrNoWorkspaceContext if the slug cannot be resolved.
//
//	r := scope.ScopeFunc(ctx, "acme", func(ctx context.Context) core.Result {
//	    ws := tenant.WorkspaceFromCtx(ctx).Value.(*tenant.Workspace)
//	    return ten.Can(ctx, ws, "pages", 1).AsError()
//	})
func (s *WorkspaceScope) ScopeFunc(ctx context.Context, slug string, fn func(context.Context) core.Result) core.Result {
	if s == nil || s.tenant == nil {
		return core.Fail(ErrNoWorkspaceContext)
	}
	r := s.tenant.GetWorkspace(ctx, slug)
	if !r.OK {
		if r.Value == ErrWorkspaceNotFound {
			return core.Fail(ErrNoWorkspaceContext)
		}
		return r
	}
	workspace := r.Value.(*Workspace)
	if workspace == nil {
		return core.Fail(ErrNoWorkspaceContext)
	}
	requestContext := WorkspaceContext{Context: ctx}.WithWorkspace(workspace)
	return fn(requestContext.Context)
}

func (s *WorkspaceScope) resolveWorkspace(r *http.Request) core.Result {
	if s == nil || s.tenant == nil || r == nil {
		return core.Fail(ErrNoWorkspaceContext)
	}
	if idValue := r.Header.Get("X-Workspace-ID"); idValue != "" {
		if parsed := parseInt64(idValue); parsed.OK {
			return s.tenant.GetWorkspaceByID(r.Context(), parsed.Value.(int64))
		}
		return core.Fail(ErrNoWorkspaceContext)
	}
	if slug := r.Header.Get("X-Workspace-Slug"); slug != "" {
		return s.tenant.GetWorkspace(r.Context(), slug)
	}
	if r.Host != "" {
		if result := s.tenant.GetWorkspaceBySubdomain(r.Context(), r.Host); result.OK && result.Value != nil {
			return result
		}
	}
	if slug := r.URL.Query().Get("workspace"); slug != "" {
		return s.tenant.GetWorkspace(r.Context(), slug)
	}
	return core.Fail(ErrNoWorkspaceContext)
}

// parseInt64 parses a decimal string to int64 without importing strconv.
// Only accepts digit characters — no signs, whitespace, or other formatting.
//
//	parseInt64("42")     // Value is int64(42)
//	parseInt64("bogus")  // failed Result
func parseInt64(value string) core.Result {
	var result int64
	for _, r := range value {
		if r < '0' || r > '9' {
			return core.Fail(ErrNoWorkspaceContext)
		}
		result = result*10 + int64(r-'0')
	}
	return core.Ok(result)
}
