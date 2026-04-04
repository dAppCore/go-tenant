---
module: dappco.re/go/core/tenant
repo: core/go-tenant
lang: go
tier: lib
depends:
  - code/core/go
  - code/core/go/store
  - code/core/go/api
tags:
  - tenancy
  - workspace
  - entitlements
  - identity
  - access-control
  - usage-tracking
---
# go-tenant RFC — Multi-Tenancy and Entitlements

> An agent should be able to implement this library from this document alone.

**Module:** `dappco.re/go/core/tenant`
**Repository:** `core/go-tenant`
**Files:** 14

---

## 1. Overview

go-tenant is the Go counterpart of the PHP `core-tenant` package. It provides workspace
isolation, user identity, feature entitlements, and usage tracking for CoreGO services.

The Go layer is a **consumer**, not the schema owner. PHP owns the database. Go calls
PHP's REST API for all mutations and queries, then caches the results locally via
go-store. The cache is the hot path — the API is the cold path and the source of truth.

Three primary consumers:

| Consumer | What it needs |
|----------|--------------|
| `core/api` | Workspace context from request headers; entitlement gate middleware |
| `core/mcp` | Agent identity resolution; workspace-scoped tool access |
| Product services | `Can()` checks before write operations; `RecordUsage()` after |

---

## 2. File Map

| File | Purpose |
|------|---------|
| `tenant.go` | Package entry point — `Tenant` service, `Register` factory |
| `workspace.go` | `Workspace` struct, `WorkspaceContext` request-scoped holder |
| `user.go` | `User` struct, `UserTier` constants |
| `feature.go` | `Feature`, `FeatureType`, `ResetType` |
| `package.go` | `Package` — bundle of features with limits |
| `boost.go` | `Boost` — temporary or permanent limit increase |
| `usage.go` | `UsageRecord`, usage accumulation helpers |
| `entitlement.go` | `EntitlementResult`, `EntitlementService` interface |
| `client.go` | `TenantClient` — PHP REST API transport |
| `cache.go` | `TenantCache` — go-store backed local cache with TTL |
| `scope.go` | `WorkspaceScope` — middleware and context propagation |
| `alert.go` | `UsageAlert` — threshold monitoring (80/90/100%) |
| `errors.go` | Sentinel errors via `core.E()` |
| `tenant_test.go` | Unit tests (TestFilename_Function_{Good,Bad,Ugly}) |

---

## 3. Key Design Decisions

- **API consumer, not DB owner.** All writes go through the PHP REST API. Go never
  touches the entitlements schema directly. This preserves the single source of truth
  and avoids schema drift.
