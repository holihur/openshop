package domain

import "time"

// ExchangeRate converts the store base currency into a target currency. The
// rate is stored in micro units (rate * 1e6) to avoid floating point.
type ExchangeRate struct {
	Currency  string
	RateMicro int64
	UpdatedAt time.Time
}

// Convert converts minor units using a micro rate, rounding half up.
// Convert multiplies a base-currency amount by the rate of the target currency.
//
// The convention is therefore "how much of the target currency one unit of the
// base currency buys": for a store whose base is CNY, USD is about 0.14, not 7.
// Storing the inverse quote (CNY per USD) prices every foreign order at roughly
// the square of the intended amount, so the direction matters and is pinned by a
// test.
func Convert(cents, rateMicro int64) int64 {
	if rateMicro <= 0 {
		rateMicro = 1_000_000
	}
	if cents == 0 {
		return 0
	}
	if cents < 0 {
		return -((-cents*rateMicro + 500_000) / 1_000_000)
	}
	return (cents*rateMicro + 500_000) / 1_000_000
}
