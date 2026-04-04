// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"context"
	"net/http"
	"net/netip"
	"strings"
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
//
//	router.Use(scope.Middleware())
func (s *WorkspaceScope) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			workspace, _ := s.resolveWorkspace(r)
			if workspace == nil {
				if s.strict {
					http.Error(w, "workspace required", http.StatusUnauthorized)
					return
				}
				next.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithWorkspace(r.Context(), workspace)))
		})
	}
}

// RequireWorkspace is a second-stage middleware that aborts 401 if no workspace is
// in context. For use after Middleware() when strict=false on the outer scope.
//
//	authGroup.Use(scope.Middleware(), scope.RequireWorkspace())
//
//	authGroup.Use(scope.Middleware(), scope.RequireWorkspace())
func (s *WorkspaceScope) RequireWorkspace() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := WorkspaceFromCtx(r.Context()); err != nil {
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
//	err := scope.ScopeFunc(ctx, "acme", func(ctx context.Context) error {
//	    ws, _ := tenant.WorkspaceFromCtx(ctx)
//	    return ten.Can(ctx, ws, "pages", 1).AsError()
//	})
func (s *WorkspaceScope) ScopeFunc(ctx context.Context, slug string, fn func(context.Context) error) error {
	if s == nil || s.tenant == nil {
		return ErrNoWorkspaceContext
	}
	workspace, err := s.tenant.GetWorkspace(ctx, slug)
	if err != nil {
		if err == ErrWorkspaceNotFound {
			return ErrNoWorkspaceContext
		}
		return err
	}
	if workspace == nil {
		return ErrNoWorkspaceContext
	}
	return fn(WithWorkspace(ctx, workspace))
}

func (s *WorkspaceScope) resolveWorkspace(r *http.Request) (*Workspace, error) {
	if s == nil || s.tenant == nil || r == nil {
		return nil, ErrNoWorkspaceContext
	}
	if idValue := r.Header.Get("X-Workspace-ID"); idValue != "" {
		if id, err := parseInt64(idValue); err == nil {
			return s.tenant.GetWorkspaceByID(r.Context(), id)
		}
		return nil, ErrNoWorkspaceContext
	}
	if slug := r.Header.Get("X-Workspace-Slug"); slug != "" {
		return s.tenant.GetWorkspace(r.Context(), slug)
	}
	if host := cleanHost(r.Host); host != "" {
		if workspace, err := s.tenant.GetWorkspaceBySubdomain(r.Context(), host); err == nil && workspace != nil {
			return workspace, nil
		}
	}
	if slug := r.URL.Query().Get("workspace"); slug != "" {
		return s.tenant.GetWorkspace(r.Context(), slug)
	}
	return nil, ErrNoWorkspaceContext
}

func cleanHost(host string) string {
	if host == "" {
		return ""
	}
	if parsed, err := netip.ParseAddrPort(host); err == nil {
		return parsed.Addr().String()
	}
	if colon := strings.LastIndex(host, ":"); colon > 0 {
		return host[:colon]
	}
	return host
}

func parseInt64(value string) (int64, error) {
	var result int64
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, ErrNoWorkspaceContext
		}
		result = result*10 + int64(r-'0')
	}
	return result, nil
}
