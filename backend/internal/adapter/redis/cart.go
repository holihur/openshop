package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// CartRepository implements port.CartRepository. Carts live in Redis so they
// survive instance restarts and are visible to every replica. Reads fall back
// to an empty cart, matching the UX expectation that an unused cart is empty.
type CartRepository struct {
	rdb *goredis.Client
	ttl time.Duration
}

func NewCartRepository(c *Client, ttl time.Duration) *CartRepository {
	if ttl <= 0 {
		ttl = 30 * 24 * time.Hour
	}
	return &CartRepository{rdb: c.Raw(), ttl: ttl}
}

func cartKey(userID string) string { return "cart:" + userID }

func (r *CartRepository) Get(ctx context.Context, userID string) (*domain.Cart, error) {
	raw, err := r.rdb.Get(ctx, cartKey(userID)).Bytes()
	if errors.Is(err, goredis.Nil) {
		return &domain.Cart{UserID: userID, Items: []domain.CartItem{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var cart domain.Cart
	if err := json.Unmarshal(raw, &cart); err != nil {
		return nil, err
	}
	cart.UserID = userID
	if cart.Items == nil {
		cart.Items = []domain.CartItem{}
	}
	return &cart, nil
}

func (r *CartRepository) Save(ctx context.Context, cart *domain.Cart) error {
	cart.UpdatedAt = time.Now().UTC()
	raw, err := json.Marshal(cart)
	if err != nil {
		return err
	}
	return r.rdb.Set(ctx, cartKey(cart.UserID), raw, r.ttl).Err()
}

func (r *CartRepository) Delete(ctx context.Context, userID string) error {
	return r.rdb.Del(ctx, cartKey(userID)).Err()
}

var _ port.CartRepository = (*CartRepository)(nil)
