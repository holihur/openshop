package domain

import "testing"

// The rate direction is the difference between charging a shopper the right
// amount and charging them the square of it, so it is asserted explicitly.
func TestConvertMultipliesByTheTargetRate(t *testing.T) {
	const micro = 1_000_000

	// Base to itself is a no-op, which is what every base-currency order does.
	if got := Convert(19900, micro); got != 19900 {
		t.Errorf("Convert(19900, 1.0) = %d, want 19900", got)
	}
	// 1 CNY = 0.14 USD, so ¥199.00 is $27.86.
	if got := Convert(19900, 140000); got != 2786 {
		t.Errorf("Convert(19900, 0.14) = %d, want 2786", got)
	}
	// The inverse quote stored by mistake would multiply instead of divide.
	if got := Convert(19900, 7_000_000); got == 2786 {
		t.Error("a rate of 7 must not behave like 0.14")
	}
	// Rounding is to the nearest minor unit, in both directions.
	if got := Convert(1, 1_666_667); got != 2 {
		t.Errorf("Convert(1, 1.667) = %d, want 2", got)
	}
	if got := Convert(-19900, 140000); got != -2786 {
		t.Errorf("negative amounts convert symmetrically, got %d", got)
	}
	// A missing or nonsensical rate falls back to identity rather than zeroing
	// the amount.
	if got := Convert(19900, 0); got != 19900 {
		t.Errorf("Convert(19900, 0) = %d, want 19900", got)
	}
}
