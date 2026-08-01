// SPDX-License-Identifier: EUPL-1.2

// Service registration factory for the tenant package — exposes the
// canonical `NewService(opts)` shape that #1336 established across
// the canonical Go repo set.
//
//	c, _ := core.New(
//	    core.WithService(tenant.NewService(tenant.TenantOptions{
//	        APIURL:   "https://api.host.uk.com",
//	        APIToken: "bearer-token",
//	        Timeout:  5 * time.Second,
//	    })),
//	)
//
// `Register(c)` (in tenant.go) is the imperative-style alternative for
// consumers booting from Core config — it reads APIURL / APIToken /
// Timeout from the Core's Config(). Use NewService when you have
// options in hand and don't want to round-trip through Core config.
//
// Implementation lives here as a thin wrapper over the existing
// service-construction logic in tenant.go's Register, so behaviour is
// identical regardless of entry point.

package tenant

import (
	core "dappco.re/go"
)

// NewService returns a factory that constructs a *Tenant with the given
// options and registers it under "tenant" via core.WithService.
//
// Use through core.WithService:
//
//	core.WithService(tenant.NewService(tenant.TenantOptions{
//	    APIURL:   apiURL,
//	    APIToken: token,
//	}))
//
// For config-driven boot (read APIURL/APIToken/Timeout from c.Config())
// use Register(c) directly instead.
func NewService(opts TenantOptions) func(*core.Core) core.Result {
	return func(c *core.Core) core.Result {
		if c == nil {
			return core.Fail(core.E("tenant.NewService", "core is nil", nil))
		}
		service := &Tenant{
			ServiceRuntime: core.NewServiceRuntime(c, opts),
		}
		if opts.APIURL != "" && opts.APIToken != "" {
			service.client = NewTenantClient(opts.APIURL, opts.APIToken, WithTimeout(opts.Timeout))
		}
		service.cache = NewTenantCache(nil)
		service.entitlements = NewLocalEntitlementService(service.cache, service.client)
		return core.Ok(service)
	}
}
