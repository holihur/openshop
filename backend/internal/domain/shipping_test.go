package domain

import (
	"testing"
	"time"
)

func TestAddBusinessDaysSkipsWeekends(t *testing.T) {
	// Friday 2026-03-06 + 1 business day lands on Monday.
	friday := time.Date(2026, 3, 6, 10, 0, 0, 0, time.UTC)
	if got := AddBusinessDays(friday, 1); got.Weekday() != time.Monday {
		t.Errorf("Friday + 1 business day = %s, want Monday", got.Weekday())
	}
	if got := AddBusinessDays(friday, 0); !got.Equal(friday) {
		t.Errorf("zero business days must be a no-op, got %s", got)
	}
	// A full week is exactly seven calendar days regardless of the start day.
	monday := time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC)
	if got := AddBusinessDays(monday, 5); !got.Equal(monday.AddDate(0, 0, 7)) {
		t.Errorf("Monday + 5 business days = %s, want the following Monday", got.Format(time.DateOnly))
	}
	// The result is never a weekend day.
	d := monday
	for i := 0; i < 20; i++ {
		d = AddBusinessDays(d, 1)
		if wd := d.Weekday(); wd == time.Saturday || wd == time.Sunday {
			t.Fatalf("business day arithmetic produced a %s", wd)
		}
	}
}

func TestDeliveryWindowHonoursZoneOverride(t *testing.T) {
	method := &ShippingMethod{MinDays: 3, MaxDays: 5}
	if min, max := DeliveryWindow(method, nil); min != 3 || max != 5 {
		t.Errorf("method window = %d..%d, want 3..5", min, max)
	}
	// A zero on the rate means "inherit the method".
	if min, max := DeliveryWindow(method, &ShippingRate{}); min != 3 || max != 5 {
		t.Errorf("zero override = %d..%d, want 3..5", min, max)
	}
	// A remote zone can promise a longer window.
	if min, max := DeliveryWindow(method, &ShippingRate{MinDays: 7, MaxDays: 12}); min != 7 || max != 12 {
		t.Errorf("override = %d..%d, want 7..12", min, max)
	}
	// An inverted window is repaired rather than returned.
	if min, max := DeliveryWindow(method, &ShippingRate{MinDays: 9, MaxDays: 4}); min != 9 || max != 9 {
		t.Errorf("inverted override = %d..%d, want 9..9", min, max)
	}
}
