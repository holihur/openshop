package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

const currencyCacheKey = "catalog:currencies"

// CurrencyService resolves exchange rates and manages them for admins. Rates are
// expressed from the store base currency.
type CurrencyService struct {
	repo  port.CurrencyRepository
	base  string
	cache port.Cache
}

func NewCurrencyService(repo port.CurrencyRepository, base string, cache port.Cache) *CurrencyService {
	return &CurrencyService{repo: repo, base: strings.ToUpper(base), cache: cache}
}

func (s *CurrencyService) Base() string { return s.base }

func (s *CurrencyService) List(ctx context.Context) ([]domain.ExchangeRate, error) {
	var cached []domain.ExchangeRate
	if err := s.cache.GetJSON(ctx, currencyCacheKey, &cached); err == nil {
		return cached, nil
	}
	rates, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	_ = s.cache.SetJSON(ctx, currencyCacheKey, rates, 10*time.Minute)
	return rates, nil
}

func (s *CurrencyService) SetRate(ctx context.Context, currency string, rateMicro int64) (*domain.ExchangeRate, error) {
	code := strings.ToUpper(strings.TrimSpace(currency))
	if code == "" {
		return nil, fmt.Errorf("%w: currency is required", domain.ErrInvalidArgument)
	}
	if len(code) != 3 || !isAlphaUpper(code) {
		return nil, fmt.Errorf("%w: currency must be a 3-letter ISO code", domain.ErrInvalidArgument)
	}
	if code == s.base {
		return nil, fmt.Errorf("%w: cannot set a rate for the base currency", domain.ErrInvalidArgument)
	}
	if rateMicro <= 0 {
		return nil, fmt.Errorf("%w: rate must be positive", domain.ErrInvalidArgument)
	}
	if err := s.repo.Upsert(ctx, code, rateMicro); err != nil {
		return nil, err
	}
	_ = s.cache.Delete(ctx, currencyCacheKey)
	return s.repo.Find(ctx, code)
}

// Rate returns the micro rate from one currency to another. Only conversions
// from the base currency are supported; identity returns 1e6.
func (s *CurrencyService) Rate(ctx context.Context, from, to string) (int64, error) {
	from = strings.ToUpper(from)
	to = strings.ToUpper(to)
	if from == to {
		return 1_000_000, nil
	}
	if from != s.base {
		return 0, fmt.Errorf("%w: unsupported source currency %s", domain.ErrInvalidArgument, from)
	}
	rate, err := s.repo.Find(ctx, to)
	if err != nil {
		return 0, fmt.Errorf("%w: unsupported currency %s", domain.ErrInvalidArgument, to)
	}
	return rate.RateMicro, nil
}

var _ port.ExchangeRates = (*CurrencyService)(nil)

func isAlphaUpper(s string) bool {
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}
