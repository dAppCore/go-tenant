// SPDX-License-Identifier: EUPL-1.2

package tenant

import "time"

// TTL constants matching PHP's cache configuration.
const (
	TTLEntitlements = 5 * time.Minute  // packages, boosts, feature definitions
	TTLUsage        = 60 * time.Second // usage counters (more volatile)
	TTLWorkspace    = 5 * time.Minute  // workspace record
	TTLUser         = 5 * time.Minute  // user record
)

// TenantCache provides workspace-scoped caching over go-store.
// Groups are namespaced by workspace UUID to prevent cross-tenant leakage.
//
//	cache := tenant.NewTenantCache(store)
//	cache.SetPackages(ws.UUID, pkgs)
//	pkgs, ok := cache.GetPackages(ws.UUID)
type TenantCache struct {
	// store is the go-store instance backing this cache.
	// TODO: type will be *store.Store once go-store is wired
	store any
}

// NewTenantCache creates a new cache backed by the given go-store instance.
//
//	cache := tenant.NewTenantCache(st)
func NewTenantCache(st any) *TenantCache {
	// TODO: implement — accept *store.Store
	return &TenantCache{store: st}
}

// SetWorkspace stores the workspace record.
//
//	cache.SetWorkspace(ws)
func (c *TenantCache) SetWorkspace(ws *Workspace) error {
	// TODO: implement — group: ws:{uuid}:record, key: data, TTL: 5 min
	return nil
}

// GetWorkspace retrieves a cached workspace by UUID. Returns nil, false on miss.
//
//	ws, ok := cache.GetWorkspace("550e8400-...")
func (c *TenantCache) GetWorkspace(uuid string) (*Workspace, bool) {
	// TODO: implement
	return nil, false
}

// GetWorkspaceBySlug retrieves a cached workspace by slug via UUID indirection.
//
//	ws, ok := cache.GetWorkspaceBySlug("acme")
func (c *TenantCache) GetWorkspaceBySlug(slug string) (*Workspace, bool) {
	// TODO: implement — group: ws:slug:{slug}, key: uuid → then GetWorkspace
	return nil, false
}

// SetPackages stores the active package list for a workspace.
//
//	cache.SetPackages(ws.UUID, pkgs)
func (c *TenantCache) SetPackages(wsUUID string, pkgs []Package) error {
	// TODO: implement — group: ws:{uuid}:packages, key: data, TTL: 5 min
	return nil
}

// GetPackages retrieves cached packages. Returns nil, false on miss.
//
//	pkgs, ok := cache.GetPackages(ws.UUID)
func (c *TenantCache) GetPackages(wsUUID string) ([]Package, bool) {
	// TODO: implement
	return nil, false
}

// SetBoosts stores the active boost list for a workspace.
//
//	cache.SetBoosts(ws.UUID, boosts)
func (c *TenantCache) SetBoosts(wsUUID string, boosts []Boost) error {
	// TODO: implement — group: ws:{uuid}:boosts, key: data, TTL: 5 min
	return nil
}

// GetBoosts retrieves cached boosts. Returns nil, false on miss.
//
//	boosts, ok := cache.GetBoosts(ws.UUID)
func (c *TenantCache) GetBoosts(wsUUID string) ([]Boost, bool) {
	// TODO: implement
	return nil, false
}

// SetUsage stores the current usage count for a workspace+feature.
//
//	cache.SetUsage(ws.UUID, "pages", 7)
func (c *TenantCache) SetUsage(wsUUID, featureCode string, count int) error {
	// TODO: implement — group: ws:{uuid}:usage:{code}, key: count, TTL: 60 sec
	return nil
}

// GetUsage retrieves a cached usage count. Returns 0, false on miss.
//
//	used, ok := cache.GetUsage(ws.UUID, "pages")
func (c *TenantCache) GetUsage(wsUUID, featureCode string) (int, bool) {
	// TODO: implement
	return 0, false
}

// InvalidateWorkspace drops all cache entries for this workspace UUID.
// Called by RecordUsage and by external invalidation signals.
//
//	cache.InvalidateWorkspace(ws.UUID)
func (c *TenantCache) InvalidateWorkspace(wsUUID string) error {
	// TODO: implement — drop ws:{uuid}:* groups
	return nil
}

// SetFeature stores a feature definition by code. Features are global.
//
//	cache.SetFeature(feat)
func (c *TenantCache) SetFeature(feat *Feature) error {
	// TODO: implement — group: feature:{code}, key: data, TTL: 5 min
	return nil
}

// GetFeature retrieves a cached feature definition by code.
//
//	feat, ok := cache.GetFeature("pages")
func (c *TenantCache) GetFeature(code string) (*Feature, bool) {
	// TODO: implement
	return nil, false
}
