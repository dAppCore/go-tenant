# Architecture

The package centers on `Tenant`, which composes `TenantClient`, `TenantCache`,
and `EntitlementService`.

`TenantClient` talks to the PHP REST API. `TenantCache` keeps local workspace,
package, boost, usage, feature, and user data. `WorkspaceScope` resolves
workspace context for HTTP handlers and background jobs.
