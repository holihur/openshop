package service

import (
	"context"
	"time"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

const dashboardCacheKey = "admin:dashboard"

// summaryCacheKey is short-lived: the console headers are read on every page
// load but a few seconds of staleness is irrelevant for counters.
const summaryCacheKey = "admin:summary"

// AnalyticsService serves the merchant dashboard, briefly cached so repeated
// refreshes do not hammer the database.
type AnalyticsService struct {
	repo     port.AnalyticsRepository
	cache    port.Cache
	clock    port.Clock
	base     string
	settings *SettingsService
}

func NewAnalyticsService(repo port.AnalyticsRepository, cache port.Cache, clock port.Clock, base string, settings *SettingsService) *AnalyticsService {
	return &AnalyticsService{repo: repo, cache: cache, clock: clock, base: base, settings: settings}
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

// Summary returns the per-module console counters, bucketed into
// day/week/fortnight/month windows.
func (s *AnalyticsService) Summary(ctx context.Context) (domain.OpsSummary, error) {
	if s.cache != nil {
		var cached domain.OpsSummary
		if err := s.cache.GetJSON(ctx, summaryCacheKey, &cached); err == nil {
			return cached, nil
		}
	}
	summary, err := s.repo.Summary(ctx, s.clock.Now().UTC(), s.lowStockThreshold(ctx))
	if err != nil {
		return domain.OpsSummary{}, err
	}
	if s.cache != nil {
		_ = s.cache.SetJSON(ctx, summaryCacheKey, summary, 15*time.Second)
	}
	return summary, nil
}

func (s *AnalyticsService) lowStockThreshold(ctx context.Context) int {
	if s.settings == nil {
		return 0
	}
	return s.settings.Int(ctx, "inventory.low_stock_threshold")
}
