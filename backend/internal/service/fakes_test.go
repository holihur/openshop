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
