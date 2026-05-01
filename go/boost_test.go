// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"time"

	"dappco.re/go"
)

func TestBoost_Boost_IsUsable_Good(t *core.T) {
	boost := Boost{BoostType: BoostTypeAddLimit, Status: BoostStatusActive, LimitValue: 10, ConsumedQuantity: 2}
	core.AssertTrue(t, boost.IsUsable())
	core.AssertEqual(t, 8, boost.Remaining())
}

func TestBoost_Boost_IsUsable_Bad(t *core.T) {
	expired := time.Now().Add(-time.Minute)
	boost := Boost{BoostType: BoostTypeAddLimit, Status: BoostStatusActive, LimitValue: 10, ExpiresAt: &expired}
	core.AssertFalse(t, boost.IsUsable())
	core.AssertEqual(t, BoostStatusActive, boost.Status)
}

func TestBoost_Boost_IsUsable_Ugly(t *core.T) {
	starts := time.Now().Add(time.Minute)
	boost := Boost{BoostType: BoostTypeAddLimit, Status: BoostStatusActive, LimitValue: 10, StartsAt: &starts}
	core.AssertFalse(t, boost.IsUsable())
	core.AssertEqual(t, 10, boost.Remaining())
}

func TestBoost_Boost_Remaining_Good(t *core.T) {
	boost := Boost{BoostType: BoostTypeAddLimit, LimitValue: 10, ConsumedQuantity: 3}
	core.AssertEqual(t, 7, boost.Remaining())
	core.AssertEqual(t, BoostTypeAddLimit, boost.BoostType)
}

func TestBoost_Boost_Remaining_Bad(t *core.T) {
	boost := Boost{BoostType: BoostTypeAddLimit, LimitValue: 10, ConsumedQuantity: 12}
	core.AssertEqual(t, 0, boost.Remaining())
	core.AssertEqual(t, 12, boost.ConsumedQuantity)
}

func TestBoost_Boost_Remaining_Ugly(t *core.T) {
	boost := Boost{BoostType: BoostTypeUnlimited}
	core.AssertEqual(t, -1, boost.Remaining())
	core.AssertEqual(t, BoostTypeUnlimited, boost.BoostType)
}
