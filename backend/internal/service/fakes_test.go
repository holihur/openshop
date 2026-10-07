package service

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// This file contains in-memory fakes for every port. Because the services
// depend only on interfaces, the entire business logic can be tested with no
// PostgreSQL, Redis or NATS running.

type fakeUserRepo struct {
	mu   sync.Mutex
	byID map[string]*domain.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{byID: map[string]*domain.User{}}
}

func (r *fakeUserRepo) Create(_ context.Context, u *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.byID {
		if existing.Email != "" && existing.Email == u.Email {
			return domain.ErrConflict
		}
		if existing.Phone != "" && existing.Phone == u.Phone {
			return domain.ErrConflict
		}
	}
	cp := *u
	r.byID[u.ID] = &cp
	return nil
}

func (r *fakeUserRepo) Update(_ context.Context, u *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *u
	r.byID[u.ID] = &cp
	return nil
}

func (r *fakeUserRepo) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.byID, id)
	return nil
}

func (r *fakeUserRepo) List(_ context.Context, f domain.UserFilter) (domain.Page[domain.User], error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.User, 0)
	for _, u := range r.byID {
		if f.Role != nil && u.Role != *f.Role {
			continue
		}
		if f.Status != nil && u.Status != *f.Status {
			continue
		}
		out = append(out, *u)
	}
	return domain.Page[domain.User]{Items: out, Total: int64(len(out)), Page: 1, PageSize: len(out)}, nil
}

// RecordLoginFailure and ClearLoginFailures implement per-account throttling.
func (r *fakeUserRepo) RecordLoginFailure(_ context.Context, userID string, lockAfter int, lockUntil time.Time) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.byID[userID]
	if !ok {
		return 0, domain.ErrNotFound
	}
	u.FailedAttempts++
	if u.FailedAttempts >= lockAfter {
		locked := lockUntil
		u.LockedUntil = &locked
	}
	return u.FailedAttempts, nil
}

func (r *fakeUserRepo) ClearLoginFailures(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.byID[userID]; ok {
		u.FailedAttempts = 0
		u.LockedUntil = nil
	}
	return nil
}

