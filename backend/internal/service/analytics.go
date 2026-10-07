package service

import (
	"context"
	"time"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

const dashboardCacheKey = "admin:dashboard"

// AnalyticsService serves the merchant dashboard, briefly cached so repeated
// refreshes do not hammer the database.
type AnalyticsService struct {
	repo     port.AnalyticsRepository
	cache    port.Cache
	base     string
	settings *SettingsService
}

func NewAnalyticsService(repo port.AnalyticsRepository, cache port.Cache, base string, settings *SettingsService) *AnalyticsService {
	return &AnalyticsService{repo: repo, cache: cache, base: base, settings: settings}
}

func (s *AnalyticsService) Dashboard(ctx context.Context) (domain.Dashboard, error) {
	var cached domain.Dashboard
	if err := s.cache.GetJSON(ctx, dashboardCacheKey, &cached); err == nil {
		return cached, nil
	}
	dashboard, err := s.repo.Dashboard(ctx)
	if err != nil {
		return domain.Dashboard{}, err
	}
	dashboard.RevenueCents = dashboard.RevenueByCurrency[s.base]
	if items, err := s.repo.LowStock(ctx, s.lowStockThreshold(ctx)); err == nil {
		dashboard.LowStock = items
	}
	_ = s.cache.SetJSON(ctx, dashboardCacheKey, dashboard, 30*time.Second)
	return dashboard, nil
}

func (s *AnalyticsService) LowStock(ctx context.Context) ([]domain.LowStockItem, error) {
	return s.repo.LowStock(ctx, s.lowStockThreshold(ctx))
}

func (s *AnalyticsService) lowStockThreshold(ctx context.Context) int {
	if s.settings == nil {
		return 0
	}
	return s.settings.Int(ctx, "inventory.low_stock_threshold")
}
