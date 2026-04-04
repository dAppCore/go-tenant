// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"strconv"
	"sync"
	"time"

	"dappco.re/go/core"
	"dappco.re/go/core/store"
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

func (c *TenantCache) persistJSON(group, key string, value any, ttl time.Duration) error {
	if c == nil || c.store == nil {
		return nil
	}
	return c.store.SetWithTTL(group, key, core.JSONMarshalString(value), ttl)
}

func (c *TenantCache) persistString(group, key, value string, ttl time.Duration) error {
	if c == nil || c.store == nil {
		return nil
	}
	return c.store.SetWithTTL(group, key, value, ttl)
}

func (c *TenantCache) readJSON(group, key string, target any) bool {
	if c == nil || c.store == nil {
		return false
	}
	value, err := c.store.Get(group, key)
	if err != nil {
		return false
	}
	return core.JSONUnmarshalString(value, target).OK
}

func (c *TenantCache) readString(group, key string) (string, bool) {
	if c == nil || c.store == nil {
		return "", false
	}
	value, err := c.store.Get(group, key)
	if err != nil {
		return "", false
	}
	return value, true
}

func (c *TenantCache) deleteStoreGroup(group string) {
	if c == nil || c.store == nil {
		return
	}
	_ = c.store.DeleteGroup(group)
}

