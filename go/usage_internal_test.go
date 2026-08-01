// SPDX-License-Identifier: EUPL-1.2

package tenant

import "dappco.re/go"

func TestUsageInternal_newUsageRecord_Good(t *core.T) {
	record := newUsageRecord(7, "Pages", 2, testInt64(9), map[string]any{"page_id": 42})
	core.AssertEqual(t, int64(7), record.WorkspaceID)
	core.AssertEqual(t, "pages", record.FeatureCode)
	core.AssertEqual(t, 2, record.Quantity)
	core.AssertEqual(t, int64(9), *record.UserID)
	core.AssertEqual(t, 42, record.Metadata["page_id"])
}

func TestUsageInternal_newUsageRecord_Bad(t *core.T) {
	// Non-positive quantity normalises to 1; nil userID and metadata stay nil.
	record := newUsageRecord(7, "pages", 0, nil, nil)
	core.AssertEqual(t, 1, record.Quantity)
	core.AssertNil(t, record.UserID)
	core.AssertNil(t, record.Metadata)
}

func TestUsageInternal_newUsageRecord_Ugly(t *core.T) {
	record := newUsageRecord(7, "pages", -5, testInt64(0), map[string]any{})
	core.AssertEqual(t, 1, record.Quantity)
	core.AssertEqual(t, int64(0), *record.UserID)
	// Empty metadata map is not cloned (len 0), so it stays nil.
	core.AssertNil(t, record.Metadata)
}

func TestUsageInternal_addQuantity_Good(t *core.T) {
	record := newUsageRecord(7, "pages", 1, nil, nil).addQuantity(2)
	core.AssertEqual(t, 3, record.Quantity)
}

func TestUsageInternal_addQuantity_Bad(t *core.T) {
	record := newUsageRecord(7, "pages", 4, nil, nil).addQuantity(0)
	core.AssertEqual(t, 4, record.Quantity)
}

func TestUsageInternal_addQuantity_Ugly(t *core.T) {
	record := newUsageRecord(7, "pages", 4, nil, nil).addQuantity(-3)
	core.AssertEqual(t, 4, record.Quantity)
}

func TestUsageInternal_payload_Good(t *core.T) {
	payload := newUsageRecord(7, "pages", 2, testInt64(9), map[string]any{"k": "v"}).payload()
	core.AssertEqual(t, "pages", payload["feature_code"])
	core.AssertEqual(t, 2, payload["quantity"])
	core.AssertEqual(t, int64(9), payload["user_id"])
}

func TestUsageInternal_payload_Bad(t *core.T) {
	payload := newUsageRecord(7, "pages", 1, nil, nil).payload()
	_, hasUser := payload["user_id"]
	core.AssertFalse(t, hasUser)
	core.AssertNil(t, payload["metadata"])
}

func TestUsageInternal_payload_Ugly(t *core.T) {
	payload := newUsageRecord(7, "pages", 0, testInt64(-1), nil).payload()
	core.AssertEqual(t, 1, payload["quantity"])
	core.AssertEqual(t, int64(-1), payload["user_id"])
}

func TestUsageInternal_cloneInt64_Good(t *core.T) {
	original := int64(42)
	clone := cloneInt64(&original)
	core.AssertEqual(t, int64(42), *clone)
	*clone = 99
	core.AssertEqual(t, int64(42), original)
}

func TestUsageInternal_cloneInt64_Bad(t *core.T) {
	core.AssertNil(t, cloneInt64(nil))
}

func TestUsageInternal_cloneInt64_Ugly(t *core.T) {
	zero := int64(0)
	clone := cloneInt64(&zero)
	core.AssertNotNil(t, clone)
	core.AssertEqual(t, int64(0), *clone)
}

func TestUsageInternal_cloneMetadata_Good(t *core.T) {
	original := map[string]any{"a": 1}
	clone := cloneMetadata(original)
	clone["a"] = 2
	core.AssertEqual(t, 1, original["a"])
}

func TestUsageInternal_cloneMetadata_Bad(t *core.T) {
	core.AssertNil(t, cloneMetadata(nil))
}

func TestUsageInternal_cloneMetadata_Ugly(t *core.T) {
	clone := cloneMetadata(map[string]any{})
	core.AssertNotNil(t, clone)
	core.AssertEqual(t, 0, len(clone))
}
