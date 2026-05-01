// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	"dappco.re/go"
)

func testTenantServer(status int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if _, err := w.Write([]byte(body)); err != nil {
			touchError(err)
		}
	}))
}

func touchError(error) {}

func TestClient_NewTenantClient_Good(t *core.T) {
	client := NewTenantClient("https://api.example/", "token")
	core.AssertEqual(t, "https://api.example", client.baseURL)
	core.AssertNotNil(t, client.httpClient)
}

func TestClient_NewTenantClient_Bad(t *core.T) {
	client := NewTenantClient("", "")
	result := client.GetWorkspaceBySlug(context.Background(), "acme")
	requireResultFail(t, result)
	core.AssertEqual(t, "", client.baseURL)
}

func TestClient_NewTenantClient_Ugly(t *core.T) {
	client := NewTenantClient("https://api.example///", "token")
	core.AssertEqual(t, "https://api.example", client.baseURL)
	core.AssertEqual(t, 10*time.Second, client.timeout)
}

func TestClient_WithTimeout_Good(t *core.T) {
	client := NewTenantClient("https://api.example", "token", WithTimeout(time.Second))
	core.AssertEqual(t, time.Second, client.timeout)
	core.AssertEqual(t, time.Second, client.httpClient.Timeout)
}

func TestClient_WithTimeout_Bad(t *core.T) {
	client := NewTenantClient("https://api.example", "token", WithTimeout(0))
	core.AssertEqual(t, 10*time.Second, client.timeout)
	core.AssertEqual(t, 10*time.Second, client.httpClient.Timeout)
}

func TestClient_WithTimeout_Ugly(t *core.T) {
	client := &TenantClient{}
	WithTimeout(2 * time.Second)(client)
	core.AssertEqual(t, 2*time.Second, client.timeout)
	core.AssertNotNil(t, client.httpClient)
}

func TestClient_TenantClient_GetWorkspaceBySlug_Good(t *core.T) {
	server := testTenantServer(http.StatusOK, `{"ok":true,"data":{"uuid":"uuid-7","slug":"acme"}}`)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").GetWorkspaceBySlug(context.Background(), "acme")
	requireResultOK(t, result)
	core.AssertEqual(t, "acme", result.Value.(*Workspace).Slug)
}

func TestClient_TenantClient_GetWorkspaceBySlug_Bad(t *core.T) {
	server := testTenantServer(http.StatusNotFound, ``)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").GetWorkspaceBySlug(context.Background(), "missing")
	requireResultFail(t, result)
	core.AssertEqual(t, ErrWorkspaceNotFound, result.Value)
}

func TestClient_TenantClient_GetWorkspaceBySlug_Ugly(t *core.T) {
	server := testTenantServer(http.StatusOK, `{`)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").GetWorkspaceBySlug(context.Background(), "acme")
	requireResultFail(t, result)
	core.AssertEqual(t, "tenant: invalid api payload", result.Error())
}

func TestClient_TenantClient_GetWorkspaceByUUID_Good(t *core.T) {
	server := testTenantServer(http.StatusOK, `{"uuid":"uuid-7","slug":"acme"}`)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").GetWorkspaceByUUID(context.Background(), "uuid-7")
	requireResultOK(t, result)
	core.AssertEqual(t, "uuid-7", result.Value.(*Workspace).UUID)
}

func TestClient_TenantClient_GetWorkspaceByUUID_Bad(t *core.T) {
	server := testTenantServer(http.StatusNotFound, ``)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").GetWorkspaceByUUID(context.Background(), "missing")
	requireResultFail(t, result)
	core.AssertEqual(t, ErrWorkspaceNotFound, result.Value)
}

func TestClient_TenantClient_GetWorkspaceByUUID_Ugly(t *core.T) {
	server := testTenantServer(http.StatusOK, ``)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").GetWorkspaceByUUID(context.Background(), "uuid-7")
	requireResultFail(t, result)
	core.AssertEqual(t, core.EOF, result.Value)
}

