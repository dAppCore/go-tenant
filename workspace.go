// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"context"
	"time"
)

// Workspace is the tenancy boundary. All resources belong to a workspace.
// Resolved from HTTP header (X-Workspace-ID or X-Workspace-Slug) or subdomain.
//
//	ws, err := tenant.WorkspaceFromCtx(ctx)
//	result := svc.Can(ctx, ws, "pages", 1)
type Workspace struct {
	ID          int64          `json:"id"`
	UUID        string         `json:"uuid"`
	Slug        string         `json:"slug"`
	Name        string         `json:"name"`
	Domain      string         `json:"domain"`
	Icon        string         `json:"icon"`
	Colour      string         `json:"colour"`
	Description string         `json:"description"`
	Type        string         `json:"type"`
	IsActive    bool           `json:"is_active"`
	Settings    map[string]any `json:"settings"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// WorkspaceContext is a request-scoped holder for workspace and user values.
//
//	ctxHolder := tenant.WorkspaceContext{Context: r.Context()}.
//		WithWorkspace(&tenant.Workspace{UUID: "ws-7", Slug: "acme"}).
//		WithUser(&tenant.User{UUID: "user-9", Email: "ada@example.uk"})
type WorkspaceContext struct {
	Context context.Context
}

func (c WorkspaceContext) baseContext() context.Context {
	if c.Context != nil {
		return c.Context
	}
	return context.Background()
}

// WithWorkspace returns a new holder carrying the workspace.
//
//	contextHolder := tenant.WorkspaceContext{}.WithWorkspace(workspace)
func (c WorkspaceContext) WithWorkspace(ws *Workspace) WorkspaceContext {
	c.Context = WithWorkspace(c.baseContext(), ws)
	return c
}

// WithUser returns a new holder carrying the user.
//
//	contextHolder := tenant.WorkspaceContext{}.WithUser(authenticatedUser)
func (c WorkspaceContext) WithUser(user *User) WorkspaceContext {
	c.Context = WithUser(c.baseContext(), user)
	return c
}

// Workspace resolves the workspace from the holder context.
//
//	ws, err := tenant.WorkspaceContext{Context: ctx}.Workspace()
func (c WorkspaceContext) Workspace() (*Workspace, error) {
	return WorkspaceFromCtx(c.baseContext())
}

// User resolves the user from the holder context.
//
//	user, err := tenant.WorkspaceContext{Context: ctx}.User()
func (c WorkspaceContext) User() (*User, error) {
	return UserFromCtx(c.baseContext())
}

// contextKey is an unexported type for context keys to prevent collisions.
type contextKey int

const (
	workspaceContextKey contextKey = iota
	userContextKey
)

// WorkspaceFromCtx retrieves the current workspace from context.
// Returns ErrNoWorkspaceContext if no workspace was injected.
//
//	ws, err := tenant.WorkspaceFromCtx(ctx)
//	if err != nil { return core.E("tenant", "no workspace context", err) }
func WorkspaceFromCtx(ctx context.Context) (*Workspace, error) {
	if ctx == nil {
		return nil, ErrNoWorkspaceContext
	}
	ws, _ := ctx.Value(workspaceContextKey).(*Workspace)
	if ws == nil {
		return nil, ErrNoWorkspaceContext
	}
	return ws, nil
}

// WithWorkspace returns a new context carrying the workspace.
//
//	ctx = tenant.WithWorkspace(ctx, ws)
func WithWorkspace(ctx context.Context, ws *Workspace) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if ws == nil {
		return ctx
	}
	return context.WithValue(ctx, workspaceContextKey, ws)
}

// cloneWorkspace returns a deep copy of the workspace to prevent cache mutation.
// Settings map is copied so modifications to the clone do not affect the original.
//
//	safe := cloneWorkspace(cachedWorkspace)
func cloneWorkspace(ws *Workspace) *Workspace {
	if ws == nil {
		return nil
	}
	clone := *ws
	if ws.Settings != nil {
		clone.Settings = make(map[string]any, len(ws.Settings))
		for key, value := range ws.Settings {
			clone.Settings[key] = value
		}
	}
	return &clone
}
