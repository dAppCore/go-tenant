// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	"dappco.re/go"
)

// testHangingServer returns a server that sleeps long enough to trip a short
// client timeout, exercising the request deadline branch.
func testHangingServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
}

func TestClientInternal_boolField_Good(t *core.T) {
	core.AssertTrue(t, boolField(map[string]any{"ok": true}, "ok"))
	core.AssertFalse(t, boolField(map[string]any{"ok": false}, "ok"))
}

func TestClientInternal_boolField_Bad(t *core.T) {
	core.AssertFalse(t, boolField(map[string]any{"ok": "yes"}, "ok"))
	core.AssertFalse(t, boolField(map[string]any{}, "ok"))
}

func TestClientInternal_boolField_Ugly(t *core.T) {
	core.AssertFalse(t, boolField(map[string]any{"ok": nil}, "ok"))
	core.AssertFalse(t, boolField(map[string]any{"ok": 1}, "ok"))
}

func TestClientInternal_intField_Good(t *core.T) {
	value, ok := intField(map[string]any{"count": 7}, "count")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, 7, value)
	floatValue, floatOK := intField(map[string]any{"count": float64(42)}, "count")
	core.AssertTrue(t, floatOK)
	core.AssertEqual(t, 42, floatValue)
}

func TestClientInternal_intField_Bad(t *core.T) {
	_, ok := intField(map[string]any{}, "count")
	core.AssertFalse(t, ok)
	_, nilOK := intField(map[string]any{"count": nil}, "count")
	core.AssertFalse(t, nilOK)
	_, boolOK := intField(map[string]any{"count": true}, "count")
	core.AssertFalse(t, boolOK)
}

func TestClientInternal_intField_Ugly(t *core.T) {
	stringValue, stringOK := intField(map[string]any{"count": " 13 "}, "count")
	core.AssertTrue(t, stringOK)
	core.AssertEqual(t, 13, stringValue)
	_, badStringOK := intField(map[string]any{"count": "not-a-number"}, "count")
	core.AssertFalse(t, badStringOK)
}

func TestClientInternal_intField_IntegerWidths_Good(t *core.T) {
	for _, value := range []any{int8(1), int16(2), int32(3), int64(4), uint(5), uint8(6), uint16(7), uint32(8), uint64(9), float32(10)} {
		got, ok := intField(map[string]any{"count": value}, "count")
		core.AssertTrue(t, ok)
		core.AssertTrue(t, got > 0)
	}
}

func TestClientInternal_stringField_Good(t *core.T) {
	core.AssertEqual(t, "not found", stringField(map[string]any{"error": "not found"}, "error"))
}

func TestClientInternal_stringField_Bad(t *core.T) {
	core.AssertEqual(t, "", stringField(map[string]any{"error": 42}, "error"))
	core.AssertEqual(t, "", stringField(map[string]any{}, "error"))
}

func TestClientInternal_stringField_Ugly(t *core.T) {
	core.AssertEqual(t, "", stringField(map[string]any{"error": nil}, "error"))
}

func TestClientInternal_looksLikeEnvelope_Good(t *core.T) {
	core.AssertTrue(t, looksLikeEnvelope(map[string]any{"ok": true, "data": 1}))
	core.AssertTrue(t, looksLikeEnvelope(map[string]any{"error": "boom"}))
}

func TestClientInternal_looksLikeEnvelope_Bad(t *core.T) {
	core.AssertFalse(t, looksLikeEnvelope(map[string]any{"slug": "acme"}))
	core.AssertFalse(t, looksLikeEnvelope(map[string]any{}))
}

func TestClientInternal_looksLikeEnvelope_Ugly(t *core.T) {
	core.AssertFalse(t, looksLikeEnvelope(nil))
}

func TestClientInternal_decodeCount_Good(t *core.T) {
	result := decodeCount([]byte(`{"ok":true,"count":7}`))
	requireResultOK(t, result)
	core.AssertEqual(t, 7, result.Value.(int))
}

