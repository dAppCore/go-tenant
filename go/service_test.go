// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"time"

	"dappco.re/go"
)

func TestService_NewService_Good(t *core.T) {
	factory := NewService(TenantOptions{
		APIURL:   "https://api.host.uk.com",
		APIToken: "bearer-token",
		Timeout:  5 * time.Second,
	})
	result := factory(core.New())
	requireResultOK(t, result)
	service := result.Value.(*Tenant)
	core.AssertNotNil(t, service.client)
	core.AssertNotNil(t, service.cache)
	core.AssertNotNil(t, service.entitlements)
	core.AssertEqual(t, 5*time.Second, service.client.timeout)
}

func TestService_NewService_Bad(t *core.T) {
	factory := NewService(TenantOptions{})
	result := factory(nil)
	requireResultFail(t, result)
	core.AssertContains(t, result.Error(), "core is nil")
}

func TestService_NewService_Ugly(t *core.T) {
	// No APIURL/APIToken means no client is built, but the cache and
	// entitlement service are still wired (cache-only mode).
	factory := NewService(TenantOptions{})
	result := factory(core.New())
	requireResultOK(t, result)
	service := result.Value.(*Tenant)
	core.AssertNil(t, service.client)
	core.AssertNotNil(t, service.cache)
	core.AssertNotNil(t, service.entitlements)
}