func TestClient_TenantClient_GetWorkspaceByID_Good(t *core.T) {
	server := testTenantServer(http.StatusOK, `{"id":7,"uuid":"uuid-7"}`)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").GetWorkspaceByID(context.Background(), 7)
	requireResultOK(t, result)
	core.AssertEqual(t, int64(7), result.Value.(*Workspace).ID)
}

func TestClient_TenantClient_GetWorkspaceByID_Bad(t *core.T) {
	server := testTenantServer(http.StatusNotFound, ``)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").GetWorkspaceByID(context.Background(), 404)
	requireResultFail(t, result)
	core.AssertEqual(t, ErrWorkspaceNotFound, result.Value)
}

func TestClient_TenantClient_GetWorkspaceByID_Ugly(t *core.T) {
	server := testTenantServer(http.StatusInternalServerError, `{"error":"down"}`)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").GetWorkspaceByID(context.Background(), 7)
	requireResultFail(t, result)
	core.AssertContains(t, result.Error(), "down")
}

func TestClient_TenantClient_GetWorkspaceBySubdomain_Good(t *core.T) {
	server := testTenantServer(http.StatusOK, `{"uuid":"uuid-7","slug":"acme"}`)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").GetWorkspaceBySubdomain(context.Background(), "acme.host.uk.com")
	requireResultOK(t, result)
	core.AssertEqual(t, "acme", result.Value.(*Workspace).Slug)
}

func TestClient_TenantClient_GetWorkspaceBySubdomain_Bad(t *core.T) {
	server := testTenantServer(http.StatusNotFound, ``)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").GetWorkspaceBySubdomain(context.Background(), "missing.host.uk.com")
	requireResultFail(t, result)
	core.AssertEqual(t, ErrWorkspaceNotFound, result.Value)
}

func TestClient_TenantClient_GetWorkspaceBySubdomain_Ugly(t *core.T) {
	server := testTenantServer(http.StatusOK, `{`)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").GetWorkspaceBySubdomain(context.Background(), "acme.host.uk.com")
	requireResultFail(t, result)
	core.AssertEqual(t, "tenant: invalid api payload", result.Error())
}

func TestClient_TenantClient_GetUser_Good(t *core.T) {
	server := testTenantServer(http.StatusOK, `{"uuid":"user-9","email":"ada@example.uk"}`)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").GetUser(context.Background())
	requireResultOK(t, result)
	core.AssertEqual(t, "user-9", result.Value.(*User).UUID)
}

func TestClient_TenantClient_GetUser_Bad(t *core.T) {
	server := testTenantServer(http.StatusInternalServerError, `{"error":"bad token"}`)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").GetUser(context.Background())
	requireResultFail(t, result)
	core.AssertContains(t, result.Error(), "bad token")
}

func TestClient_TenantClient_GetUser_Ugly(t *core.T) {
	server := testTenantServer(http.StatusOK, ``)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").GetUser(context.Background())
	requireResultFail(t, result)
	core.AssertEqual(t, core.EOF, result.Value)
}

func TestClient_TenantClient_GetPackagesForWorkspace_Good(t *core.T) {
	server := testTenantServer(http.StatusOK, `{"ok":true,"data":[{"code":"starter","is_active":true}]}`)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").GetPackagesForWorkspace(context.Background(), "uuid-7")
	requireResultOK(t, result)
	core.AssertEqual(t, "starter", result.Value.([]Package)[0].Code)
}

func TestClient_TenantClient_GetPackagesForWorkspace_Bad(t *core.T) {
	server := testTenantServer(http.StatusInternalServerError, `{"error":"packages unavailable"}`)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").GetPackagesForWorkspace(context.Background(), "uuid-7")
	requireResultFail(t, result)
	core.AssertContains(t, result.Error(), "packages unavailable")
}

func TestClient_TenantClient_GetPackagesForWorkspace_Ugly(t *core.T) {
	server := testTenantServer(http.StatusOK, `{`)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").GetPackagesForWorkspace(context.Background(), "uuid-7")
	requireResultFail(t, result)
	core.AssertEqual(t, "tenant: invalid api payload", result.Error())
}

