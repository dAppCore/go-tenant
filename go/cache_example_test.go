// SPDX-License-Identifier: EUPL-1.2

package tenant

func ExampleNewTenantCache() {
	cache := NewTenantCache(nil)
	cache.GetWorkspace("uuid-7")
}

func ExampleTenantCache_SetWorkspace() {
	cache := NewTenantCache(nil)
	cache.SetWorkspace(&Workspace{ID: 7, UUID: "uuid-7", Slug: "acme"})
}

func ExampleTenantCache_GetWorkspace() {
	cache := NewTenantCache(nil)
	cache.SetWorkspace(&Workspace{UUID: "uuid-7"})
	cache.GetWorkspace("uuid-7")
}

func ExampleTenantCache_GetWorkspaceByID() {
	cache := NewTenantCache(nil)
	cache.SetWorkspace(&Workspace{ID: 7, UUID: "uuid-7"})
	cache.GetWorkspaceByID(7)
}

func ExampleTenantCache_GetWorkspaceBySlug() {
	cache := NewTenantCache(nil)
	cache.SetWorkspace(&Workspace{UUID: "uuid-7", Slug: "acme"})
	cache.GetWorkspaceBySlug("acme")
}

func ExampleTenantCache_SetPackages() {
	cache := NewTenantCache(nil)
	cache.SetPackages("uuid-7", []Package{{Code: "starter"}})
}

func ExampleTenantCache_GetPackages() {
	cache := NewTenantCache(nil)
	cache.SetPackages("uuid-7", []Package{{Code: "starter"}})
	cache.GetPackages("uuid-7")
}

func ExampleTenantCache_SetBoosts() {
	cache := NewTenantCache(nil)
	cache.SetBoosts("uuid-7", []Boost{{FeatureCode: "pages"}})
}

func ExampleTenantCache_GetBoosts() {
	cache := NewTenantCache(nil)
	cache.SetBoosts("uuid-7", []Boost{{FeatureCode: "pages"}})
	cache.GetBoosts("uuid-7")
}

func ExampleTenantCache_SetUsage() {
	cache := NewTenantCache(nil)
	cache.SetUsage("uuid-7", "pages", 3)
}

func ExampleTenantCache_GetUsage() {
	cache := NewTenantCache(nil)
	cache.SetUsage("uuid-7", "pages", 3)
	cache.GetUsage("uuid-7", "pages")
}

func ExampleTenantCache_InvalidateWorkspace() {
	cache := NewTenantCache(nil)
	cache.SetWorkspace(&Workspace{UUID: "uuid-7"})
	cache.InvalidateWorkspace("uuid-7")
}

func ExampleTenantCache_SetFeature() {
	cache := NewTenantCache(nil)
	cache.SetFeature(&Feature{Code: "pages", Type: FeatureTypeLimit})
}

func ExampleTenantCache_GetFeature() {
	cache := NewTenantCache(nil)
	cache.SetFeature(&Feature{Code: "pages", Type: FeatureTypeLimit})
	cache.GetFeature("pages")
}

func ExampleTenantCache_SetUser() {
	cache := NewTenantCache(nil)
	cache.SetUser(&User{UUID: "user-9", Email: "ada@example.uk"})
}

func ExampleTenantCache_GetUser() {
	cache := NewTenantCache(nil)
	cache.SetUser(&User{UUID: "user-9", Email: "ada@example.uk"})
	cache.GetUser("user-9")
}
