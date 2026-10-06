package service

import (
	"context"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// AccountService implements GDPR data export and erasure.
type AccountService struct {
	users     port.UserRepository
	addresses port.AddressRepository
	wishlist  port.WishlistRepository
	reviews   port.ReviewRepository
	orders    port.OrderRepository
	auth      *AuthService
	logger    port.Logger
}

func NewAccountService(
	users port.UserRepository,
	addresses port.AddressRepository,
	wishlist port.WishlistRepository,
	reviews port.ReviewRepository,
	orders port.OrderRepository,
	auth *AuthService,
	logger port.Logger,
) *AccountService {
	return &AccountService{
		users: users, addresses: addresses, wishlist: wishlist, reviews: reviews,
		orders: orders, auth: auth, logger: logger,
	}
}

// Export is the complete data bundle for one user.
type Export struct {
	User      *domain.User     `json:"user"`
	Addresses []domain.Address `json:"addresses"`
	Orders    []domain.Order   `json:"orders"`
	Reviews   []domain.Review  `json:"reviews"`
	Wishlist  []domain.Product `json:"wishlist"`
}

func (s *AccountService) Export(ctx context.Context, userID string) (*Export, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	addresses, err := s.addresses.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	reviews, err := s.reviews.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	wishlist, err := s.wishlist.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	var orders []domain.Order
	for page := 1; ; page++ {
		result, err := s.orders.List(ctx, domain.OrderFilter{UserID: userID, Page: page, PageSize: 100})
		if err != nil {
			return nil, err
		}
		orders = append(orders, result.Items...)
		if len(result.Items) == 0 || int64(page*100) >= result.Total {
			break
		}
	}

	return &Export{
		User: user, Addresses: addresses, Orders: orders, Reviews: reviews, Wishlist: wishlist,
	}, nil
}

// Delete erases a user's personal data. Orders are anonymised rather than
// removed so financial records survive; every other personal record is deleted.
func (s *AccountService) Delete(ctx context.Context, userID string) error {
	if _, err := s.users.FindByID(ctx, userID); err != nil {
		return err
	}
	s.auth.RevokeAllSessions(ctx, userID)

	if err := s.addresses.DeleteByUser(ctx, userID); err != nil {
		return err
	}
	if err := s.wishlist.DeleteByUser(ctx, userID); err != nil {
		return err
	}
	if err := s.reviews.DeleteByUser(ctx, userID); err != nil {
		return err
	}
	if err := s.orders.AnonymizeByUser(ctx, userID); err != nil {
		return err
	}
	return s.users.Delete(ctx, userID)
}
