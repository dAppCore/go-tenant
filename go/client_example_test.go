// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"context"
	"time"
)

func ExampleNewTenantClient() {
	client := NewTenantClient("https://api.example", "token")
	client.GetUser(context.Background())
}

func ExampleWithTimeout() {
	client := NewTenantClient("https://api.example", "token", WithTimeout(5*time.Second))
	client.GetUser(context.Background())
}

func ExampleTenantClient_GetWorkspaceBySlug() {
	client := NewTenantClient("https://api.example", "token")
	client.GetWorkspaceBySlug(context.Background(), "acme")
}

func ExampleTenantClient_GetWorkspaceByUUID() {
	client := NewTenantClient("https://api.example", "token")
	client.GetWorkspaceByUUID(context.Background(), "uuid-7")
}

func ExampleTenantClient_GetWorkspaceByID() {
	client := NewTenantClient("https://api.example", "token")
	client.GetWorkspaceByID(context.Background(), 7)
}

func ExampleTenantClient_GetWorkspaceBySubdomain() {
	client := NewTenantClient("https://api.example", "token")
	client.GetWorkspaceBySubdomain(context.Background(), "acme.host.uk.com")
}

func ExampleTenantClient_GetUser() {
	client := NewTenantClient("https://api.example", "token")
	client.GetUser(context.Background())
}

func ExampleTenantClient_GetPackagesForWorkspace() {
	client := NewTenantClient("https://api.example", "token")
	client.GetPackagesForWorkspace(context.Background(), "uuid-7")
}

func ExampleTenantClient_GetBoostsForWorkspace() {
	client := NewTenantClient("https://api.example", "token")
	client.GetBoostsForWorkspace(context.Background(), "uuid-7")
}

func ExampleTenantClient_GetCurrentUsage() {
	client := NewTenantClient("https://api.example", "token")
	client.GetCurrentUsage(context.Background(), "uuid-7", "pages")
}

func ExampleTenantClient_RecordUsage() {
	client := NewTenantClient("https://api.example", "token")
	client.RecordUsage(context.Background(), "uuid-7", "pages", 1, nil, nil)
}

func ExampleTenantClient_GetFeature() {
	client := NewTenantClient("https://api.example", "token")
	client.GetFeature(context.Background(), "pages")
}