func (c *TenantCache) deleteStorePrefix(prefix string) {
	if c == nil || c.store == nil {
		return
	}
	_ = c.store.DeletePrefix(prefix)
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
func (c *TenantCache) SetWorkspace(ws *Workspace) error {
	if ws == nil {
		return ErrNoWorkspaceContext
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	for id, uuid := range c.workspaceIDs {
		if uuid == ws.UUID {
			delete(c.workspaceIDs, id)
			c.deleteStoreGroup(workspaceIDGroup(id))
		}
	}
	for slug, uuid := range c.workspaceSlugs {
		if uuid == ws.UUID {
			delete(c.workspaceSlugs, slug)
			c.deleteStoreGroup(workspaceSlugGroup(slug))
		}
	}

	clone := cloneWorkspace(ws)
	c.workspacesByUUID[ws.UUID] = cacheEntry{value: clone, expiresAt: c.now().Add(TTLWorkspace)}
	if ws.ID != 0 {
		c.workspaceIDs[ws.ID] = ws.UUID
		if err := c.persistString(workspaceIDGroup(ws.ID), "uuid", ws.UUID, TTLWorkspace); err != nil {
			return err
		}
	}
	if ws.Slug != "" {
		c.workspaceSlugs[ws.Slug] = ws.UUID
		if err := c.persistString(workspaceSlugGroup(ws.Slug), "uuid", ws.UUID, TTLWorkspace); err != nil {
			return err
		}
	}
	if err := c.persistJSON(workspaceRecordGroup(ws.UUID), "data", clone, TTLWorkspace); err != nil {
		return err
	}
	return nil
}

// GetWorkspace retrieves a cached workspace by UUID. Returns nil, false on miss.
//
//	workspace, ok := cache.GetWorkspace("uuid-7")
func (c *TenantCache) GetWorkspace(uuid string) (*Workspace, bool) {
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
			_ = c.SetWorkspace(&workspace)
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
func (c *TenantCache) SetPackages(wsUUID string, packages []Package) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.packages[wsUUID] = cacheEntry{value: clonePackages(packages), expiresAt: c.now().Add(TTLEntitlements)}
	if err := c.persistJSON(packagesGroup(wsUUID), "data", packages, TTLEntitlements); err != nil {
		return err
	}
	return nil
}

// GetPackages retrieves cached packages. Returns nil, false on miss.
//
//	packages, ok := cache.GetPackages(ws.UUID)
func (c *TenantCache) GetPackages(wsUUID string) ([]Package, bool) {
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
func (c *TenantCache) SetBoosts(wsUUID string, boosts []Boost) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.boosts[wsUUID] = cacheEntry{value: cloneBoosts(boosts), expiresAt: c.now().Add(TTLEntitlements)}
	if err := c.persistJSON(boostsGroup(wsUUID), "data", boosts, TTLEntitlements); err != nil {
		return err
	}
	return nil
}

// GetBoosts retrieves cached boosts. Returns nil, false on miss.
//
//	boosts, ok := cache.GetBoosts(ws.UUID)
func (c *TenantCache) GetBoosts(wsUUID string) ([]Boost, bool) {
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
func (c *TenantCache) SetUsage(wsUUID, featureCode string, count int) error {
	featureCode = normalizedFeatureCode(featureCode)
	c.lock.Lock()
	defer c.lock.Unlock()
	c.usage[usageCacheKey(wsUUID, featureCode)] = cacheEntry{value: count, expiresAt: c.now().Add(TTLUsage)}
	if err := c.persistString(usageGroup(wsUUID, featureCode), "count", strconv.Itoa(count), TTLUsage); err != nil {
		return err
	}
	return nil
}

// GetUsage retrieves a cached usage count. Returns 0, false on miss.
//
//	used, ok := cache.GetUsage(ws.UUID, "pages")
func (c *TenantCache) GetUsage(wsUUID, featureCode string) (int, bool) {
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
func (c *TenantCache) invalidateUsage(wsUUID, featureCode string) {
	featureCode = normalizedFeatureCode(featureCode)
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.usage, usageCacheKey(wsUUID, featureCode))
	c.deleteStoreGroup(usageGroup(wsUUID, featureCode))
}

// InvalidateWorkspace drops all cache entries for this workspace UUID.
//
//	cache.InvalidateWorkspace(ws.UUID)
func (c *TenantCache) InvalidateWorkspace(wsUUID string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	delete(c.workspacesByUUID, wsUUID)
	delete(c.packages, wsUUID)
	delete(c.boosts, wsUUID)
	c.deleteStoreGroup(workspaceRecordGroup(wsUUID))
	c.deleteStoreGroup(packagesGroup(wsUUID))
	c.deleteStoreGroup(boostsGroup(wsUUID))
	c.deleteStorePrefix(usagePrefix(wsUUID))

	for key := range c.usage {
		if hasUsagePrefix(key, wsUUID) {
			delete(c.usage, key)
		}
	}
	for id, uuid := range c.workspaceIDs {
		if uuid == wsUUID {
			delete(c.workspaceIDs, id)
			c.deleteStoreGroup(workspaceIDGroup(id))
		}
	}
	for slug, uuid := range c.workspaceSlugs {
		if uuid == wsUUID {
			delete(c.workspaceSlugs, slug)
			c.deleteStoreGroup(workspaceSlugGroup(slug))
		}
	}
	return nil
}

// SetFeature stores a feature definition by code. Features are global.
//
//	cache.SetFeature(&tenant.Feature{Code: "pages", Type: tenant.FeatureTypeLimit})
func (c *TenantCache) SetFeature(feature *Feature) error {
	if feature == nil {
		return ErrFeatureNotFound
	}
	featureCode := normalizedFeatureCode(feature.Code)
	c.lock.Lock()
	defer c.lock.Unlock()
	c.features[featureCode] = cacheEntry{value: cloneFeature(feature), expiresAt: c.now().Add(TTLEntitlements)}
	if err := c.persistJSON(featureGroup(featureCode), "data", feature, TTLEntitlements); err != nil {
		return err
	}
	return nil
}

// GetFeature retrieves a cached feature definition by code.
//
//	feature, ok := cache.GetFeature("pages")
func (c *TenantCache) GetFeature(code string) (*Feature, bool) {
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
func (c *TenantCache) SetUser(user *User) error {
	if user == nil {
		return ErrNoUserContext
	}
	if user.UUID == "" {
		return ErrNoUserContext
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	c.users[user.UUID] = cacheEntry{value: cloneUser(user), expiresAt: c.now().Add(TTLUser)}
	if err := c.persistJSON(userGroup(user.UUID), "data", user, TTLUser); err != nil {
		return err
	}
	return nil
}

// GetUser retrieves a cached user by UUID. Returns nil, false on miss.
//
//	user, ok := cache.GetUser("user-7")
func (c *TenantCache) GetUser(uuid string) (*User, bool) {
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

func usageCacheKey(wsUUID, featureCode string) string {
	return wsUUID + "\x00" + normalizedFeatureCode(featureCode)
}

func hasUsagePrefix(key, wsUUID string) bool {
	if len(key) < len(wsUUID)+1 {
		return false
	}
	return key[:len(wsUUID)] == wsUUID && key[len(wsUUID)] == '\x00'
}

func workspaceRecordGroup(wsUUID string) string {
	return "ws:" + wsUUID + ":record"
}

func workspaceSlugGroup(slug string) string {
	return "ws:slug:" + slug
}

func workspaceIDGroup(id int64) string {
	return "ws:id:" + strconv.FormatInt(id, 10)
}

func packagesGroup(wsUUID string) string {
	return "ws:" + wsUUID + ":packages"
}

func boostsGroup(wsUUID string) string {
	return "ws:" + wsUUID + ":boosts"
}

func usageGroup(wsUUID, featureCode string) string {
	return "ws:" + wsUUID + ":usage:" + featureCode
}

func usagePrefix(wsUUID string) string {
	return "ws:" + wsUUID + ":usage:"
}

func featureGroup(code string) string {
	return "feature:" + normalizedFeatureCode(code)
}

func userGroup(uuid string) string {
	return "user:" + uuid
}
