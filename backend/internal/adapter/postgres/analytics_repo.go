package postgres

import (
	"context"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// AnalyticsRepository implements port.AnalyticsRepository with a handful of
// aggregate queries. It runs against the same connection pool as everything
// else, so it scales with read replicas like the rest of the store.
type AnalyticsRepository struct{ db *DB }

func NewAnalyticsRepository(db *DB) *AnalyticsRepository { return &AnalyticsRepository{db: db} }

func (r *AnalyticsRepository) Dashboard(ctx context.Context) (domain.Dashboard, error) {
	var out domain.Dashboard
	session := r.db.session(ctx)

	// Revenue is grouped by settlement currency; the service picks the base
	// currency for the headline figure.
	var revenueRows []struct {
		Currency string
		Total    int64
	}
	if err := session.Model(&orderModel{}).
		Select("currency, coalesce(sum(total_cents), 0) as total").
		Where("status IN ?", []string{
			string(domain.OrderPaid), string(domain.OrderShipped), string(domain.OrderCompleted),
		}).
		Group("currency").
		Scan(&revenueRows).Error; err != nil {
		return domain.Dashboard{}, translate(err)
	}
	out.RevenueByCurrency = make(map[string]int64, len(revenueRows))
	for _, row := range revenueRows {
		out.RevenueByCurrency[row.Currency] = row.Total
	}

	counts := map[string]int64{}
	var rows []struct {
		Status string
		Count  int64
	}
	if err := session.Model(&orderModel{}).
		Select("status, count(*) as count").Group("status").Scan(&rows).Error; err != nil {
		return domain.Dashboard{}, translate(err)
	}
	for _, row := range rows {
		counts[row.Status] = row.Count
		out.TotalOrders += row.Count
	}
	out.PaidOrders = counts[string(domain.OrderPaid)] + counts[string(domain.OrderShipped)] + counts[string(domain.OrderCompleted)]
	out.PendingOrders = counts[string(domain.OrderPendingPayment)]
	out.CancelledOrders = counts[string(domain.OrderCancelled)]

	if err := session.Model(&productModel{}).Count(&out.TotalProducts).Error; err != nil {
		return domain.Dashboard{}, translate(err)
	}
	if err := session.Model(&userModel{}).Count(&out.TotalUsers).Error; err != nil {
		return domain.Dashboard{}, translate(err)
	}

	var recent []orderModel
	if err := session.Preload("Items").Order("created_at desc").Limit(5).Find(&recent).Error; err != nil {
		return domain.Dashboard{}, translate(err)
	}
	for i := range recent {
		out.RecentOrders = append(out.RecentOrders, *toOrder(&recent[i]))
	}
	return out, nil
}

var _ port.AnalyticsRepository = (*AnalyticsRepository)(nil)
