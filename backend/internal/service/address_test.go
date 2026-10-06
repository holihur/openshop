package service

import (
	"context"
	"testing"
	"time"

	"github.com/holihur/openshop/internal/domain"
)

func newAddressService() (*AddressService, *fakeAddressRepo) {
	repo := newFakeAddressRepo()
	svc := NewAddressService(repo, &seqIDs{}, fixedClock{t: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)})
	return svc, repo
}

func TestFirstAddressBecomesDefault(t *testing.T) {
	svc, _ := newAddressService()
	ctx := context.Background()

	first, err := svc.Create(ctx, "u1", AddressInput{Recipient: "Alice", Line1: "1 Main St"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if !first.Default {
		t.Fatal("first address should be the default")
	}

	second, err := svc.Create(ctx, "u1", AddressInput{Recipient: "Bob", Line1: "2 Side St"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if second.Default {
		t.Fatal("second address should not be the default")
	}
}

func TestSetDefaultSwitchesExclusively(t *testing.T) {
	svc, _ := newAddressService()
	ctx := context.Background()

	a1, _ := svc.Create(ctx, "u1", AddressInput{Recipient: "Alice", Line1: "1 Main St"})
	a2, _ := svc.Create(ctx, "u1", AddressInput{Recipient: "Bob", Line1: "2 Side St"})

	if _, err := svc.SetDefault(ctx, "u1", a2.ID); err != nil {
		t.Fatalf("set default: %v", err)
	}
	list, _ := svc.List(ctx, "u1")
	defaults := 0
	for _, a := range list {
		if a.Default {
			defaults++
			if a.ID != a2.ID {
				t.Fatalf("wrong default: %s", a.ID)
			}
		}
	}
	if defaults != 1 {
		t.Fatalf("defaults = %d, want exactly 1", defaults)
	}
	_ = a1
}

func TestAddressOwnershipIsEnforced(t *testing.T) {
	svc, _ := newAddressService()
	ctx := context.Background()

	a, _ := svc.Create(ctx, "u1", AddressInput{Recipient: "Alice", Line1: "1 Main St"})

	if _, err := svc.Update(ctx, "u2", a.ID, AddressInput{Recipient: "Mallory", Line1: "x"}); err != domain.ErrNotFound {
		t.Fatalf("cross-user update err = %v, want ErrNotFound", err)
	}
	if err := svc.Delete(ctx, "u2", a.ID); err != domain.ErrNotFound {
		t.Fatalf("cross-user delete err = %v, want ErrNotFound", err)
	}
}

func TestAddressOneLine(t *testing.T) {
	a := domain.Address{Province: "Beijing", City: "Beijing", District: "Haidian", Line1: "1 Zhongguancun", PostalCode: "100080"}
	if got := a.OneLine(); got != "Beijing Beijing Haidian 1 Zhongguancun 100080" {
		t.Fatalf("OneLine = %q", got)
	}
}