func TestClient_TenantClient_GetBoostsForWorkspace_Good(t *core.T) {
	server := testTenantServer(http.StatusOK, `{"data":[{"feature_code":"pages","status":"active"}]}`)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").GetBoostsForWorkspace(context.Background(), "uuid-7")
	requireResultOK(t, result)
	core.AssertEqual(t, "pages", result.Value.([]Boost)[0].FeatureCode)
}

func TestClient_TenantClient_GetBoostsForWorkspace_Bad(t *core.T) {
	server := testTenantServer(http.StatusInternalServerError, `{"error":"boosts unavailable"}`)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").GetBoostsForWorkspace(context.Background(), "uuid-7")
	requireResultFail(t, result)
	core.AssertContains(t, result.Error(), "boosts unavailable")
}

func TestClient_TenantClient_GetBoostsForWorkspace_Ugly(t *core.T) {
	server := testTenantServer(http.StatusOK, ``)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").GetBoostsForWorkspace(context.Background(), "uuid-7")
	requireResultFail(t, result)
	core.AssertEqual(t, core.EOF, result.Value)
}

func TestClient_TenantClient_GetCurrentUsage_Good(t *core.T) {
	server := testTenantServer(http.StatusOK, `{"ok":true,"count":7}`)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").GetCurrentUsage(context.Background(), "uuid-7", "pages")
	requireResultOK(t, result)
	core.AssertEqual(t, 7, result.Value.(int))
}

func TestClient_TenantClient_GetCurrentUsage_Bad(t *core.T) {
	server := testTenantServer(http.StatusInternalServerError, `{"error":"usage unavailable"}`)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").GetCurrentUsage(context.Background(), "uuid-7", "pages")
	requireResultFail(t, result)
	core.AssertContains(t, result.Error(), "usage unavailable")
}

func TestClient_TenantClient_GetCurrentUsage_Ugly(t *core.T) {
	server := testTenantServer(http.StatusOK, `"bad"`)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").GetCurrentUsage(context.Background(), "uuid-7", "pages")
	requireResultFail(t, result)
	core.AssertEqual(t, "tenant: invalid count payload", result.Error())
}

func TestClient_TenantClient_RecordUsage_Good(t *core.T) {
	server := testTenantServer(http.StatusOK, `{"ok":true}`)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").RecordUsage(context.Background(), "uuid-7", "pages", 1, testInt64(9), nil)
	requireResultOK(t, result)
	core.AssertNotNil(t, result.Value)
}

func TestClient_TenantClient_RecordUsage_Bad(t *core.T) {
	server := testTenantServer(http.StatusInternalServerError, `{"error":"write failed"}`)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").RecordUsage(context.Background(), "uuid-7", "pages", 1, nil, nil)
	requireResultFail(t, result)
	core.AssertContains(t, result.Error(), "write failed")
}

func TestClient_TenantClient_RecordUsage_Ugly(t *core.T) {
	server := testTenantServer(http.StatusOK, `{"ok":true}`)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").RecordUsage(context.Background(), "uuid-7", "pages", 0, nil, map[string]any{"source": "test"})
	requireResultOK(t, result)
	core.AssertNotNil(t, result.Value)
}

func TestClient_TenantClient_GetFeature_Good(t *core.T) {
	server := testTenantServer(http.StatusOK, `{"code":"pages","type":"limit"}`)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").GetFeature(context.Background(), "pages")
	requireResultOK(t, result)
	core.AssertEqual(t, "pages", result.Value.(*Feature).Code)
}

func TestClient_TenantClient_GetFeature_Bad(t *core.T) {
	server := testTenantServer(http.StatusNotFound, ``)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").GetFeature(context.Background(), "missing")
	requireResultFail(t, result)
	core.AssertEqual(t, ErrFeatureNotFound, result.Value)
}

func TestClient_TenantClient_GetFeature_Ugly(t *core.T) {
	server := testTenantServer(http.StatusOK, `{`)
	defer server.Close()
	result := NewTenantClient(server.URL, "token").GetFeature(context.Background(), "pages")
	requireResultFail(t, result)
	core.AssertEqual(t, "tenant: invalid api payload", result.Error())
}
