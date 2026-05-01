// SPDX-License-Identifier: EUPL-1.2

package tenant

func ExampleBoost_IsUsable() {
	boost := Boost{BoostType: BoostTypeAddLimit, Status: BoostStatusActive, LimitValue: 10}
	if boost.IsUsable() {
		boost.Remaining()
	}
}

func ExampleBoost_Remaining() {
	boost := Boost{BoostType: BoostTypeAddLimit, LimitValue: 10, ConsumedQuantity: 3}
	boost.Remaining()
}
