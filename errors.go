// SPDX-License-Identifier: EUPL-1.2

package tenant

import "errors"

// Sentinel errors for the tenant package.

// ErrNoWorkspaceContext is returned when context carries no workspace.
//
//	ws, err := tenant.WorkspaceFromCtx(ctx)
//	// err == tenant.ErrNoWorkspaceContext when no workspace was injected
var ErrNoWorkspaceContext = errors.New("tenant: no workspace in context")

// ErrNoUserContext is returned when context carries no user.
//
//	user, err := tenant.UserFromCtx(ctx)
//	// err == tenant.ErrNoUserContext when no user was injected
var ErrNoUserContext = errors.New("tenant: no user in context")

// ErrWorkspaceNotFound is returned when slug or UUID does not resolve.
//
//	ws, err := ten.GetWorkspace(ctx, "nonexistent")
//	// err == tenant.ErrWorkspaceNotFound
var ErrWorkspaceNotFound = errors.New("tenant: workspace not found")

// ErrFeatureNotFound is returned when the feature code is unknown.
//
//	feat, err := client.GetFeature(ctx, "unknown_feature")
//	// err == tenant.ErrFeatureNotFound
var ErrFeatureNotFound = errors.New("tenant: feature not found")

// ErrEntitlementDenied is returned by EntitlementResult.AsError() on denial.
//
//	if err := svc.Can(ctx, ws, "pages", 1).AsError(); err != nil { ... }
var ErrEntitlementDenied = errors.New("tenant: entitlement denied")

// ErrClientTimeout is returned when the PHP API does not respond in time.
//
//	ws, err := client.GetWorkspaceBySlug(ctx, "acme")
//	// err == tenant.ErrClientTimeout after 10s
var ErrClientTimeout = errors.New("tenant: api client timeout")