func TestClientInternal_decodeCount_DirectInteger_Good(t *core.T) {
	result := decodeCount([]byte(`42`))
	requireResultOK(t, result)
	core.AssertEqual(t, 42, result.Value.(int))
}

func TestClientInternal_decodeCount_NestedData_Good(t *core.T) {
	// Nested data recurses on the inner payload; a bare integer under "data"
	// decodes successfully. (A nested object without ok/error/data keys is
	// not treated as an envelope, so {"data":{"count":5}} does NOT resolve.)
	bareNested := decodeCount([]byte(`{"data":11}`))
	requireResultOK(t, bareNested)
	core.AssertEqual(t, 11, bareNested.Value.(int))
}

func TestClientInternal_decodeCount_NestedNonEnvelope_Ugly(t *core.T) {
	// Inner object lacks envelope markers and is not a bare number, so the
	// recursion fails and the outer call reports an invalid count payload.
	result := decodeCount([]byte(`{"data":{"count":5}}`))
	requireResultFail(t, result)
	core.AssertEqual(t, "tenant: invalid count payload", result.Error())
}

func TestClientInternal_decodeCount_UsageKey_Good(t *core.T) {
	result := decodeCount([]byte(`{"ok":true,"usage":9}`))
	requireResultOK(t, result)
	core.AssertEqual(t, 9, result.Value.(int))
}

func TestClientInternal_decodeCount_Bad(t *core.T) {
	result := decodeCount([]byte(``))
	requireResultFail(t, result)
	core.AssertEqual(t, core.EOF, result.Value)
}

func TestClientInternal_decodeCount_Ugly(t *core.T) {
	result := decodeCount([]byte(`{"ok":false,"error":"counter offline"}`))
	requireResultFail(t, result)
	core.AssertContains(t, result.Error(), "counter offline")
	invalid := decodeCount([]byte(`"bad"`))
	requireResultFail(t, invalid)
	core.AssertEqual(t, "tenant: invalid count payload", invalid.Error())
}

func TestClientInternal_decodeEnvelope_ErrorEnvelope_Bad(t *core.T) {
	var workspace Workspace
	result := decodeEnvelope([]byte(`{"ok":false,"error":"workspace gone"}`), &workspace)
	requireResultFail(t, result)
	core.AssertContains(t, result.Error(), "workspace gone")
}

func TestClientInternal_decodeEnvelope_NilData_Ugly(t *core.T) {
	// data:null skips the nested branch and decodes the whole envelope into
	// the target — unknown envelope keys are ignored, leaving a zero value.
	var workspace Workspace
	result := decodeEnvelope([]byte(`{"ok":true,"data":null}`), &workspace)
	requireResultOK(t, result)
	core.AssertEqual(t, "", workspace.UUID)
}

func TestClientInternal_decodeEnvelope_Empty_Bad(t *core.T) {
	var workspace Workspace
	result := decodeEnvelope([]byte(``), &workspace)
	requireResultFail(t, result)
	core.AssertEqual(t, core.EOF, result.Value)
}

func TestClientInternal_request_NilClient_Bad(t *core.T) {
	var client *TenantClient
	result := client.request(context.Background(), http.MethodGet, "/x", nil)
	requireResultFail(t, result)
	core.AssertContains(t, result.Error(), "tenant client is nil")
}

func TestClientInternal_request_Timeout_Ugly(t *core.T) {
	server := testHangingServer()
	defer server.Close()
	client := NewTenantClient(server.URL, "token", WithTimeout(50*time.Millisecond))
	result := client.request(context.Background(), http.MethodGet, "/slow", nil)
	requireResultFail(t, result)
	core.AssertEqual(t, ErrClientTimeout, result.Value)
}

func TestClientInternal_request_BadURL_Bad(t *core.T) {
	client := NewTenantClient("://not-a-url", "token")
	result := client.request(context.Background(), "  bad method  ", "/x", nil)
	requireResultFail(t, result)
}
