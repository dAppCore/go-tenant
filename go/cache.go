// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"strconv"
	"sync"
	"time"

	"dappco.re/go"
	"dappco.re/go/store"
)

// TTL constants matching PHP's cache configuration.
const (
	TTLEntitlements = 5 * time.Minute // packages, boosts, feature definitions
	TTLUsage        = 60 * time.Second
	TTLWorkspace    = 5 * time.Minute
	TTLUser         = 5 * time.Minute
)

type cacheEntry struct {
	value     any
	expiresAt time.Time
}

func (e cacheEntry) expired(now time.Time) bool {
	return !e.expiresAt.IsZero() && now.After(e.expiresAt)
}

// TenantCache provides workspace-scoped caching over go-store with an in-memory
// hot path. The store is optional, but when present it persists the same TTL
// semantics as the RFC cache contract.
//
//	cache := tenant.NewTenantCache(storeHandle)
//	cache.SetWorkspace(&tenant.Workspace{UUID: "ws-7", Slug: "acme"})
type TenantCache struct {
	store *store.Store

	lock             sync.RWMutex
	workspacesByUUID map[string]cacheEntry
	workspaceIDs     map[int64]string
	workspaceSlugs   map[string]string
	packages         map[string]cacheEntry
	boosts           map[string]cacheEntry
	usage            map[string]cacheEntry
	features         map[string]cacheEntry
	users            map[string]cacheEntry
}

// NewTenantCache creates a new cache backed by the given store handle.
//
//	storeHandle, _ := store.New(":memory:")
//	cache := tenant.NewTenantCache(storeHandle)
func NewTenantCache(st *store.Store) *TenantCache {
	return &TenantCache{
		store:            st,
		workspacesByUUID: map[string]cacheEntry{},
		workspaceIDs:     map[int64]string{},
		workspaceSlugs:   map[string]string{},
		packages:         map[string]cacheEntry{},
		boosts:           map[string]cacheEntry{},
		usage:            map[string]cacheEntry{},
		features:         map[string]cacheEntry{},
		users:            map[string]cacheEntry{},
	}
}

// persistJSON writes a JSON-serialised value to go-store with the given TTL.
//
//	c.persistJSON("ws:uuid-7:record", "data", workspace, TTLWorkspace)
func (c *TenantCache) persistJSON(group, key string, value any, ttl time.Duration) core.Result {
	if c == nil || c.store == nil {
		return core.Ok(nil)
	}
	return core.ResultOf(nil, c.store.SetWithTTL(group, key, core.JSONMarshalString(value), ttl))
}

// persistString writes a plain string value to go-store with the given TTL.
//
//	c.persistString("ws:slug:acme", "uuid", "uuid-7", TTLWorkspace)
func (c *TenantCache) persistString(group, key, value string, ttl time.Duration) core.Result {
	if c == nil || c.store == nil {
		return core.Ok(nil)
	}
	return core.ResultOf(nil, c.store.SetWithTTL(group, key, value, ttl))
}

// readJSON reads and deserialises a JSON value from go-store.
//
//	var workspace Workspace
//	ok := c.readJSON("ws:uuid-7:record", "data", &workspace)
func (c *TenantCache) readJSON(group, key string, target any) bool {
	if c == nil || c.store == nil {
		return false
	}
	value, result := c.store.Get(group, key)
	if !result.OK {
		return false
	}
	return core.JSONUnmarshalString(value, target).OK
}

// readString reads a plain string value from go-store.
//
//	uuid, ok := c.readString("ws:slug:acme", "uuid")
func (c *TenantCache) readString(group, key string) (string, bool) {
	if c == nil || c.store == nil {
		return "", false
	}
	value, result := c.store.Get(group, key)
	if !result.OK {
		return "", false
	}
	return value, true
}

