# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

Multi-tenancy, workspace isolation, user identity, feature entitlements, and usage tracking for CoreGO services. Go consumer of the PHP `core-tenant` REST API with go-store backed caching. Module: `dappco.re/go/core/tenant`

## Spec

Implementation spec at `docs/RFC.md`. AX design principles at `docs/RFC-025-AGENT-EXPERIENCE.md`. Read both fully before writing any code.

## Commands

```bash
go test ./...                        # Run all tests
go test -v -run TestTenant_Can_Good  # Run single test
go test -race ./...                  # Race detector (must pass before commit)
go test -cover ./...                 # Coverage
go vet ./...                         # Vet
```

## Architecture

**API consumer, not DB owner.** PHP owns the database. Go calls PHP's REST API for all mutations and queries, then caches the results locally via go-store. The cache is the hot path — the API is the cold path and the source of truth.

**Cache-first reads.** EntitlementResult for a workspace+feature pair is cached for 5 minutes. Usage records cached for 60 seconds. Cache keys are namespaced by workspace UUID.

**Context propagation.** WorkspaceScope middleware injects the resolved Workspace into context.Context. Downstream code calls `tenant.WorkspaceFromCtx(ctx)`.

**Entitlement cascade.** Packages (stackable) + boosts = effective limit. Pool code resolution for child features.

## File Map

| File | Purpose |
|------|---------|
| `tenant.go` | Root service, `Register` factory, passthrough methods |
| `workspace.go` | `Workspace` struct, context propagation |
| `user.go` | `User` struct, `UserTier` constants |
| `feature.go` | `Feature`, `FeatureType`, `ResetType` |
| `package.go` | `Package` — bundle of features with limits |
| `boost.go` | `Boost` — temporary or permanent limit increase |
| `usage.go` | `UsageRecord`, `UsageSummaryItem` |
| `entitlement.go` | `EntitlementResult`, `EntitlementService` interface |
| `client.go` | `TenantClient` — PHP REST API transport |
| `cache.go` | `TenantCache` — go-store backed local cache with TTL |
| `scope.go` | `WorkspaceScope` — middleware and context propagation |
| `alert.go` | `UsageAlert` — threshold monitoring (80/90/100%) |
| `errors.go` | Sentinel errors via `core.E()` |

## Banned Imports

| Banned | Use Instead |
|--------|-------------|
| `fmt` | `core.Sprintf`, `core.Print` |
| `log` | `core.Print`, `core.Error` |
| `errors` | `core.E("tenant", message, cause)` |
| `os` | `c.Fs()` |
| `os/exec` | `c.Process()` |
| `strings` | `core.Contains`, `core.TrimPrefix`, etc. |
| `path/filepath` | `core.JoinPath`, `core.PathBase` |
| `encoding/json` | `core.JSONMarshalString`, `core.JSONUnmarshalString` |

## Coding Standards

- **UK English**: colour, organisation, centre (never American spellings)
- **Strict types**: All parameters and return types
- **Comments**: Usage examples with concrete values, not prose descriptions
- **Error handling**: All errors via `core.E("tenant", message, cause)`
- **No ORM**: Plain Go structs with JSON tags
- **Licence**: EUPL-1.2

## Test Naming

```
TestFilename_Function_{Good,Bad,Ugly}
```

All three categories mandatory per exported function. Use cache-only mode (`NewLocalEntitlementService(cache, nil)`) for unit tests.

## Commit Convention

```
type(scope): description

Co-Authored-By: Virgil <virgil@lethean.io>
```
