// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"errors"
	"sync"
	"time"
)

// TTL constants matching PHP's cache configuration.
const (
	TTLEntitlements = 5 * time.Minute // packages, boosts, feature definitions
	TTLUsage        = 60 * time.Second
	TTLWorkspace    = 5 * time.Minute
	TTLUser         = 5 * time.Minute
)

type cacheEntry[T any] struct {
	value     T
	expiresAt time.Time
}

func (e cacheEntry[T]) expired(now time.Time) bool {
	return !e.expiresAt.IsZero() && now.After(e.expiresAt)
}

// TenantCache provides workspace-scoped caching over an in-memory store.
// The external go-store dependency is not available in this checkout, so the
// cache keeps the RFC behaviour locally with the same TTL semantics.
type TenantCache struct {
	store any

	mu               sync.RWMutex
	workspacesByUUID map[string]cacheEntry[*Workspace]
	workspaceIDs     map[int64]string
	workspaceSlugs   map[string]string
	packages         map[string]cacheEntry[[]Package]
	boosts           map[string]cacheEntry[[]Boost]
	usage            map[string]cacheEntry[int]
	features         map[string]cacheEntry[*Feature]
}

// NewTenantCache creates a new cache backed by the given store handle.
// The store handle is retained for compatibility but not used directly.
func NewTenantCache(st any) *TenantCache {
	return &TenantCache{
		store:            st,
		workspacesByUUID: map[string]cacheEntry[*Workspace]{},
		workspaceIDs:     map[int64]string{},
		workspaceSlugs:   map[string]string{},
		packages:         map[string]cacheEntry[[]Package]{},
		boosts:           map[string]cacheEntry[[]Boost]{},
		usage:            map[string]cacheEntry[int]{},
		features:         map[string]cacheEntry[*Feature]{},
	}
}

func (c *TenantCache) now() time.Time {
	return time.Now()
}

func clonePackages(packages []Package) []Package {
	if packages == nil {
		return nil
	}
	clone := make([]Package, len(packages))
	copy(clone, packages)
	return clone
}

func cloneBoosts(boosts []Boost) []Boost {
	if boosts == nil {
		return nil
	}
	clone := make([]Boost, len(boosts))
	copy(clone, boosts)
	return clone
}

func cloneFeature(feature *Feature) *Feature {
	if feature == nil {
		return nil
	}
	clone := *feature
	return &clone
}

// SetWorkspace stores the workspace record.
func (c *TenantCache) SetWorkspace(ws *Workspace) error {
	if ws == nil {
		return ErrNoWorkspaceContext
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	clone := cloneWorkspace(ws)
	c.workspacesByUUID[ws.UUID] = cacheEntry[*Workspace]{value: clone, expiresAt: c.now().Add(TTLWorkspace)}
	if ws.ID != 0 {
		c.workspaceIDs[ws.ID] = ws.UUID
	}
	if ws.Slug != "" {
		c.workspaceSlugs[ws.Slug] = ws.UUID
	}
	return nil
}

// GetWorkspace retrieves a cached workspace by UUID. Returns nil, false on miss.
func (c *TenantCache) GetWorkspace(uuid string) (*Workspace, bool) {
	c.mu.RLock()
	entry, ok := c.workspacesByUUID[uuid]
	c.mu.RUnlock()
	if !ok || entry.expired(c.now()) {
		if ok {
			c.mu.Lock()
			delete(c.workspacesByUUID, uuid)
			c.mu.Unlock()
		}
		return nil, false
	}
	return cloneWorkspace(entry.value), true
}

// GetWorkspaceByID retrieves a cached workspace by integer ID.
func (c *TenantCache) GetWorkspaceByID(id int64) (*Workspace, bool) {
	c.mu.RLock()
	uuid, ok := c.workspaceIDs[id]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	return c.GetWorkspace(uuid)
}

// GetWorkspaceBySlug retrieves a cached workspace by slug via UUID indirection.
func (c *TenantCache) GetWorkspaceBySlug(slug string) (*Workspace, bool) {
	c.mu.RLock()
	uuid, ok := c.workspaceSlugs[slug]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	return c.GetWorkspace(uuid)
}

// SetPackages stores the active package list for a workspace.
func (c *TenantCache) SetPackages(wsUUID string, packages []Package) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.packages[wsUUID] = cacheEntry[[]Package]{value: clonePackages(packages), expiresAt: c.now().Add(TTLEntitlements)}
	return nil
}

