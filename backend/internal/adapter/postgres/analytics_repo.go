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

// LowStock lists published products and active variants at or below threshold.
func (r *AnalyticsRepository) LowStock(ctx context.Context, threshold int) ([]domain.LowStockItem, error) {
	session := r.db.session(ctx)
	out := make([]domain.LowStockItem, 0)

	var products []productModel
	if err := session.Where("status = ? AND stock <= ?", string(domain.ProductPublished), threshold).
		Order("stock asc").Limit(100).Find(&products).Error; err != nil {
		return nil, translate(err)
	}
	for i := range products {
		out = append(out, domain.LowStockItem{
			Type: "product", ID: products[i].ID, ProductID: products[i].ID,
			Title: products[i].Title, Stock: products[i].Stock,
		})
	}

	var rows []struct {
		ID        string
		ProductID string
		Name      string
		SKU       string
		Stock     int
		Title     string
	}
	if err := session.Table("product_variants AS v").
		Select("v.id, v.product_id, v.name, v.sku, v.stock, p.title").
		Joins("JOIN products AS p ON p.id = v.product_id").
		Where("v.active AND v.stock <= ?", threshold).
		Order("v.stock asc").Limit(100).
		Scan(&rows).Error; err != nil {
		return nil, translate(err)
	}
	for _, row := range rows {
		out = append(out, domain.LowStockItem{
			Type: "variant", ID: row.ID, ProductID: row.ProductID, Title: row.Title,
			VariantName: row.Name, SKU: row.SKU, Stock: row.Stock,
		})
	}
	return out, nil
}