// deleteStoreGroup removes all entries in a go-store group.
//
//	c.deleteStoreGroup("ws:uuid-7:record")
func (c *TenantCache) deleteStoreGroup(group string) core.Result {
	if c == nil || c.store == nil {
		return core.Ok(nil)
	}
	return core.ResultOf(nil, c.store.DeleteGroup(group))
}

// deleteStorePrefix removes all entries whose group starts with the given prefix.
//
//	c.deleteStorePrefix("ws:uuid-7:usage:")
func (c *TenantCache) deleteStorePrefix(prefix string) core.Result {
	if c == nil || c.store == nil {
		return core.Ok(nil)
	}
	return core.ResultOf(nil, c.store.DeletePrefix(prefix))
}

func (c *TenantCache) now() time.Time {
	return time.Now()
}

// clonePackages returns a shallow copy of the package slice to prevent cache mutation.
//
//	safe := clonePackages(cachedPackages)
func clonePackages(packages []Package) []Package {
	if packages == nil {
		return nil
	}
	clone := make([]Package, len(packages))
	copy(clone, packages)
	return clone
}

// cloneBoosts returns a shallow copy of the boost slice to prevent cache mutation.
//
//	safe := cloneBoosts(cachedBoosts)
func cloneBoosts(boosts []Boost) []Boost {
	if boosts == nil {
		return nil
	}
	clone := make([]Boost, len(boosts))
	copy(clone, boosts)
	return clone
}

// cloneFeature returns a copy of the feature to prevent cache mutation.
//
//	safe := cloneFeature(cachedFeature)
func cloneFeature(feature *Feature) *Feature {
	if feature == nil {
		return nil
	}
	clone := *feature
	return &clone
}

// cloneUser returns a copy of the user to prevent cache mutation.
//
//	safe := cloneUser(cachedUser)
func cloneUser(user *User) *User {
	if user == nil {
		return nil
	}
	clone := *user
	return &clone
}