// GetPackages retrieves cached packages. Returns nil, false on miss.
func (c *TenantCache) GetPackages(wsUUID string) ([]Package, bool) {
	c.mu.RLock()
	entry, ok := c.packages[wsUUID]
	c.mu.RUnlock()
	if !ok || entry.expired(c.now()) {
		if ok {
			c.mu.Lock()
			delete(c.packages, wsUUID)
			c.mu.Unlock()
		}
		return nil, false
	}
	return clonePackages(entry.value), true
}

// SetBoosts stores the active boost list for a workspace.
func (c *TenantCache) SetBoosts(wsUUID string, boosts []Boost) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.boosts[wsUUID] = cacheEntry[[]Boost]{value: cloneBoosts(boosts), expiresAt: c.now().Add(TTLEntitlements)}
	return nil
}

// GetBoosts retrieves cached boosts. Returns nil, false on miss.
func (c *TenantCache) GetBoosts(wsUUID string) ([]Boost, bool) {
	c.mu.RLock()
	entry, ok := c.boosts[wsUUID]
	c.mu.RUnlock()
	if !ok || entry.expired(c.now()) {
		if ok {
			c.mu.Lock()
			delete(c.boosts, wsUUID)
			c.mu.Unlock()
		}
		return nil, false
	}
	return cloneBoosts(entry.value), true
}

// SetUsage stores the current usage count for a workspace+feature.
func (c *TenantCache) SetUsage(wsUUID, featureCode string, count int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.usage[usageCacheKey(wsUUID, featureCode)] = cacheEntry[int]{value: count, expiresAt: c.now().Add(TTLUsage)}
	return nil
}

// GetUsage retrieves a cached usage count. Returns 0, false on miss.
func (c *TenantCache) GetUsage(wsUUID, featureCode string) (int, bool) {
	c.mu.RLock()
	entry, ok := c.usage[usageCacheKey(wsUUID, featureCode)]
	c.mu.RUnlock()
	if !ok || entry.expired(c.now()) {
		if ok {
			c.mu.Lock()
			delete(c.usage, usageCacheKey(wsUUID, featureCode))
			c.mu.Unlock()
		}
		return 0, false
	}
	return entry.value, true
}

// invalidateUsage drops the cached usage counter for a workspace+feature pair.
func (c *TenantCache) invalidateUsage(wsUUID, featureCode string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.usage, usageCacheKey(wsUUID, featureCode))
}

// InvalidateWorkspace drops all cache entries for this workspace UUID.
func (c *TenantCache) InvalidateWorkspace(wsUUID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.workspacesByUUID, wsUUID)
	delete(c.packages, wsUUID)
	delete(c.boosts, wsUUID)

	for key := range c.usage {
		if hasUsagePrefix(key, wsUUID) {
			delete(c.usage, key)
		}
	}
	for id, uuid := range c.workspaceIDs {
		if uuid == wsUUID {
			delete(c.workspaceIDs, id)
		}
	}
	for slug, uuid := range c.workspaceSlugs {
		if uuid == wsUUID {
			delete(c.workspaceSlugs, slug)
		}
	}
	return nil
}

// SetFeature stores a feature definition by code. Features are global.
func (c *TenantCache) SetFeature(feature *Feature) error {
	if feature == nil {
		return ErrFeatureNotFound
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.features[feature.Code] = cacheEntry[*Feature]{value: cloneFeature(feature), expiresAt: c.now().Add(TTLEntitlements)}
	return nil
}

// GetFeature retrieves a cached feature definition by code.
func (c *TenantCache) GetFeature(code string) (*Feature, bool) {
	c.mu.RLock()
	entry, ok := c.features[code]
	c.mu.RUnlock()
	if !ok || entry.expired(c.now()) {
		if ok {
			c.mu.Lock()
			delete(c.features, code)
			c.mu.Unlock()
		}
		return nil, false
	}
	return cloneFeature(entry.value), true
}

func usageCacheKey(wsUUID, featureCode string) string {
	return wsUUID + "\x00" + featureCode
}

func hasUsagePrefix(key, wsUUID string) bool {
	if len(key) < len(wsUUID)+1 {
		return false
	}
	return key[:len(wsUUID)] == wsUUID && key[len(wsUUID)] == '\x00'
}

var errCacheMiss = errors.New("cache miss")
