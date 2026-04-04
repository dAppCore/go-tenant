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
