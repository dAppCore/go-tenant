// SPDX-License-Identifier: EUPL-1.2

package tenant

import "dappco.re/go"

// Sentinel errors for the tenant package.

// ErrNoWorkspaceContext is returned when context carries no workspace.
//
//	ws, err := tenant.WorkspaceFromCtx(ctx)
//	// err == tenant.ErrNoWorkspaceContext when no workspace was injected
var ErrNoWorkspaceContext = core.E("tenant", "no workspace in context", nil)

// ErrNoUserContext is returned when context carries no user.
//
//	user, err := tenant.UserFromCtx(ctx)
//	// err == tenant.ErrNoUserContext when no user was injected
var ErrNoUserContext = core.E("tenant", "no user in context", nil)

// ErrWorkspaceNotFound is returned when slug or UUID does not resolve.
//
//	ws, err := ten.GetWorkspace(ctx, "nonexistent")
//	// err == tenant.ErrWorkspaceNotFound
var ErrWorkspaceNotFound = core.E("tenant", "workspace not found", nil)

// ErrFeatureNotFound is returned when the feature code is unknown.
//
//	feat, err := client.GetFeature(ctx, "unknown_feature")
//	// err == tenant.ErrFeatureNotFound
var ErrFeatureNotFound = core.E("tenant", "feature not found", nil)

// ErrEntitlementDenied is returned by EntitlementResult.AsError() on denial.
//
//	if err := svc.Can(ctx, ws, "pages", 1).AsError(); err != nil { ... }
var ErrEntitlementDenied = core.E("tenant", "entitlement denied", nil)

// ErrClientTimeout is returned when the PHP API does not respond in time.
//
//	ws, err := client.GetWorkspaceBySlug(ctx, "acme")
//	// err == tenant.ErrClientTimeout after 10s
var ErrClientTimeout = core.E("tenant", "api client timeout", nil)
