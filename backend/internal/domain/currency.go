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
