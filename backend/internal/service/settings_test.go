package service

import (
	"context"
	"testing"
)

func TestSettingsOverrideAndValidation(t *testing.T) {
	ctx := context.Background()
	svc := newTestSettings(map[string]string{"checkout.tax_rate_bps": "800"})

	if got := svc.Int(ctx, "checkout.tax_rate_bps"); got != 800 {
		t.Fatalf("override not applied: got %d, want 800", got)
	}

	// Unknown keys are rejected.
	if err := svc.Update(ctx, map[string]string{"does.not.exist": "1"}); err == nil {
		t.Fatal("expected an error for an unknown key")
	}
	// Out-of-range integers are rejected.
	if err := svc.Update(ctx, map[string]string{"checkout.tax_rate_bps": "99999"}); err == nil {
		t.Fatal("expected an error for an out-of-range value")
	}
	// Booleans are validated.
	if err := svc.Update(ctx, map[string]string{"auth.require_email_verification": "maybe"}); err == nil {
		t.Fatal("expected an error for a non-boolean")
	}

	// A valid update is persisted and visible immediately.
	if err := svc.Update(ctx, map[string]string{
		"checkout.tax_rate_bps":           "250",
		"auth.require_email_verification": "true",
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if got := svc.Int(ctx, "checkout.tax_rate_bps"); got != 250 {
		t.Fatalf("update not visible: got %d, want 250", got)
	}
	if !svc.Bool(ctx, "auth.require_email_verification") {
		t.Fatal("bool update not visible")
	}

	// List exposes every registered key with its default.
	items := svc.List(ctx)
	if len(items) == 0 {
		t.Fatal("expected settings")
	}
	for _, s := range items {
		if s.Key == "checkout.tax_rate_bps" && s.Value != "250" {
			t.Fatalf("list value = %q, want 250", s.Value)
		}
	}
}