func (r *fakeUserRepo) FindByID(_ context.Context, id string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.byID[id]; ok {
		cp := *u
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}

func (r *fakeUserRepo) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.byID {
		if u.Email == email {
			cp := *u
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeUserRepo) FindByPhone(_ context.Context, phone string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.byID {
		if u.Phone == phone {
			cp := *u
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

type fakeProductRepo struct {
	mu   sync.Mutex
	data map[string]*domain.Product
}

func newFakeProductRepo() *fakeProductRepo {
	return &fakeProductRepo{data: map[string]*domain.Product{}}
}

func (r *fakeProductRepo) put(p *domain.Product) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *p
	r.data[p.ID] = &cp
}

func (r *fakeProductRepo) Create(_ context.Context, p *domain.Product) error {
	r.put(p)
	return nil
}

func (r *fakeProductRepo) Update(_ context.Context, p *domain.Product) error {
	r.put(p)
	return nil
}

func (r *fakeProductRepo) FindByID(_ context.Context, id string) (*domain.Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.data[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}

func (r *fakeProductRepo) FindBySlug(_ context.Context, slug string) (*domain.Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.data {
		if p.Slug == slug {
			cp := *p
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeProductRepo) List(_ context.Context, _ domain.ProductFilter) (domain.Page[domain.Product], error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.Product, 0, len(r.data))
	for _, p := range r.data {
		out = append(out, *p)
	}
	return domain.Page[domain.Product]{Items: out, Total: int64(len(out))}, nil
}

func (r *fakeProductRepo) DecreaseStock(_ context.Context, id string, qty int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.data[id]
	if !ok {
		return domain.ErrNotFound
	}
	if p.Stock < qty {
		return domain.ErrInsufficientStock
	}
	p.Stock -= qty
	return nil
}

func (r *fakeProductRepo) IncreaseStock(_ context.Context, id string, qty int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.data[id]
	if !ok {
		return domain.ErrNotFound
	}
	p.Stock += qty
	return nil
}

type fakeCartRepo struct {
	mu    sync.Mutex
	carts map[string]*domain.Cart
}

func newFakeCartRepo() *fakeCartRepo {
	return &fakeCartRepo{carts: map[string]*domain.Cart{}}
}

func (r *fakeCartRepo) Get(_ context.Context, userID string) (*domain.Cart, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.carts[userID]; ok {
		cp := *c
		cp.Items = append([]domain.CartItem(nil), c.Items...)
		return &cp, nil
	}
	return &domain.Cart{UserID: userID}, nil
}

func (r *fakeCartRepo) Save(_ context.Context, c *domain.Cart) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *c
	cp.Items = append([]domain.CartItem(nil), c.Items...)
	r.carts[c.UserID] = &cp
	return nil
}

func (r *fakeCartRepo) Delete(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.carts, userID)
	return nil
}

type fakeOrderRepo struct {
	mu   sync.Mutex
	data map[string]*domain.Order
}

func newFakeOrderRepo() *fakeOrderRepo {
	return &fakeOrderRepo{data: map[string]*domain.Order{}}
}

func (r *fakeOrderRepo) Create(_ context.Context, o *domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *o
	r.data[o.ID] = &cp
	return nil
}

func (r *fakeOrderRepo) Update(_ context.Context, o *domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *o
	r.data[o.ID] = &cp
	return nil
}

func (r *fakeOrderRepo) AnonymizeByUser(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, o := range r.data {
		if o.UserID == userID {
			o.UserID = ""
			o.ShippingAddress = nil
		}
	}
	return nil
}

func (r *fakeOrderRepo) HasPurchasedProduct(_ context.Context, userID, productID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, o := range r.data {
		if o.UserID != userID {
			continue
		}
		switch o.Status {
		case domain.OrderPaid, domain.OrderShipped, domain.OrderCompleted:
			for _, it := range o.Items {
				if it.ProductID == productID {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func (r *fakeOrderRepo) FindByID(_ context.Context, id string) (*domain.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if o, ok := r.data[id]; ok {
		cp := *o
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}

func (r *fakeOrderRepo) FindByOrderNo(_ context.Context, no string) (*domain.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, o := range r.data {
		if o.OrderNo == no {
			cp := *o
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeOrderRepo) FindByAccessToken(_ context.Context, token string) (*domain.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, o := range r.data {
		if o.AccessToken == token && token != "" {
			cp := *o
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeOrderRepo) FindExpiredPending(_ context.Context, now time.Time, limit int) ([]domain.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []domain.Order{}
	for _, o := range r.data {
		if o.Expired(now) {
			out = append(out, *o)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ExpiresAt.Before(out[j].ExpiresAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *fakeOrderRepo) List(_ context.Context, _ domain.OrderFilter) (domain.Page[domain.Order], error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.Order, 0, len(r.data))
	for _, o := range r.data {
		out = append(out, *o)
	}
	return domain.Page[domain.Order]{Items: out, Total: int64(len(out))}, nil
}

type fakePaymentRepo struct {
	mu   sync.Mutex
	data map[string]*domain.Payment
}

func newFakePaymentRepo() *fakePaymentRepo {
	return &fakePaymentRepo{data: map[string]*domain.Payment{}}
}

func (r *fakePaymentRepo) Create(_ context.Context, p *domain.Payment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *p
	r.data[p.ID] = &cp
	return nil
}

func (r *fakePaymentRepo) Update(_ context.Context, p *domain.Payment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *p
	r.data[p.ID] = &cp
	return nil
}

func (r *fakePaymentRepo) FindByID(_ context.Context, id string) (*domain.Payment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.data[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}

func (r *fakePaymentRepo) FindByProviderRef(_ context.Context, provider, ref string) (*domain.Payment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.data {
		if p.Provider == provider && p.ProviderRef == ref {
			cp := *p
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

type fakeCouponRepo struct {
	mu          sync.Mutex
	byCode      map[string]*domain.Coupon
	redemptions []domain.CouponRedemption
}

func newFakeCouponRepo() *fakeCouponRepo {
	return &fakeCouponRepo{byCode: map[string]*domain.Coupon{}}
}

func (r *fakeCouponRepo) put(c *domain.Coupon) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *c
	r.byCode[cp.Code] = &cp
}

func (r *fakeCouponRepo) Create(_ context.Context, c *domain.Coupon) error {
	r.put(c)
	return nil
}

func (r *fakeCouponRepo) Update(_ context.Context, c *domain.Coupon) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for code, existing := range r.byCode {
		if existing.ID == c.ID {
			cp := *c
			r.byCode[code] = &cp
			return nil
		}
	}
	return domain.ErrNotFound
}

func (r *fakeCouponRepo) FindByID(_ context.Context, id string) (*domain.Coupon, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.byCode {
		if c.ID == id {
			cp := *c
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeCouponRepo) FindByCode(_ context.Context, code string) (*domain.Coupon, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.byCode[code]; ok {
		cp := *c
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}

func (r *fakeCouponRepo) List(_ context.Context, _ domain.CouponFilter) (domain.Page[domain.Coupon], error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.Coupon, 0, len(r.byCode))
	for _, c := range r.byCode {
		out = append(out, *c)
	}
	return domain.Page[domain.Coupon]{Items: out, Total: int64(len(out)), Page: 1, PageSize: len(out)}, nil
}

func (r *fakeCouponRepo) IncrementUsage(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.byCode {
		if c.ID == id {
			if c.UsageLimit > 0 && c.UsedCount >= c.UsageLimit {
				return domain.ErrCouponExhausted
			}
			c.UsedCount++
			return nil
		}
	}
	return domain.ErrNotFound
}

func (r *fakeCouponRepo) CountRedemptions(_ context.Context, couponID, userID string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var count int64
	for _, red := range r.redemptions {
		if red.CouponID == couponID && red.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (r *fakeCouponRepo) CreateRedemption(_ context.Context, red *domain.CouponRedemption) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.redemptions = append(r.redemptions, *red)
	return nil
}

func (r *fakeCouponRepo) ListRedemptions(_ context.Context, couponID string, _ domain.CouponFilter) (domain.Page[domain.CouponRedemption], error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.CouponRedemption, 0)
	for _, red := range r.redemptions {
		if red.CouponID == couponID {
			out = append(out, red)
		}
	}
	return domain.Page[domain.CouponRedemption]{Items: out, Total: int64(len(out)), Page: 1, PageSize: len(out)}, nil
}

type fakeVariantRepo struct {
	mu   sync.Mutex
	data map[string]*domain.Variant
}

func newFakeVariantRepo() *fakeVariantRepo {
	return &fakeVariantRepo{data: map[string]*domain.Variant{}}
}

func (r *fakeVariantRepo) put(v *domain.Variant) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *v
	r.data[v.ID] = &cp
}

func (r *fakeVariantRepo) Create(_ context.Context, v *domain.Variant) error {
	r.put(v)
	return nil
}

func (r *fakeVariantRepo) Update(_ context.Context, v *domain.Variant) error {
	r.put(v)
	return nil
}

func (r *fakeVariantRepo) FindByID(_ context.Context, id string) (*domain.Variant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if v, ok := r.data[id]; ok {
		cp := *v
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}

func (r *fakeVariantRepo) ListByProduct(_ context.Context, productID string) ([]domain.Variant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []domain.Variant{}
	for _, v := range r.data {
		if v.ProductID == productID {
			out = append(out, *v)
		}
	}
	return out, nil
}

func (r *fakeVariantRepo) DecreaseStock(_ context.Context, id string, qty int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.data[id]
	if !ok {
		return domain.ErrNotFound
	}
	if v.Stock < qty {
		return domain.ErrInsufficientStock
	}
	v.Stock -= qty
	return nil
}

func (r *fakeVariantRepo) IncreaseStock(_ context.Context, id string, qty int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.data[id]
	if !ok {
		return domain.ErrNotFound
	}
	v.Stock += qty
	return nil
}

type fakeAddressRepo struct {
	mu   sync.Mutex
	data map[string]*domain.Address
}

func newFakeAddressRepo() *fakeAddressRepo {
	return &fakeAddressRepo{data: map[string]*domain.Address{}}
}

func (r *fakeAddressRepo) put(a *domain.Address) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *a
	r.data[a.ID] = &cp
}

func (r *fakeAddressRepo) Create(_ context.Context, a *domain.Address) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *a
	r.data[a.ID] = &cp
	return nil
}

func (r *fakeAddressRepo) Update(_ context.Context, a *domain.Address) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *a
	r.data[a.ID] = &cp
	return nil
}

func (r *fakeAddressRepo) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.data, id)
	return nil
}

func (r *fakeAddressRepo) FindByID(_ context.Context, id string) (*domain.Address, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if a, ok := r.data[id]; ok {
		cp := *a
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}

func (r *fakeAddressRepo) ListByUser(_ context.Context, userID string) ([]domain.Address, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []domain.Address{}
	for _, a := range r.data {
		if a.UserID == userID {
			out = append(out, *a)
		}
	}
	return out, nil
}

func (r *fakeAddressRepo) ClearDefault(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, a := range r.data {
		if a.UserID == userID {
			a.Default = false
		}
	}
	return nil
}

func (r *fakeAddressRepo) DeleteByUser(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, a := range r.data {
		if a.UserID == userID {
			delete(r.data, id)
		}
	}
	return nil
}

type fakeShippingRepo struct {
	mu   sync.Mutex
	data map[string]*domain.ShippingMethod
}

func newFakeShippingRepo() *fakeShippingRepo {
	return &fakeShippingRepo{data: map[string]*domain.ShippingMethod{}}
}

func (r *fakeShippingRepo) put(m *domain.ShippingMethod) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *m
	r.data[m.ID] = &cp
}

func (r *fakeShippingRepo) Create(_ context.Context, m *domain.ShippingMethod) error {
	r.put(m)
	return nil
}

func (r *fakeShippingRepo) Update(_ context.Context, m *domain.ShippingMethod) error {
	r.put(m)
	return nil
}

func (r *fakeShippingRepo) FindByID(_ context.Context, id string) (*domain.ShippingMethod, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if m, ok := r.data[id]; ok {
		cp := *m
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}

func (r *fakeShippingRepo) List(_ context.Context, activeOnly bool) ([]domain.ShippingMethod, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []domain.ShippingMethod{}
	for _, m := range r.data {
		if activeOnly && !m.Active {
			continue
		}
		out = append(out, *m)
	}
	return out, nil
}

func (r *fakeShippingRepo) Default(_ context.Context) (*domain.ShippingMethod, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, m := range r.data {
		if m.Active {
			cp := *m
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

type fakeZoneRepo struct {
	mu    sync.Mutex
	zones map[string]*domain.ShippingZone
	rates map[string]*domain.ShippingRate
}

func newFakeZoneRepo() *fakeZoneRepo {
	return &fakeZoneRepo{zones: map[string]*domain.ShippingZone{}, rates: map[string]*domain.ShippingRate{}}
}

func rateKey(zoneID, methodID string) string { return zoneID + "|" + methodID }

func (r *fakeZoneRepo) Create(_ context.Context, z *domain.ShippingZone) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *z
	r.zones[z.ID] = &cp
	return nil
}

func (r *fakeZoneRepo) Update(_ context.Context, z *domain.ShippingZone) error {
	return r.Create(context.Background(), z)
}

func (r *fakeZoneRepo) List(_ context.Context, activeOnly bool) ([]domain.ShippingZone, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []domain.ShippingZone{}
	for _, z := range r.zones {
		if activeOnly && !z.Active {
			continue
		}
		out = append(out, *z)
	}
	return out, nil
}

func (r *fakeZoneRepo) FindByID(_ context.Context, id string) (*domain.ShippingZone, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if z, ok := r.zones[id]; ok {
		cp := *z
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}

func (r *fakeZoneRepo) FindByProvince(_ context.Context, province string) (*domain.ShippingZone, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, z := range r.zones {
		if z.Active && z.Matches(province) {
			cp := *z
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeZoneRepo) UpsertRate(_ context.Context, rate *domain.ShippingRate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *rate
	r.rates[rateKey(rate.ZoneID, rate.MethodID)] = &cp
	return nil
}

func (r *fakeZoneRepo) FindRate(_ context.Context, zoneID, methodID string) (*domain.ShippingRate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rate, ok := r.rates[rateKey(zoneID, methodID)]; ok {
		cp := *rate
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}

type fakeCache struct {
	mu   sync.Mutex
	data map[string]string
}

func newFakeCache() *fakeCache { return &fakeCache{data: map[string]string{}} }

func (c *fakeCache) Get(_ context.Context, key string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if v, ok := c.data[key]; ok {
		return v, nil
	}
	return "", port.ErrCacheMiss
}

func (c *fakeCache) Set(_ context.Context, key, value string, _ time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
	return nil
}

func (c *fakeCache) Delete(_ context.Context, keys ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, k := range keys {
		delete(c.data, k)
	}
	return nil
}

func (c *fakeCache) GetJSON(ctx context.Context, key string, dest any) error {
	raw, err := c.Get(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), dest)
}

func (c *fakeCache) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.Set(ctx, key, string(raw), ttl)
}

func (c *fakeCache) Incr(_ context.Context, key string, delta int64) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var n int64
	if v, ok := c.data[key]; ok {
		n, _ = strconv.ParseInt(v, 10, 64)
	}
	n += delta
	c.data[key] = strconv.FormatInt(n, 10)
	return n, nil
}

func (c *fakeCache) Expire(_ context.Context, _ string, _ time.Duration) error { return nil }

type fakeLock struct{ released bool }

func (l *fakeLock) Release(_ context.Context) error { l.released = true; return nil }

type fakeLocker struct {
	mu    sync.Mutex
	locks map[string]bool
}

func newFakeLocker() *fakeLocker { return &fakeLocker{locks: map[string]bool{}} }

func (l *fakeLocker) Acquire(_ context.Context, key string, _ time.Duration, _ time.Duration) (port.Lock, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.locks[key] {
		return nil, domain.ErrLockUnavailable
	}
	l.locks[key] = true
	return &releaseFn{fn: func() { l.mu.Lock(); delete(l.locks, key); l.mu.Unlock() }}, nil
}

type releaseFn struct{ fn func() }

func (r *releaseFn) Release(_ context.Context) error { r.fn(); return nil }

type fakeTx struct{}

func (fakeTx) WithinTx(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

type fakeBus struct {
	mu     sync.Mutex
	events []port.Event
}

func newFakeBus() *fakeBus { return &fakeBus{} }

func (b *fakeBus) Publish(_ context.Context, evt port.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, evt)
	return nil
}

func (b *fakeBus) Subscribe(string, string, string, port.EventHandler) error { return nil }
func (b *fakeBus) Close() error                                              { return nil }

func (b *fakeBus) subjects() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]string, 0, len(b.events))
	for _, e := range b.events {
		out = append(out, e.Subject)
	}
	return out
}

// fakeOutbox records events enqueued inside a transaction. In tests the
// transaction is a pass-through, so this doubles as the published-event log.
type fakeOutbox struct {
	mu     sync.Mutex
	events []port.Event
}

func newFakeOutbox() *fakeOutbox { return &fakeOutbox{} }

func (o *fakeOutbox) Enqueue(_ context.Context, evt port.Event) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.events = append(o.events, evt)
	return nil
}

func (o *fakeOutbox) Claim(context.Context, int, time.Duration) ([]port.OutboxMessage, error) {
	return nil, nil
}
func (o *fakeOutbox) MarkPublished(context.Context, string) error { return nil }
func (o *fakeOutbox) MarkFailed(context.Context, port.OutboxMessage, string, time.Time) error {
	return nil
}
func (o *fakeOutbox) Reclaim(context.Context, time.Time) (int64, error) { return 0, nil }

func (o *fakeOutbox) subjects() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make([]string, 0, len(o.events))
	for _, e := range o.events {
		out = append(out, e.Subject)
	}
	return out
}

type seqIDs struct {
	mu sync.Mutex
	n  int
}

func (g *seqIDs) NewID() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.n++
	return "id-" + strconv.Itoa(g.n)
}

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

type nopLogger struct{}

func (nopLogger) Debug(string, ...any)    {}
func (nopLogger) Info(string, ...any)     {}
func (nopLogger) Warn(string, ...any)     {}
func (nopLogger) Error(string, ...any)    {}
func (nopLogger) With(...any) port.Logger { return nopLogger{} }

type fakeHasher struct{}

func (fakeHasher) Hash(plain string) (string, error) { return "h:" + plain, nil }
func (fakeHasher) Compare(hash, plain string) bool   { return hash == "h:"+plain }

type fakeTokens struct{}

func (fakeTokens) Issue(c port.TokenClaims) (string, error) { return "token:" + c.Subject, nil }
func (fakeTokens) Verify(string) (*port.TokenClaims, error) { return nil, errors.New("not used") }

type fakeMailer struct {
	mu   sync.Mutex
	sent []port.Email
}

func (m *fakeMailer) Send(_ context.Context, msg port.Email) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, msg)
	return nil
}

func (m *fakeMailer) last() (port.Email, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.sent) == 0 {
		return port.Email{}, false
	}
	return m.sent[len(m.sent)-1], true
}

type fakeSettingsRepo struct {
	mu     sync.Mutex
	values map[string]string
}

func (r *fakeSettingsRepo) All(_ context.Context) (map[string]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[string]string, len(r.values))
	for k, v := range r.values {
		out[k] = v
	}
	return out, nil
}

func (r *fakeSettingsRepo) Upsert(_ context.Context, values map[string]string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.values == nil {
		r.values = map[string]string{}
	}
	for k, v := range values {
		r.values[k] = v
	}
	return nil
}

// newTestSettings builds a SettingsService whose overrides are the given map
// (defaults otherwise). The clock is fixed so the cache never expires in tests.
func newTestSettings(values map[string]string) *SettingsService {
	return NewSettingsService(&fakeSettingsRepo{values: values},
		fixedClock{t: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}, nil)
}