- **Cache-first reads.** `EntitlementResult` for a workspace+feature pair is cached
  for 5 minutes (matching PHP's `CACHE_TTL = 300`). Usage records are cached for 60
  seconds. Cache keys are namespaced by workspace UUID.
- **Context propagation via `context.Context`.** PHP uses Laravel's request attributes
  and session. Go uses `context.Context`. `WorkspaceScope` middleware injects the
  resolved `Workspace` into the context. Downstream code calls
  `tenant.WorkspaceFromCtx(ctx)` to retrieve it.
- **No ORM.** Structs are plain Go types with JSON tags. Persistence is via the PHP
  API; local state is via go-store.
- **Entitlement cascade is client-side.** PHP enforces entitlements server-side too,
  but Go implements the same cascade logic locally (namespace → workspace → user tier)
  to avoid a round-trip per feature check.

---

## 4. Domain Types

### 4.1 Workspace

```go
// Workspace is the tenancy boundary. All resources belong to a workspace.
// Resolved from HTTP header (X-Workspace-ID or X-Workspace-Slug) or subdomain.
//
//   ws, err := tenant.WorkspaceFromCtx(ctx)
//   result := svc.Can(ctx, ws, "pages", 1)
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
```

### 4.2 User

```go
// User is the authenticated identity. Tier controls feature access for personal namespaces.
//
//   u, err := tenant.UserFromCtx(ctx)
//   if u.Tier == tenant.TierHades { ... }
type User struct {
    ID              int64      `json:"id"`
    UUID            string     `json:"uuid"`
    Name            string     `json:"name"`
    Email           string     `json:"email"`
    Tier            UserTier   `json:"tier"`
    TierExpiresAt   *time.Time `json:"tier_expires_at"`
    EmailVerifiedAt *time.Time `json:"email_verified_at"`
    CreatedAt       time.Time  `json:"created_at"`
}

// UserTier maps to PHP's UserTier enum.
type UserTier string

const (
    TierFree   UserTier = "free"
    TierApollo UserTier = "apollo"  // Standard paid tier
    TierHades  UserTier = "hades"   // Premium tier
)

// MaxWorkspaces returns the workspace limit for this tier.
// Returns -1 for unlimited (Hades).
//
//   if u.Tier.MaxWorkspaces() != -1 && count >= u.Tier.MaxWorkspaces() { ... }
func (t UserTier) MaxWorkspaces() int {
    switch t {
    case TierApollo:
        return 5
    case TierHades:
        return -1
    default:
        return 1
    }
}

// HasFeature checks whether this tier includes the named feature code.
//
//   if u.Tier.HasFeature("api_access") { ... }
func (t UserTier) HasFeature(code string) bool
```

### 4.3 Feature

```go
// Feature defines a single capability that can be enabled, limited, or unlimited.
//
//   f := tenant.Feature{Code: "pages", Type: tenant.FeatureTypeLimit}
//   if f.IsBoolean() { ... }
type Feature struct {
    ID                int64       `json:"id"`
    Code              string      `json:"code"`
    Name              string      `json:"name"`
    Description       string      `json:"description"`
    Category          string      `json:"category"`
    Type              FeatureType `json:"type"`
    ResetType         ResetType   `json:"reset_type"`
    RollingWindowDays int         `json:"rolling_window_days"`
    ParentFeatureID   *int64      `json:"parent_feature_id"`
    ParentCode        *string     `json:"parent_code"`
    IsActive          bool        `json:"is_active"`
}

// FeatureType controls how a feature is evaluated.
type FeatureType string

const (
    FeatureTypeBoolean   FeatureType = "boolean"
    FeatureTypeLimit     FeatureType = "limit"
    FeatureTypeUnlimited FeatureType = "unlimited"
)

// ResetType controls when usage counters reset.
type ResetType string

const (
    ResetNone    ResetType = "none"
    ResetMonthly ResetType = "monthly"
    ResetRolling ResetType = "rolling"
)

// IsBoolean reports whether this feature is a boolean toggle.
func (f Feature) IsBoolean() bool { return f.Type == FeatureTypeBoolean }

// HasLimit reports whether this feature has a numeric cap.
func (f Feature) HasLimit() bool { return f.Type == FeatureTypeLimit }

// IsUnlimited reports whether this feature has no cap.
func (f Feature) IsUnlimited() bool { return f.Type == FeatureTypeUnlimited }

// PoolCode returns the feature code to use for usage pooling.
// Child features pool against their parent's usage counter.
//
//   code := f.PoolCode()  // "pages" for child feature "pages.bio"
func (f Feature) PoolCode() string {
    if f.ParentCode != nil {
        return *f.ParentCode
    }
    return f.Code
}
```

### 4.4 Package

```go
// Package is a bundle of features with defined limits. Workspaces are assigned packages.
// Multiple packages may stack when is_stackable is true.
//
//   pkg.GetFeatureLimit("pages")  // 10 for starter, nil for unlimited
type Package struct {
    ID            int64            `json:"id"`
    Code          string           `json:"code"`
    Name          string           `json:"name"`
    Description   string           `json:"description"`
    IsStackable   bool             `json:"is_stackable"`
    IsBasePackage bool             `json:"is_base_package"`
    IsActive      bool             `json:"is_active"`
    IsPublic      bool             `json:"is_public"`
    Features      []PackageFeature `json:"features"`
}

// PackageFeature is the join between Package and Feature with the limit value.
type PackageFeature struct {
    FeatureCode string `json:"feature_code"`
    LimitValue  *int   `json:"limit_value"`  // nil = boolean feature; -1 = unlimited
}

// GetFeatureLimit returns the limit for featureCode in this package.
// Returns nil if the feature is not in this package.
//
//   if lim := pkg.GetFeatureLimit("pages"); lim != nil { total += *lim }
func (p Package) GetFeatureLimit(featureCode string) *int {
    for _, f := range p.Features {
        if f.FeatureCode == featureCode {
            return f.LimitValue
        }
    }
    return nil
}
```

### 4.5 Boost

```go
// Boost is a temporary or permanent addition to a feature limit.
// Assigned to a workspace outside the base package.
//
//   if boost.IsUsable() { limit += boost.Remaining() }
type Boost struct {
    ID               int64       `json:"id"`
    WorkspaceID      *int64      `json:"workspace_id"`
    FeatureCode      string      `json:"feature_code"`
    BoostType        BoostType   `json:"boost_type"`
    DurationType     string      `json:"duration_type"`
    LimitValue       int         `json:"limit_value"`
    ConsumedQuantity int         `json:"consumed_quantity"`
    Status           BoostStatus `json:"status"`
    StartsAt         *time.Time  `json:"starts_at"`
    ExpiresAt        *time.Time  `json:"expires_at"`
}

// BoostType controls how the boost contributes to the effective limit.
type BoostType string

const (
    BoostTypeAddLimit  BoostType = "add_limit"  // adds N to the package limit
    BoostTypeEnable    BoostType = "enable"      // enables a boolean feature
    BoostTypeUnlimited BoostType = "unlimited"   // removes the cap entirely
)

// BoostStatus reflects the current lifecycle state.
type BoostStatus string

const (
    BoostStatusActive    BoostStatus = "active"
    BoostStatusExhausted BoostStatus = "exhausted"
    BoostStatusExpired   BoostStatus = "expired"
    BoostStatusCancelled BoostStatus = "cancelled"
)

// IsUsable reports whether this boost can contribute to a limit check now.
//
//   for _, b := range boosts { if b.IsUsable() { effectiveLimit += b.Remaining() } }
func (b Boost) IsUsable() bool

// Remaining returns the unconsumed portion of an add_limit boost.
// Returns -1 for unlimited boosts. Returns 0 for exhausted boosts.
//
//   remaining := boost.Remaining()  // 7 if limit_value=10 consumed=3
func (b Boost) Remaining() int
```

### 4.6 UsageRecord

```go
// UsageRecord tracks consumption of a limit-based feature.
// Written after a successful operation — never before.
//
//   svc.RecordUsage(ctx, ws, "pages", 1, userID, nil)
type UsageRecord struct {
    ID          int64          `json:"id"`
    WorkspaceID int64          `json:"workspace_id"`
    FeatureCode string         `json:"feature_code"`
    Quantity    int            `json:"quantity"`
    UserID      *int64         `json:"user_id"`
    Metadata    map[string]any `json:"metadata"`
    RecordedAt  time.Time      `json:"recorded_at"`
}
```

### 4.7 EntitlementResult

```go
// EntitlementResult is the value object returned by every entitlement check.
// It carries the decision plus usage context for display and logging.
//
//   result := svc.Can(ctx, ws, "pages", 1)
//   if result.IsDenied() { return core.E("tenant", result.Reason, nil) }
//   fmt.Printf("Used %d of %d pages\n", *result.Used, *result.Limit)
type EntitlementResult struct {
    Allowed     bool
    Reason      string  // human-readable denial message; empty if allowed
    FeatureCode string
    Limit       *int    // nil for boolean features
    Used        *int    // nil for boolean features
    Remaining   *int    // nil for boolean features
    Unlimited   bool
}

// IsAllowed reports whether the requested quantity can be consumed.
func (r EntitlementResult) IsAllowed() bool { return r.Allowed }

// IsDenied is the inverse of IsAllowed.
func (r EntitlementResult) IsDenied() bool { return !r.Allowed }

// UsagePercent returns 0–100 usage as float64. Returns nil for unlimited/boolean.
//
//   if pct := result.UsagePercent(); pct != nil && *pct >= 80 { warnNearLimit() }
func (r EntitlementResult) UsagePercent() *float64

// IsNearLimit reports whether usage exceeds 80% of the limit.
func (r EntitlementResult) IsNearLimit() bool

// IsAtLimit reports whether remaining capacity is zero.
func (r EntitlementResult) IsAtLimit() bool

// AsError converts to nil (allowed) or ErrEntitlementDenied (denied).
// Convenience for callers that want to treat denial as an error.
//
//   if err := svc.Can(ctx, ws, "pages", 1).AsError(); err != nil { return err }
func (r EntitlementResult) AsError() error

// Constructors

// Allow constructs an allowed result with usage context.
//
//   return tenant.Allow("pages", &limit, &used)
func Allow(featureCode string, limit, used *int) EntitlementResult

// Deny constructs a denied result with a human-readable reason.
//
//   return tenant.Deny("pages", "Your plan does not include pages.", nil, nil)
func Deny(featureCode, reason string, limit, used *int) EntitlementResult

// AllowUnlimited constructs an unlimited allowed result.
//
//   return tenant.AllowUnlimited("pages")
func AllowUnlimited(featureCode string) EntitlementResult
```

---

## 5. Context Propagation

```go
// WorkspaceFromCtx retrieves the current workspace from context.
// Returns ErrNoWorkspaceContext if no workspace was injected.
//
//   ws, err := tenant.WorkspaceFromCtx(ctx)
//   if err != nil { return core.E("tenant", "no workspace context", err) }
func WorkspaceFromCtx(ctx context.Context) (*Workspace, error)

// UserFromCtx retrieves the authenticated user from context.
// Returns ErrNoUserContext if no user was injected.
//
//   user, err := tenant.UserFromCtx(ctx)
func UserFromCtx(ctx context.Context) (*User, error)

// WithWorkspace returns a new context carrying the workspace.
//
//   ctx = tenant.WithWorkspace(ctx, ws)
func WithWorkspace(ctx context.Context, ws *Workspace) context.Context

// WithUser returns a new context carrying the user.
//
//   ctx = tenant.WithUser(ctx, user)
func WithUser(ctx context.Context, user *User) context.Context
```

---

## 6. EntitlementService

The central entitlement engine. Checks whether a workspace can consume a feature
and tracks usage. Cache-first; falls back to the PHP API on miss.

```go
// EntitlementService is the interface all entitlement checks go through.
// The default implementation calls TenantClient and caches results via TenantCache.
type EntitlementService interface {
    // Can checks whether the workspace can consume quantity units of featureCode.
    // quantity=1 for standard single-resource checks.
    //
    //   result := svc.Can(ctx, ws, "pages", 1)
    //   if result.IsDenied() { return core.E("pages", result.Reason, nil) }
    Can(ctx context.Context, ws *Workspace, featureCode string, quantity int) EntitlementResult

    // RecordUsage records consumption of featureCode after a successful operation.
    // Invalidates the usage cache entry for this workspace+feature.
    //
    //   svc.RecordUsage(ctx, ws, "pages", 1, &userID, map[string]any{"page_id": 42})
    RecordUsage(ctx context.Context, ws *Workspace, featureCode string, quantity int, userID *int64, metadata map[string]any) error

    // GetUsageSummary returns all tracked features with current usage for ws.
    //
    //   summary, _ := svc.GetUsageSummary(ctx, ws)
    //   for _, item := range summary { fmt.Printf("%s: %d/%d\n", item.FeatureCode, item.Used, item.Limit) }
    GetUsageSummary(ctx context.Context, ws *Workspace) ([]UsageSummaryItem, error)

    // InvalidateWorkspace drops all cached entitlement data for ws.
    //
    //   svc.InvalidateWorkspace(ws.UUID)
    InvalidateWorkspace(wsUUID string)
}

// NewLocalEntitlementService creates the default implementation backed by cache and client.
// client may be nil for unit tests (cache-only mode — no API fallback on miss).
//
//   svc := tenant.NewLocalEntitlementService(cache, client)
//   svc := tenant.NewLocalEntitlementService(cache, nil)  // test mode
func NewLocalEntitlementService(cache *TenantCache, client *TenantClient) EntitlementService

// UsageSummaryItem is one row in the usage summary dashboard.
type UsageSummaryItem struct {
    FeatureCode string
    FeatureName string
    Limit       *int
    Used        *int
    Remaining   *int
    Unlimited   bool
    ResetType   ResetType
}
```

### 6.1 Entitlement Cascade

When checking a workspace, limits are aggregated from:

1. All active packages assigned to the workspace (summed when stackable)
2. All active, non-expired boosts for the feature

```
pool_code    = feature.PoolCode()
total_limit  = sum(pkg.GetFeatureLimit(pool_code) for each active package)
             + sum(boost.Remaining() for each usable add_limit boost)

if any boost.BoostType == BoostTypeUnlimited → return AllowUnlimited(featureCode)
if total_limit == nil                         → feature not in any package → Deny
if feature.IsBoolean()                        → no usage check → Allow
if used + quantity > total_limit              → Deny with usage context
else                                          → Allow with usage context
```

Pool code resolution: child features (`pages.bio`) accumulate usage against the
parent pool (`pages`). `feature.PoolCode()` handles the indirection.

---

## 7. TenantClient — PHP API Transport

```go
// TenantClient calls the PHP REST API to read and mutate tenant data.
// Reads are cached by TenantCache; the client is called on cache miss.
//
//   client := tenant.NewTenantClient("https://api.host.uk.com", token)
//   ws, err := client.GetWorkspaceBySlug(ctx, "acme")
type TenantClient struct {
    baseURL    string
    token      string
    httpClient *http.Client
    timeout    time.Duration
}

func NewTenantClient(baseURL, token string, opts ...ClientOption) *TenantClient

type ClientOption func(*TenantClient)

// WithTimeout overrides the default 10-second request timeout.
//
//   client := tenant.NewTenantClient(url, token, tenant.WithTimeout(5*time.Second))
func WithTimeout(d time.Duration) ClientOption

// GetWorkspaceBySlug fetches a workspace by its slug.
//
//   ws, err := client.GetWorkspaceBySlug(ctx, "acme")
func (c *TenantClient) GetWorkspaceBySlug(ctx context.Context, slug string) (*Workspace, error)

// GetWorkspaceByUUID fetches a workspace by UUID.
//
//   ws, err := client.GetWorkspaceByUUID(ctx, "550e8400-...")
func (c *TenantClient) GetWorkspaceByUUID(ctx context.Context, uuid string) (*Workspace, error)

// GetWorkspaceBySubdomain resolves a hostname to a workspace.
// Checks slug match first, then domain prefix match (matches PHP's WorkspaceService).
//
//   ws, err := client.GetWorkspaceBySubdomain(ctx, "acme.host.uk.com")
func (c *TenantClient) GetWorkspaceBySubdomain(ctx context.Context, host string) (*Workspace, error)

// GetUser fetches the authenticated user by the bearer token on the client.
//
//   user, err := client.GetUser(ctx)
func (c *TenantClient) GetUser(ctx context.Context) (*User, error)

// GetPackagesForWorkspace returns all active packages assigned to the workspace.
//
//   pkgs, err := client.GetPackagesForWorkspace(ctx, ws.UUID)
func (c *TenantClient) GetPackagesForWorkspace(ctx context.Context, wsUUID string) ([]Package, error)

// GetBoostsForWorkspace returns all active, usable boosts for the workspace.
//
//   boosts, err := client.GetBoostsForWorkspace(ctx, ws.UUID)
func (c *TenantClient) GetBoostsForWorkspace(ctx context.Context, wsUUID string) ([]Boost, error)

// GetCurrentUsage returns total usage count for wsUUID+featureCode since reset boundary.
//
//   used, err := client.GetCurrentUsage(ctx, ws.UUID, "pages")
func (c *TenantClient) GetCurrentUsage(ctx context.Context, wsUUID, featureCode string) (int, error)

// RecordUsage POSTs a usage record to the PHP API.
//
//   err := client.RecordUsage(ctx, ws.UUID, "pages", 1, &userID, nil)
func (c *TenantClient) RecordUsage(ctx context.Context, wsUUID, featureCode string, quantity int, userID *int64, metadata map[string]any) error

// GetFeature fetches a single feature definition by code.
//
//   feat, err := client.GetFeature(ctx, "pages")
func (c *TenantClient) GetFeature(ctx context.Context, code string) (*Feature, error)
```

### 7.1 PHP API Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/v1/workspaces/{slug}` | Workspace by slug |
| GET | `/api/v1/workspaces/uuid/{uuid}` | Workspace by UUID |
| GET | `/api/v1/workspaces/subdomain/{host}` | Workspace by hostname |
| GET | `/api/v1/user` | Authenticated user |
| GET | `/api/v1/workspaces/{uuid}/packages` | Active packages |
| GET | `/api/v1/workspaces/{uuid}/boosts` | Active boosts |
| GET | `/api/v1/workspaces/{uuid}/usage/{code}` | Current usage count |
| POST | `/api/v1/workspaces/{uuid}/usage` | Record usage |
| GET | `/api/v1/features/{code}` | Feature definition |

All requests carry `Authorization: Bearer <token>` and `Accept: application/json`.
Error responses: `{"ok": false, "error": "message"}`.

---

## 8. TenantCache — go-store Backed Cache

```go
// TenantCache provides workspace-scoped caching over go-store.
// Groups are namespaced by workspace UUID to prevent cross-tenant leakage.
//
//   cache := tenant.NewTenantCache(store)
//   cache.SetPackages(ws.UUID, pkgs)
//   pkgs, ok := cache.GetPackages(ws.UUID)
type TenantCache struct {
    store *store.Store
}

func NewTenantCache(st *store.Store) *TenantCache

// TTL constants matching PHP's cache configuration.
const (
    TTLEntitlements = 5 * time.Minute   // packages, boosts, feature definitions
    TTLUsage        = 60 * time.Second  // usage counters (more volatile)
    TTLWorkspace    = 5 * time.Minute   // workspace record
    TTLUser         = 5 * time.Minute   // user record
)

// SetWorkspace stores the workspace record.
//
//   cache.SetWorkspace(ws)
func (c *TenantCache) SetWorkspace(ws *Workspace) error

// GetWorkspace retrieves a cached workspace by UUID. Returns nil, false on miss.
//
//   ws, ok := cache.GetWorkspace("550e8400-...")
func (c *TenantCache) GetWorkspace(uuid string) (*Workspace, bool)

// GetWorkspaceBySlug retrieves a cached workspace by slug via UUID indirection.
//
//   ws, ok := cache.GetWorkspaceBySlug("acme")
func (c *TenantCache) GetWorkspaceBySlug(slug string) (*Workspace, bool)

// SetPackages stores the active package list for a workspace.
//
//   cache.SetPackages(ws.UUID, pkgs)
func (c *TenantCache) SetPackages(wsUUID string, pkgs []Package) error

// GetPackages retrieves cached packages. Returns nil, false on miss.
//
//   pkgs, ok := cache.GetPackages(ws.UUID)
func (c *TenantCache) GetPackages(wsUUID string) ([]Package, bool)

// SetBoosts stores the active boost list for a workspace.
//
//   cache.SetBoosts(ws.UUID, boosts)
func (c *TenantCache) SetBoosts(wsUUID string, boosts []Boost) error

// GetBoosts retrieves cached boosts. Returns nil, false on miss.
//
//   boosts, ok := cache.GetBoosts(ws.UUID)
func (c *TenantCache) GetBoosts(wsUUID string) ([]Boost, bool)

// SetUsage stores the current usage count for a workspace+feature.
//
//   cache.SetUsage(ws.UUID, "pages", 7)
func (c *TenantCache) SetUsage(wsUUID, featureCode string, count int) error

// GetUsage retrieves a cached usage count. Returns 0, false on miss.
//
//   used, ok := cache.GetUsage(ws.UUID, "pages")
func (c *TenantCache) GetUsage(wsUUID, featureCode string) (int, bool)

// InvalidateWorkspace drops all cache entries for this workspace UUID.
// Called by RecordUsage and by external invalidation signals.
//
//   cache.InvalidateWorkspace(ws.UUID)
func (c *TenantCache) InvalidateWorkspace(wsUUID string) error

// SetFeature stores a feature definition by code. Features are global.
//
//   cache.SetFeature(feat)
func (c *TenantCache) SetFeature(feat *Feature) error

// GetFeature retrieves a cached feature definition by code.
//
//   feat, ok := cache.GetFeature("pages")
func (c *TenantCache) GetFeature(code string) (*Feature, bool)
```

Cache key scheme (all stored in go-store):

| Group | Key | TTL |
|-------|-----|-----|
| `ws:{uuid}:record` | `data` | 5 min |
| `ws:slug:{slug}` | `uuid` | 5 min |
| `ws:{uuid}:packages` | `data` | 5 min |
| `ws:{uuid}:boosts` | `data` | 5 min |
| `ws:{uuid}:usage:{code}` | `count` | 60 sec |
| `feature:{code}` | `data` | 5 min |

---

## 9. WorkspaceScope — Middleware and Context

The Go equivalent of PHP's `RequireWorkspaceContext` middleware and `BelongsToWorkspace`
trait. Resolves workspace from request metadata, injects into `context.Context`, and
provides a scope guard for handlers requiring a workspace.

```go
// WorkspaceScope resolves and injects workspace context for HTTP handlers.
// Resolution order (matches PHP's RequireWorkspaceContext.resolveWorkspace):
//   1. X-Workspace-ID header (integer workspace ID)
//   2. X-Workspace-Slug header (slug string)
//   3. Host header subdomain (e.g., "acme" from "acme.host.uk.com")
//   4. ?workspace= query parameter (slug)
//
//   router.Use(tenant.NewWorkspaceScope(tenantSvc).Middleware())
type WorkspaceScope struct {
    tenant *Tenant
    strict bool
}

func NewWorkspaceScope(t *Tenant) *WorkspaceScope

// WithStrict enables or disables strict mode.
// Strict (default true): requests without a resolvable workspace are rejected 401.
// Non-strict: middleware continues without workspace — handler receives empty context.
//
//   scope := tenant.NewWorkspaceScope(t).WithStrict(false)  // public routes
func (s *WorkspaceScope) WithStrict(strict bool) *WorkspaceScope

// Middleware returns an http.Handler middleware for use with net/http or Gin.
//
//   router.Use(scope.Middleware())
func (s *WorkspaceScope) Middleware() func(http.Handler) http.Handler

// RequireWorkspace is a second-stage middleware that aborts 401 if no workspace is
// in context. For use after Middleware() when strict=false on the outer scope.
//
//   authGroup.Use(scope.Middleware(), scope.RequireWorkspace())
func (s *WorkspaceScope) RequireWorkspace() func(http.Handler) http.Handler

// ScopeFunc is a helper for non-HTTP contexts (agents, background jobs).
// It resolves a workspace by slug, injects it into ctx, then runs fn.
// Returns ErrNoWorkspaceContext if the slug cannot be resolved.
//
//   err := scope.ScopeFunc(ctx, "acme", func(ctx context.Context) error {
//       ws, _ := tenant.WorkspaceFromCtx(ctx)
//       return ten.Can(ctx, ws, "pages", 1).AsError()
//   })
func (s *WorkspaceScope) ScopeFunc(ctx context.Context, slug string, fn func(context.Context) error) error
```

---

## 10. Tenant Service

The root service wiring together client, cache, and entitlement engine.
Registered with Core via `Register`.

```go
// Tenant is the root service. Register once per application instance.
// All tenant operations flow through this type.
//
//   ten := core.MustServiceFor[*tenant.Tenant](c, "tenant")
//   result := ten.Can(ctx, ws, "pages", 1)
type Tenant struct {
    *core.ServiceRuntime[TenantOptions]
    client       *TenantClient
    cache        *TenantCache
    entitlements EntitlementService
    alertHandlers []AlertHandler
}

// TenantOptions configures the tenant service via Core config.
type TenantOptions struct {
    APIURL   string        `json:"api_url"`   // PHP API base URL
    APIToken string        `json:"api_token"` // Bearer token
    Timeout  time.Duration `json:"timeout"`   // HTTP timeout (default 10s)
}

// Register is the Core service factory. Called by core.WithService.
//
//   core.New(core.WithService(tenant.Register))
func Register(c *core.Core) core.Result

// GetWorkspace resolves a workspace by slug. Checks cache first, then PHP API.
//
//   ws, err := ten.GetWorkspace(ctx, "acme")
func (t *Tenant) GetWorkspace(ctx context.Context, slug string) (*Workspace, error)

// GetWorkspaceByUUID resolves a workspace by UUID.
//
//   ws, err := ten.GetWorkspaceByUUID(ctx, "550e8400-...")
func (t *Tenant) GetWorkspaceByUUID(ctx context.Context, uuid string) (*Workspace, error)

// GetWorkspaceBySubdomain resolves the workspace for an incoming hostname.
//
//   ws, err := ten.GetWorkspaceBySubdomain(ctx, r.Host)
func (t *Tenant) GetWorkspaceBySubdomain(ctx context.Context, host string) (*Workspace, error)

// Can checks whether ws can consume quantity units of featureCode.
//
//   result := ten.Can(ctx, ws, "pages", 1)
func (t *Tenant) Can(ctx context.Context, ws *Workspace, featureCode string, quantity int) EntitlementResult

// RecordUsage records feature consumption for ws after a successful operation.
//
//   ten.RecordUsage(ctx, ws, "pages", 1, &userID, nil)
func (t *Tenant) RecordUsage(ctx context.Context, ws *Workspace, featureCode string, quantity int, userID *int64, metadata map[string]any) error

// GetUsageSummary returns all features with their current usage for ws.
//
//   items, err := ten.GetUsageSummary(ctx, ws)
func (t *Tenant) GetUsageSummary(ctx context.Context, ws *Workspace) ([]UsageSummaryItem, error)

// InvalidateWorkspace drops the local cache for ws.
//
//   ten.InvalidateWorkspace(ws.UUID)
func (t *Tenant) InvalidateWorkspace(wsUUID string)

// OnUsageAlert registers a handler invoked when a usage threshold is crossed.
// Multiple handlers can be registered; all fire in registration order.
//
//   ten.OnUsageAlert(func(a tenant.UsageAlert) { notify(a.WorkspaceUUID, a.Threshold) })
func (t *Tenant) OnUsageAlert(h AlertHandler)

// Scope returns a configured WorkspaceScope for middleware wiring.
//
//   router.Use(ten.Scope().Middleware())
func (t *Tenant) Scope() *WorkspaceScope
```

---

## 11. UsageAlert — Threshold Monitoring

```go
// AlertThreshold constants for usage percentage triggers.
const (
    AlertThresholdWarning  = 80   // 80% of limit consumed
    AlertThresholdCritical = 90   // 90% of limit consumed
    AlertThresholdLimit    = 100  // limit reached
)

// UsageAlert represents a triggered threshold condition for a workspace+feature.
type UsageAlert struct {
    WorkspaceUUID string
    FeatureCode   string
    Threshold     int     // 80, 90, or 100
    Used          int
    Limit         int
    Percentage    float64
    TriggeredAt   time.Time
}

// AlertHandler is called when a usage threshold is crossed.
type AlertHandler func(UsageAlert)

// CheckUsageAlerts inspects an EntitlementResult after RecordUsage and fires
// registered AlertHandlers if a threshold is newly crossed.
// Called internally by Tenant.RecordUsage — not usually called directly.
//
//   ten.CheckUsageAlerts(ws, "pages", result)
func (t *Tenant) CheckUsageAlerts(ws *Workspace, featureCode string, result EntitlementResult)
```

---

## 12. Errors

```go
// Sentinel errors. All constructed via core.E("tenant", message, nil).
var (
    // ErrNoWorkspaceContext is returned when context carries no workspace.
    ErrNoWorkspaceContext = core.E("tenant", "no workspace in context", nil)

    // ErrNoUserContext is returned when context carries no user.
    ErrNoUserContext = core.E("tenant", "no user in context", nil)

    // ErrWorkspaceNotFound is returned when slug or UUID does not resolve.
    ErrWorkspaceNotFound = core.E("tenant", "workspace not found", nil)

    // ErrFeatureNotFound is returned when the feature code is unknown.
    ErrFeatureNotFound = core.E("tenant", "feature not found", nil)

    // ErrEntitlementDenied is returned by EntitlementResult.AsError() on denial.
    ErrEntitlementDenied = core.E("tenant", "entitlement denied", nil)

    // ErrClientTimeout is returned when the PHP API does not respond in time.
    ErrClientTimeout = core.E("tenant", "api client timeout", nil)
)
```

---

## 13. Test Patterns

File: `tenant_test.go`. Three categories (Good/Bad/Ugly) mandatory per function.

```go
// TestWorkspace_Can_Good verifies allowed result when under limit.
func TestWorkspace_Can_Good(t *testing.T) {
    st, _ := store.New(":memory:")
    cache := tenant.NewTenantCache(st)
    _ = cache.SetPackages(wsUUID, []tenant.Package{{
        Code:     "starter",
        Features: []tenant.PackageFeature{{FeatureCode: "pages", LimitValue: intPtr(10)}},
    }})
    _ = cache.SetUsage(wsUUID, "pages", 3)
    svc := tenant.NewLocalEntitlementService(cache, nil)
    result := svc.Can(context.Background(), ws, "pages", 1)
    if result.IsDenied() {
        t.Fatalf("expected allowed, got denied: %s", result.Reason)
    }
    if *result.Used != 3 {
        t.Errorf("used: want 3, got %d", *result.Used)
    }
}

// TestWorkspace_Can_Bad verifies denial when at limit.
func TestWorkspace_Can_Bad(t *testing.T) {
    _ = cache.SetUsage(wsUUID, "pages", 10)
    result := svc.Can(context.Background(), ws, "pages", 1)
    if result.IsAllowed() {
        t.Fatal("expected denied at limit")
    }
    if result.Reason == "" {
        t.Error("denial reason must not be empty")
    }
}

// TestWorkspace_Can_Ugly verifies denial when feature is absent from all packages.
func TestWorkspace_Can_Ugly(t *testing.T) {
    st, _ := store.New(":memory:")
    svc := tenant.NewLocalEntitlementService(tenant.NewTenantCache(st), nil)
    result := svc.Can(context.Background(), ws, "pages", 1)
    if result.IsAllowed() {
        t.Fatal("expected denied: feature not in any package")
    }
}

// TestTenantCache_SetGet_Good verifies workspace round-trip.
func TestTenantCache_SetGet_Good(t *testing.T)

// TestTenantCache_SetGet_Bad verifies cache miss returns nil,false.
func TestTenantCache_SetGet_Bad(t *testing.T)

// TestTenantCache_SetGet_Ugly verifies InvalidateWorkspace drops all entries.
func TestTenantCache_SetGet_Ugly(t *testing.T)

// TestBoost_IsUsable_Good verifies active non-expired boost is usable.
func TestBoost_IsUsable_Good(t *testing.T)

// TestBoost_IsUsable_Bad verifies expired boost is not usable.
func TestBoost_IsUsable_Bad(t *testing.T)

// TestBoost_IsUsable_Ugly verifies exhausted boost is not usable.
func TestBoost_IsUsable_Ugly(t *testing.T)

// TestEntitlementResult_AsError_Good verifies allowed returns nil.
func TestEntitlementResult_AsError_Good(t *testing.T)

// TestEntitlementResult_AsError_Bad verifies denied returns ErrEntitlementDenied.
func TestEntitlementResult_AsError_Bad(t *testing.T)

// TestEntitlementResult_AsError_Ugly verifies unlimited returns nil.
func TestEntitlementResult_AsError_Ugly(t *testing.T)

// TestFeature_PoolCode_Good verifies root feature returns own code.
func TestFeature_PoolCode_Good(t *testing.T)

// TestFeature_PoolCode_Bad verifies child feature returns parent code.
func TestFeature_PoolCode_Bad(t *testing.T)

// TestFeature_PoolCode_Ugly verifies missing parent code falls back to own code.
func TestFeature_PoolCode_Ugly(t *testing.T)
```

---

## 14. Data Flow

### 14.1 Entitlement Check (cache-hit path)

```
HTTP request arrives
  → WorkspaceScope.Middleware()
      → read X-Workspace-Slug header: "acme"
      → TenantCache.GetWorkspaceBySlug("acme") → hit → *Workspace
      → tenant.WithWorkspace(ctx, ws)
  → handler calls ten.Can(ctx, ws, "pages", 1)
      → EntitlementService.Can()
          → TenantCache.GetPackages(ws.UUID) → hit → []Package
          → TenantCache.GetBoosts(ws.UUID) → hit → []Boost
          → TenantCache.GetUsage(ws.UUID, "pages") → hit → 3
          → cascade: total_limit=10, used=3, 3+1 <= 10
      → return EntitlementResult{Allowed: true, Limit: &10, Used: &3, Remaining: &7}
```

### 14.2 Entitlement Check (cache-miss path)

```
  → TenantCache.GetPackages(ws.UUID) → miss
  → TenantClient.GetPackagesForWorkspace(ctx, ws.UUID) → []Package from PHP API
  → TenantCache.SetPackages(ws.UUID, pkgs)   [TTL: 5 min]
  → (same pattern for boosts, usage)
  → cascade evaluation → EntitlementResult
```

### 14.3 Usage Recording

```
Handler successfully creates a page
  → ten.RecordUsage(ctx, ws, "pages", 1, &userID, {"page_id": 42})
      → TenantClient.RecordUsage(ctx, ws.UUID, "pages", 1, ...)
          → POST /api/v1/workspaces/{uuid}/usage → 200 OK
      → TenantCache.InvalidateWorkspace(ws.UUID)
      → ten.CheckUsageAlerts(ws, "pages", result)
          → if result.IsNearLimit() → fire all AlertHandlers
```

### 14.4 Agent Identity (core/mcp consumer)

```
MCP tool call arrives with X-Workspace-Slug header
  → WorkspaceScope.ScopeFunc(ctx, "acme", func(ctx context.Context) error {
        ws, _ := tenant.WorkspaceFromCtx(ctx)
        if err := ten.Can(ctx, ws, "api_calls", 1).AsError(); err != nil {
            return err
        }
        // execute tool
        return ten.RecordUsage(ctx, ws, "api_calls", 1, nil, nil)
    })
```

---

## 15. Implementation Sequence

### Phase 1 — Core types and errors
- [ ] `errors.go` — sentinel errors via `core.E()`
- [ ] `workspace.go` — `Workspace`, `WorkspaceFromCtx`, `WithWorkspace`
- [ ] `user.go` — `User`, `UserTier`, `UserFromCtx`, `WithUser`
- [ ] `feature.go` — `Feature`, `FeatureType`, `ResetType`, `PoolCode()`
- [ ] `package.go` — `Package`, `PackageFeature`, `GetFeatureLimit()`
- [ ] `boost.go` — `Boost`, `BoostType`, `BoostStatus`, `IsUsable()`, `Remaining()`
- [ ] `usage.go` — `UsageRecord`, `UsageSummaryItem`

### Phase 2 — EntitlementResult and cache
- [ ] `entitlement.go` — `EntitlementResult`, constructors, `AsError()`
- [ ] `cache.go` — `TenantCache`, all `Set`/`Get`/`Invalidate` methods, TTL constants

### Phase 3 — API client
- [ ] `client.go` — `TenantClient`, `ClientOption`, all endpoint methods

### Phase 4 — Entitlement engine
- [ ] `entitlement.go` — `EntitlementService` interface
- [ ] `entitlement.go` — `localEntitlementService` struct with cascade logic
- [ ] `NewLocalEntitlementService(cache, client)` — client may be nil for unit tests

### Phase 5 — Tenant service and scope middleware
- [ ] `scope.go` — `WorkspaceScope`, `Middleware()`, `RequireWorkspace()`, `ScopeFunc()`
- [ ] `alert.go` — `UsageAlert`, `AlertHandler`, `CheckUsageAlerts()`
- [ ] `tenant.go` — `Tenant`, `TenantOptions`, `Register`, all passthrough methods

### Phase 6 — Tests
- [ ] `tenant_test.go` — Good/Bad/Ugly for every exported function

---

## 16. Reference

| Resource | Location |
|----------|----------|
| Core Go RFC | `code/core/go/RFC.md` |
| go-store RFC | `code/core/go/store/RFC.md` |
| go-api RFC | `code/core/go/api/RFC.md` |
| PHP EntitlementService | `core-tenant/Services/EntitlementService.php` |
| PHP EntitlementResult | `core-tenant/Services/EntitlementResult.php` |
| PHP WorkspaceManager | `core-tenant/Services/WorkspaceManager.php` |
| PHP models | `core-tenant/Models/{Workspace,Feature,Package,Boost,UsageRecord}.php` |
| PHP middleware | `core-tenant/Middleware/RequireWorkspaceContext.php` |