// SetWorkspace stores the workspace record.
//
//	cache.SetWorkspace(&tenant.Workspace{ID: 7, UUID: "uuid-7", Slug: "acme"})
func (c *TenantCache) SetWorkspace(ws *Workspace) core.Result {
	if c == nil {
		return core.Fail(ErrNoWorkspaceContext)
	}
	if ws == nil {
		return core.Fail(ErrNoWorkspaceContext)
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	for id, uuid := range c.workspaceIDs {
		if uuid == ws.UUID {
			delete(c.workspaceIDs, id)
			if r := c.deleteStoreGroup(workspaceIDGroup(id)); !r.OK {
				return r
			}
		}
	}
	for slug, uuid := range c.workspaceSlugs {
		if uuid == ws.UUID {
			delete(c.workspaceSlugs, slug)
			if r := c.deleteStoreGroup(workspaceSlugGroup(slug)); !r.OK {
				return r
			}
		}
	}

	clone := cloneWorkspace(ws)
	c.workspacesByUUID[ws.UUID] = cacheEntry{value: clone, expiresAt: c.now().Add(TTLWorkspace)}
	if ws.ID != 0 {
		c.workspaceIDs[ws.ID] = ws.UUID
		if r := c.persistString(workspaceIDGroup(ws.ID), "uuid", ws.UUID, TTLWorkspace); !r.OK {
			return r
		}
	}
	if ws.Slug != "" {
		c.workspaceSlugs[ws.Slug] = ws.UUID
		if r := c.persistString(workspaceSlugGroup(ws.Slug), "uuid", ws.UUID, TTLWorkspace); !r.OK {
			return r
		}
	}
	if r := c.persistJSON(workspaceRecordGroup(ws.UUID), "data", clone, TTLWorkspace); !r.OK {
		return r
	}
	return core.Ok(nil)
}

// GetWorkspace retrieves a cached workspace by UUID. Returns nil, false on miss.
//
//	workspace, ok := cache.GetWorkspace("uuid-7")
func (c *TenantCache) GetWorkspace(uuid string) (*Workspace, bool) {
	if c == nil {
		return nil, false
	}
	c.lock.RLock()
	entry, ok := c.workspacesByUUID[uuid]
	c.lock.RUnlock()
	if !ok || entry.expired(c.now()) {
		if ok {
			c.lock.Lock()
			delete(c.workspacesByUUID, uuid)
			c.lock.Unlock()
		}
		var workspace Workspace
		if c.readJSON(workspaceRecordGroup(uuid), "data", &workspace) {
			if r := c.SetWorkspace(&workspace); !r.OK {
				return nil, false
			}
			return cloneWorkspace(&workspace), true
		}
		return nil, false
	}
	workspace, _ := entry.value.(*Workspace)
	return cloneWorkspace(workspace), true
}

// GetWorkspaceByID retrieves a cached workspace by integer ID.
//
//	workspace, ok := cache.GetWorkspaceByID(7)
func (c *TenantCache) GetWorkspaceByID(id int64) (*Workspace, bool) {
	if c == nil {
		return nil, false
	}
	c.lock.RLock()
	uuid, ok := c.workspaceIDs[id]
	c.lock.RUnlock()
	if !ok {
		if value, ok := c.readString(workspaceIDGroup(id), "uuid"); ok {
			return c.GetWorkspace(value)
		}
		return nil, false
	}
	return c.GetWorkspace(uuid)
}

// GetWorkspaceBySlug retrieves a cached workspace by slug via UUID indirection.
//
//	workspace, ok := cache.GetWorkspaceBySlug("acme")
func (c *TenantCache) GetWorkspaceBySlug(slug string) (*Workspace, bool) {
	if c == nil {
		return nil, false
	}
	c.lock.RLock()
	uuid, ok := c.workspaceSlugs[slug]
	c.lock.RUnlock()
	if !ok {
		if value, ok := c.readString(workspaceSlugGroup(slug), "uuid"); ok {
			return c.GetWorkspace(value)
		}
		return nil, false
	}
	return c.GetWorkspace(uuid)
}

// SetPackages stores the active package list for a workspace.
//
//	cache.SetPackages(ws.UUID, []tenant.Package{{Code: "starter"}})
func (c *TenantCache) SetPackages(wsUUID string, packages []Package) core.Result {
	if c == nil {
		return core.Fail(ErrNoWorkspaceContext)
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	c.packages[wsUUID] = cacheEntry{value: clonePackages(packages), expiresAt: c.now().Add(TTLEntitlements)}
	if r := c.persistJSON(packagesGroup(wsUUID), "data", packages, TTLEntitlements); !r.OK {
		return r
	}
	return core.Ok(nil)
}

// GetPackages retrieves cached packages. Returns nil, false on miss.
//
//	packages, ok := cache.GetPackages(ws.UUID)
func (c *TenantCache) GetPackages(wsUUID string) ([]Package, bool) {
	if c == nil {
		return nil, false
	}
	c.lock.RLock()
	entry, ok := c.packages[wsUUID]
	c.lock.RUnlock()
	if !ok || entry.expired(c.now()) {
		if ok {
			c.lock.Lock()
			delete(c.packages, wsUUID)
			c.lock.Unlock()
		}
		var packages []Package
		if c.readJSON(packagesGroup(wsUUID), "data", &packages) {
			c.lock.Lock()
			c.packages[wsUUID] = cacheEntry{value: clonePackages(packages), expiresAt: c.now().Add(TTLEntitlements)}
			c.lock.Unlock()
			return clonePackages(packages), true
		}
		return nil, false
	}
	packages, _ := entry.value.([]Package)
	return clonePackages(packages), true
}

// SetBoosts stores the active boost list for a workspace.
//
//	cache.SetBoosts(ws.UUID, []tenant.Boost{{FeatureCode: "pages"}})
func (c *TenantCache) SetBoosts(wsUUID string, boosts []Boost) core.Result {
	if c == nil {
		return core.Fail(ErrNoWorkspaceContext)
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	c.boosts[wsUUID] = cacheEntry{value: cloneBoosts(boosts), expiresAt: c.now().Add(TTLEntitlements)}
	if r := c.persistJSON(boostsGroup(wsUUID), "data", boosts, TTLEntitlements); !r.OK {
		return r
	}
	return core.Ok(nil)
}

// GetBoosts retrieves cached boosts. Returns nil, false on miss.
//
//	boosts, ok := cache.GetBoosts(ws.UUID)
func (c *TenantCache) GetBoosts(wsUUID string) ([]Boost, bool) {
	if c == nil {
		return nil, false
	}
	c.lock.RLock()
	entry, ok := c.boosts[wsUUID]
	c.lock.RUnlock()
	if !ok || entry.expired(c.now()) {
		if ok {
			c.lock.Lock()
			delete(c.boosts, wsUUID)
			c.lock.Unlock()
		}
		var boosts []Boost
		if c.readJSON(boostsGroup(wsUUID), "data", &boosts) {
			c.lock.Lock()
			c.boosts[wsUUID] = cacheEntry{value: cloneBoosts(boosts), expiresAt: c.now().Add(TTLEntitlements)}
			c.lock.Unlock()
			return cloneBoosts(boosts), true
		}
		return nil, false
	}
	boosts, _ := entry.value.([]Boost)
	return cloneBoosts(boosts), true
}

// SetUsage stores the current usage count for a workspace+feature.
//
//	cache.SetUsage(ws.UUID, "pages", 7)
func (c *TenantCache) SetUsage(wsUUID, featureCode string, count int) core.Result {
	if c == nil {
		return core.Fail(ErrNoWorkspaceContext)
	}
	featureCode = normalizedFeatureCode(featureCode)
	c.lock.Lock()
	defer c.lock.Unlock()
	c.usage[usageCacheKey(wsUUID, featureCode)] = cacheEntry{value: count, expiresAt: c.now().Add(TTLUsage)}
	if r := c.persistString(usageGroup(wsUUID, featureCode), "count", strconv.Itoa(count), TTLUsage); !r.OK {
		return r
	}
	return core.Ok(nil)
}

// GetUsage retrieves a cached usage count. Returns 0, false on miss.
//
//	used, ok := cache.GetUsage(ws.UUID, "pages")
func (c *TenantCache) GetUsage(wsUUID, featureCode string) (int, bool) {
	if c == nil {
		return 0, false
	}
	featureCode = normalizedFeatureCode(featureCode)
	c.lock.RLock()
	entry, ok := c.usage[usageCacheKey(wsUUID, featureCode)]
	c.lock.RUnlock()
	if !ok || entry.expired(c.now()) {
		if ok {
			c.lock.Lock()
			delete(c.usage, usageCacheKey(wsUUID, featureCode))
			c.lock.Unlock()
		}
		if value, ok := c.readString(usageGroup(wsUUID, featureCode), "count"); ok {
			if count, err := strconv.Atoi(value); err == nil {
				c.lock.Lock()
				c.usage[usageCacheKey(wsUUID, featureCode)] = cacheEntry{value: count, expiresAt: c.now().Add(TTLUsage)}
				c.lock.Unlock()
				return count, true
			}
		}
		return 0, false
	}
	count, _ := entry.value.(int)
	return count, true
}

// invalidateUsage drops the cached usage counter for a workspace+feature pair.
//
//	c.invalidateUsage(ws.UUID, "pages")
func (c *TenantCache) invalidateUsage(wsUUID, featureCode string) core.Result {
	featureCode = normalizedFeatureCode(featureCode)
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.usage, usageCacheKey(wsUUID, featureCode))
	return c.deleteStoreGroup(usageGroup(wsUUID, featureCode))
}

// InvalidateWorkspace drops all cache entries for this workspace UUID.
//
//	cache.InvalidateWorkspace(ws.UUID)
func (c *TenantCache) InvalidateWorkspace(wsUUID string) core.Result {
	if c == nil {
		return core.Ok(nil)
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	delete(c.workspacesByUUID, wsUUID)
	delete(c.packages, wsUUID)
	delete(c.boosts, wsUUID)
	first := core.Ok(nil)
	if r := c.deleteStoreGroup(workspaceRecordGroup(wsUUID)); !r.OK && first.OK {
		first = r
	}
	if r := c.deleteStoreGroup(packagesGroup(wsUUID)); !r.OK && first.OK {
		first = r
	}
	if r := c.deleteStoreGroup(boostsGroup(wsUUID)); !r.OK && first.OK {
		first = r
	}
	if r := c.deleteStorePrefix(usagePrefix(wsUUID)); !r.OK && first.OK {
		first = r
	}

	for key := range c.usage {
		if hasUsagePrefix(key, wsUUID) {
			delete(c.usage, key)
		}
	}
	for id, uuid := range c.workspaceIDs {
		if uuid == wsUUID {
			delete(c.workspaceIDs, id)
			if r := c.deleteStoreGroup(workspaceIDGroup(id)); !r.OK && first.OK {
				first = r
			}
		}
	}
	for slug, uuid := range c.workspaceSlugs {
		if uuid == wsUUID {
			delete(c.workspaceSlugs, slug)
			if r := c.deleteStoreGroup(workspaceSlugGroup(slug)); !r.OK && first.OK {
				first = r
			}
		}
	}
	return first
}

// SetFeature stores a feature definition by code. Features are global.
//
//	cache.SetFeature(&tenant.Feature{Code: "pages", Type: tenant.FeatureTypeLimit})
func (c *TenantCache) SetFeature(feature *Feature) core.Result {
	if c == nil {
		return core.Fail(ErrFeatureNotFound)
	}
	if feature == nil {
		return core.Fail(ErrFeatureNotFound)
	}
	featureCode := normalizedFeatureCode(feature.Code)
	c.lock.Lock()
	defer c.lock.Unlock()
	c.features[featureCode] = cacheEntry{value: cloneFeature(feature), expiresAt: c.now().Add(TTLEntitlements)}
	if r := c.persistJSON(featureGroup(featureCode), "data", feature, TTLEntitlements); !r.OK {
		return r
	}
	return core.Ok(nil)
}

// GetFeature retrieves a cached feature definition by code.
//
//	feature, ok := cache.GetFeature("pages")
func (c *TenantCache) GetFeature(code string) (*Feature, bool) {
	if c == nil {
		return nil, false
	}
	code = normalizedFeatureCode(code)
	c.lock.RLock()
	entry, ok := c.features[code]
	c.lock.RUnlock()
	if !ok || entry.expired(c.now()) {
		if ok {
			c.lock.Lock()
			delete(c.features, code)
			c.lock.Unlock()
		}
		var feature Feature
		if c.readJSON(featureGroup(code), "data", &feature) {
			c.lock.Lock()
			c.features[code] = cacheEntry{value: cloneFeature(&feature), expiresAt: c.now().Add(TTLEntitlements)}
			c.lock.Unlock()
			return cloneFeature(&feature), true
		}
		return nil, false
	}
	feature, _ := entry.value.(*Feature)
	return cloneFeature(feature), true
}

// SetUser stores the authenticated user record.
//
//	cache.SetUser(&tenant.User{UUID: "user-7", Email: "ada@example.uk"})
func (c *TenantCache) SetUser(user *User) core.Result {
	if c == nil {
		return core.Fail(ErrNoUserContext)
	}
	if user == nil {
		return core.Fail(ErrNoUserContext)
	}
	if user.UUID == "" {
		return core.Fail(ErrNoUserContext)
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	c.users[user.UUID] = cacheEntry{value: cloneUser(user), expiresAt: c.now().Add(TTLUser)}
	if r := c.persistJSON(userGroup(user.UUID), "data", user, TTLUser); !r.OK {
		return r
	}
	return core.Ok(nil)
}

// GetUser retrieves a cached user by UUID. Returns nil, false on miss.
//
//	user, ok := cache.GetUser("user-7")
func (c *TenantCache) GetUser(uuid string) (*User, bool) {
	if c == nil {
		return nil, false
	}
	c.lock.RLock()
	entry, ok := c.users[uuid]
	c.lock.RUnlock()
	if !ok || entry.expired(c.now()) {
		if ok {
			c.lock.Lock()
			delete(c.users, uuid)
			c.lock.Unlock()
		}
		var user User
		if c.readJSON(userGroup(uuid), "data", &user) {
			c.lock.Lock()
			c.users[uuid] = cacheEntry{value: cloneUser(&user), expiresAt: c.now().Add(TTLUser)}
			c.lock.Unlock()
			return cloneUser(&user), true
		}
		return nil, false
	}
	user, _ := entry.value.(*User)
	return cloneUser(user), true
}

// usageCacheKey builds the in-memory cache key for a workspace+feature usage counter.
//
//	usageCacheKey("uuid-7", "pages")  // "uuid-7\x00pages"
func usageCacheKey(wsUUID, featureCode string) string {
	return wsUUID + "\x00" + normalizedFeatureCode(featureCode)
}

// hasUsagePrefix checks whether a cache key belongs to the given workspace UUID.
//
//	hasUsagePrefix("uuid-7\x00pages", "uuid-7")  // true
//	hasUsagePrefix("uuid-9\x00pages", "uuid-7")  // false
func hasUsagePrefix(key, wsUUID string) bool {
	if len(key) < len(wsUUID)+1 {
		return false
	}
	return key[:len(wsUUID)] == wsUUID && key[len(wsUUID)] == '\x00'
}

// workspaceRecordGroup returns the go-store group key for a workspace record.
//
//	workspaceRecordGroup("uuid-7")  // "ws:uuid-7:record"
func workspaceRecordGroup(wsUUID string) string {
	return "ws:" + wsUUID + ":record"
}

// workspaceSlugGroup returns the go-store group key for a slug-to-UUID mapping.
//
//	workspaceSlugGroup("acme")  // "ws:slug:acme"
func workspaceSlugGroup(slug string) string {
	return "ws:slug:" + slug
}

// workspaceIDGroup returns the go-store group key for an ID-to-UUID mapping.
//
//	workspaceIDGroup(42)  // "ws:id:42"
func workspaceIDGroup(id int64) string {
	return "ws:id:" + strconv.FormatInt(id, 10)
}

// packagesGroup returns the go-store group key for a workspace's packages.
//
//	packagesGroup("uuid-7")  // "ws:uuid-7:packages"
func packagesGroup(wsUUID string) string {
	return "ws:" + wsUUID + ":packages"
}

// boostsGroup returns the go-store group key for a workspace's boosts.
//
//	boostsGroup("uuid-7")  // "ws:uuid-7:boosts"
func boostsGroup(wsUUID string) string {
	return "ws:" + wsUUID + ":boosts"
}

// usageGroup returns the go-store group key for a workspace+feature usage counter.
//
//	usageGroup("uuid-7", "pages")  // "ws:uuid-7:usage:pages"
func usageGroup(wsUUID, featureCode string) string {
	return "ws:" + wsUUID + ":usage:" + featureCode
}

// usagePrefix returns the go-store prefix for all usage entries of a workspace.
//
//	usagePrefix("uuid-7")  // "ws:uuid-7:usage:"
func usagePrefix(wsUUID string) string {
	return "ws:" + wsUUID + ":usage:"
}

// featureGroup returns the go-store group key for a feature definition.
//
//	featureGroup("pages")  // "feature:pages"
func featureGroup(code string) string {
	return "feature:" + normalizedFeatureCode(code)
}

// userGroup returns the go-store group key for a user record.
//
//	userGroup("user-7")  // "user:user-7"
func userGroup(uuid string) string {
	return "user:" + uuid
}
