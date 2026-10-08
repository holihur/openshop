package postgres

import (
	"context"
	"time"

	"github.com/holihur/openshop/internal/domain"
)

// periodRow is the shape of the four FILTER columns every window query returns.
type periodRow struct {
	Today     int64
	Week      int64
	Fortnight int64
	Month     int64
}

func (p periodRow) counts() domain.PeriodCounts {
	return domain.PeriodCounts{Today: p.Today, Week: p.Week, Fortnight: p.Fortnight, Month: p.Month}
}

// windowArgs returns the four lower bounds in the order the queries bind them:
// start of today (UTC), then 7, 14 and 30 days ago.
func windowArgs(now time.Time) (time.Time, time.Time, time.Time, time.Time) {
	now = now.UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return today, now.AddDate(0, 0, -7), now.AddDate(0, 0, -14), now.AddDate(0, 0, -30)
}

// Summary aggregates the console headers. Each module is one query using
// count(*) FILTER so a single scan produces all four windows; the SQL fragments
// are constants in this file, and every value is bound as a parameter.
func (r *AnalyticsRepository) Summary(ctx context.Context, now time.Time, lowStockThreshold int) (domain.OpsSummary, error) {
	session := r.db.session(ctx)
	today, week, fortnight, month := windowArgs(now)
	out := domain.OpsSummary{GeneratedAt: now.UTC()}

	// --- orders ------------------------------------------------------------
	var statusRows []struct {
		Status string
		Count  int64
	}
	if err := session.Model(&orderModel{}).
		Select("status, count(*) as count").Group("status").Scan(&statusRows).Error; err != nil {
		return out, translate(err)
	}
	out.Orders.ByStatus = make(map[string]int64, len(statusRows))
	for _, row := range statusRows {
		out.Orders.ByStatus[row.Status] = row.Count
		out.Orders.Total += row.Count
	}

	var orderWindows struct {
		CreatedToday, CreatedWeek, CreatedFortnight, CreatedMonth int64
		PaidToday, PaidWeek, PaidFortnight, PaidMonth             int64
		RevenueToday, RevenueWeek, RevenueFortnight, RevenueMonth int64
	}
	if err := session.Raw(`
		SELECT
		  count(*) FILTER (WHERE created_at >= ?) AS created_today,
		  count(*) FILTER (WHERE created_at >= ?) AS created_week,
		  count(*) FILTER (WHERE created_at >= ?) AS created_fortnight,
		  count(*) FILTER (WHERE created_at >= ?) AS created_month,
		  count(*) FILTER (WHERE created_at >= ? AND status IN ('paid','shipped','completed')) AS paid_today,
		  count(*) FILTER (WHERE created_at >= ? AND status IN ('paid','shipped','completed')) AS paid_week,
		  count(*) FILTER (WHERE created_at >= ? AND status IN ('paid','shipped','completed')) AS paid_fortnight,
		  count(*) FILTER (WHERE created_at >= ? AND status IN ('paid','shipped','completed')) AS paid_month,
		  coalesce(sum(total_cents) FILTER (WHERE created_at >= ? AND status IN ('paid','shipped','completed')), 0) AS revenue_today,
		  coalesce(sum(total_cents) FILTER (WHERE created_at >= ? AND status IN ('paid','shipped','completed')), 0) AS revenue_week,
		  coalesce(sum(total_cents) FILTER (WHERE created_at >= ? AND status IN ('paid','shipped','completed')), 0) AS revenue_fortnight,
		  coalesce(sum(total_cents) FILTER (WHERE created_at >= ? AND status IN ('paid','shipped','completed')), 0) AS revenue_month
		FROM orders`,
		today, week, fortnight, month,
		today, week, fortnight, month,
		today, week, fortnight, month,
	).Scan(&orderWindows).Error; err != nil {
		return out, translate(err)
	}
	out.Orders.Created = domain.PeriodCounts{
		Today: orderWindows.CreatedToday, Week: orderWindows.CreatedWeek,
		Fortnight: orderWindows.CreatedFortnight, Month: orderWindows.CreatedMonth,
	}
	out.Orders.Paid = domain.PeriodCounts{
		Today: orderWindows.PaidToday, Week: orderWindows.PaidWeek,
		Fortnight: orderWindows.PaidFortnight, Month: orderWindows.PaidMonth,
	}
	out.Orders.RevenueCents = domain.PeriodCounts{
		Today: orderWindows.RevenueToday, Week: orderWindows.RevenueWeek,
		Fortnight: orderWindows.RevenueFortnight, Month: orderWindows.RevenueMonth,
	}

	// --- customers ---------------------------------------------------------
	if err := session.Model(&userModel{}).Where("role = ?", string(domain.RoleCustomer)).
		Count(&out.Customers.Total).Error; err != nil {
		return out, translate(err)
	}
	if err := session.Model(&userModel{}).Where("status = ?", string(domain.UserDisabled)).
		Count(&out.Customers.Disabled).Error; err != nil {
		return out, translate(err)
	}
	var custWindows periodRow
	if err := session.Raw(`
		SELECT
		  count(*) FILTER (WHERE created_at >= ?) AS today,
		  count(*) FILTER (WHERE created_at >= ?) AS week,
		  count(*) FILTER (WHERE created_at >= ?) AS fortnight,
		  count(*) FILTER (WHERE created_at >= ?) AS month
		FROM users WHERE role = 'customer'`,
		today, week, fortnight, month).Scan(&custWindows).Error; err != nil {
		return out, translate(err)
	}
	out.Customers.New = custWindows.counts()

	// --- products (a snapshot; inventory is not a window) ------------------
	if err := session.Model(&productModel{}).Count(&out.Products.Total).Error; err != nil {
		return out, translate(err)
	}
	if err := session.Model(&productModel{}).Where("status = ?", string(domain.ProductPublished)).
		Count(&out.Products.Published).Error; err != nil {
		return out, translate(err)
	}
	if err := session.Model(&productModel{}).Where("status = ?", string(domain.ProductDraft)).
		Count(&out.Products.Draft).Error; err != nil {
		return out, translate(err)
	}
	if err := session.Model(&productModel{}).Where("status = ?", string(domain.ProductPublished)).
		Where("stock <= 0").Count(&out.Products.OutOfStock).Error; err != nil {
		return out, translate(err)
	}
	if err := session.Model(&productModel{}).Where("status = ?", string(domain.ProductPublished)).
		Where("stock > 0 AND stock <= ?", lowStockThreshold).
		Count(&out.Products.LowStock).Error; err != nil {
		return out, translate(err)
	}

	// --- coupons -----------------------------------------------------------
	if err := session.Model(&couponModel{}).Count(&out.Coupons.Total).Error; err != nil {
		return out, translate(err)
	}
	if err := session.Model(&couponModel{}).Where("active").Count(&out.Coupons.Active).Error; err != nil {
		return out, translate(err)
	}
	var couponWindows periodRow
	if err := session.Raw(`
		SELECT
		  count(*) FILTER (WHERE created_at >= ?) AS today,
		  count(*) FILTER (WHERE created_at >= ?) AS week,
		  count(*) FILTER (WHERE created_at >= ?) AS fortnight,
		  count(*) FILTER (WHERE created_at >= ?) AS month
		FROM coupon_redemptions`,
		today, week, fortnight, month).Scan(&couponWindows).Error; err != nil {
		return out, translate(err)
	}
	out.Coupons.Redemptions = couponWindows.counts()

	// --- tickets -----------------------------------------------------------
	var ticketRows []struct {
		Status string
		Count  int64
	}
	if err := session.Model(&ticketModel{}).
		Select("status, count(*) as count").Group("status").Scan(&ticketRows).Error; err != nil {
		return out, translate(err)
	}
	for _, row := range ticketRows {
		switch row.Status {
		case "open":
			out.Tickets.Open = row.Count
		case "pending":
			out.Tickets.Pending = row.Count
		}
	}
	if err := session.Model(&ticketModel{}).
		Where("assignee_id IS NULL AND status IN ?", []string{"open", "pending"}).
		Count(&out.Tickets.Unassigned).Error; err != nil {
		return out, translate(err)
	}
	var ticketWindows periodRow
	if err := session.Raw(`
		SELECT
		  count(*) FILTER (WHERE created_at >= ?) AS today,
		  count(*) FILTER (WHERE created_at >= ?) AS week,
		  count(*) FILTER (WHERE created_at >= ?) AS fortnight,
		  count(*) FILTER (WHERE created_at >= ?) AS month
		FROM tickets`,
		today, week, fortnight, month).Scan(&ticketWindows).Error; err != nil {
		return out, translate(err)
	}
	out.Tickets.Created = ticketWindows.counts()

	// --- returns -----------------------------------------------------------
	var returnRows []struct {
		Status string
		Count  int64
	}
	if err := session.Model(&returnModel{}).
		Select("status, count(*) as count").Group("status").Scan(&returnRows).Error; err != nil {
		return out, translate(err)
	}
	for _, row := range returnRows {
		switch row.Status {
		case "requested":
			out.Returns.Requested = row.Count
		case "approved":
			out.Returns.Approved = row.Count
		}
	}
	var returnWindows periodRow
	if err := session.Raw(`
		SELECT
		  count(*) FILTER (WHERE created_at >= ?) AS today,
		  count(*) FILTER (WHERE created_at >= ?) AS week,
		  count(*) FILTER (WHERE created_at >= ?) AS fortnight,
		  count(*) FILTER (WHERE created_at >= ?) AS month
		FROM return_requests`,
		today, week, fortnight, month).Scan(&returnWindows).Error; err != nil {
		return out, translate(err)
	}
	out.Returns.Created = returnWindows.counts()

	// --- withdrawals -------------------------------------------------------
	var withdrawalRows []struct {
		Status string
		Count  int64
	}
	if err := session.Model(&withdrawalModel{}).
		Select("status, count(*) as count").Group("status").Scan(&withdrawalRows).Error; err != nil {
		return out, translate(err)
	}
	for _, row := range withdrawalRows {
		switch row.Status {
		case "requested":
			out.Withdrawals.Requested = row.Count
		case "approved":
			out.Withdrawals.Approved = row.Count
		case "paid":
			out.Withdrawals.Paid = row.Count
		}
	}
	var withdrawalWindows periodRow
	if err := session.Raw(`
		SELECT
		  count(*) FILTER (WHERE created_at >= ?) AS today,
		  count(*) FILTER (WHERE created_at >= ?) AS week,
		  count(*) FILTER (WHERE created_at >= ?) AS fortnight,
		  count(*) FILTER (WHERE created_at >= ?) AS month
		FROM withdrawals`,
		today, week, fortnight, month).Scan(&withdrawalWindows).Error; err != nil {
		return out, translate(err)
	}
	out.Withdrawals.Created = withdrawalWindows.counts()

	// --- reviews -----------------------------------------------------------
	if err := session.Model(&reviewModel{}).Count(&out.Reviews.Total).Error; err != nil {
		return out, translate(err)
	}
	var reviewWindows periodRow
	if err := session.Raw(`
		SELECT
		  count(*) FILTER (WHERE created_at >= ?) AS today,
		  count(*) FILTER (WHERE created_at >= ?) AS week,
		  count(*) FILTER (WHERE created_at >= ?) AS fortnight,
		  count(*) FILTER (WHERE created_at >= ?) AS month
		FROM reviews`,
		today, week, fortnight, month).Scan(&reviewWindows).Error; err != nil {
		return out, translate(err)
	}
	out.Reviews.Created = reviewWindows.counts()

	// --- commissions -------------------------------------------------------
	var commissionRows []struct {
		Status string
		Count  int64
	}
	if err := session.Model(&commissionModel{}).
		Select("status, count(*) as count").Group("status").Scan(&commissionRows).Error; err != nil {
		return out, translate(err)
	}
	for _, row := range commissionRows {
		switch row.Status {
		case string(domain.CommissionPending):
			out.Commissions.Pending = row.Count
		case string(domain.CommissionApproved):
			out.Commissions.Approved = row.Count
		}
	}
	var commissionWindows periodRow
	if err := session.Raw(`
		SELECT
		  count(*) FILTER (WHERE created_at >= ?) AS today,
		  count(*) FILTER (WHERE created_at >= ?) AS week,
		  count(*) FILTER (WHERE created_at >= ?) AS fortnight,
		  count(*) FILTER (WHERE created_at >= ?) AS month
		FROM commissions`,
		today, week, fortnight, month).Scan(&commissionWindows).Error; err != nil {
		return out, translate(err)
	}
	out.Commissions.Created = commissionWindows.counts()

	return out, nil
}
