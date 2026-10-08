package postgres

import (
	"context"
	"time"

	"github.com/holihur/openshop/internal/domain"
)

// This file derives every console counter from one daily series per module.
// Building windows, period-over-period comparisons and the trend chart from the
// same buckets keeps a single definition of "today", "last 7 days" and so on,
// and is cheaper than a dozen FILTER columns per metric.
//
// The table names below are compile-time constants in this file; every value is
// bound as a parameter.

type dayCount struct {
	Day   time.Time
	Count int64
}

type dayTotal struct {
	Day   time.Time
	Total int64
}

// bounds holds the inclusive day range of each window and of the equivalent
// window immediately before it.
type bounds struct {
	today     time.Time
	yesterday time.Time
	week      time.Time
	prevWeek  time.Time
	fortnight time.Time
	prevFortn time.Time
	month     time.Time
	prevMonth time.Time
	// days is how far back a series has to reach for the month-over-month
	// comparison (60 days including today).
	days int
}

func newBounds(now time.Time) bounds {
	now = now.UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	day := func(n int) time.Time { return today.AddDate(0, 0, -n) }
	return bounds{
		today:     today,
		yesterday: day(1),
		week:      day(6),  // last 7 days, today inclusive
		prevWeek:  day(13), // the 7 days before that
		fortnight: day(13),
		prevFortn: day(27),
		month:     day(29),
		prevMonth: day(59),
		days:      60,
	}
}

// sum counts the buckets in [from, to] inclusive.
func sum(counts map[time.Time]int64, from, to time.Time) int64 {
	var total int64
	for day := from; !day.After(to); day = day.AddDate(0, 0, 1) {
		total += counts[day]
	}
	return total
}

func at(counts map[time.Time]int64, day time.Time) int64 { return counts[day] }

// metric assembles the current and previous windows for one counter.
func metric(counts map[time.Time]int64, b bounds) domain.PeriodMetric {
	current := domain.PeriodCounts{
		Today:     at(counts, b.today),
		Yesterday: at(counts, b.yesterday),
		Week:      sum(counts, b.week, b.today),
		Fortnight: sum(counts, b.fortnight, b.today),
		Month:     sum(counts, b.month, b.today),
	}
	previous := domain.PeriodCounts{
		// The previous "today" is the whole previous day.
		Today:     at(counts, b.yesterday),
		Yesterday: at(counts, b.yesterday.AddDate(0, 0, -1)),
		Week:      sum(counts, b.prevWeek, b.week.AddDate(0, 0, -1)),
		Fortnight: sum(counts, b.prevFortn, b.fortnight.AddDate(0, 0, -1)),
		Month:     sum(counts, b.prevMonth, b.month.AddDate(0, 0, -1)),
	}
	return domain.PeriodMetric{Current: current, Previous: previous}
}

// series renders the last n days (oldest first) for the trend chart.
func series(counts map[time.Time]int64, b bounds, days int) []domain.DailyPoint {
	out := make([]domain.DailyPoint, 0, days)
	start := b.today.AddDate(0, 0, -(days - 1))
	for day := start; !day.After(b.today); day = day.AddDate(0, 0, 1) {
		out = append(out, domain.DailyPoint{
			Date:  day.Format("2006-01-02"),
			Value: counts[day],
		})
	}
	return out
}

// Summary aggregates the console headers from a daily series per module.
func (r *AnalyticsRepository) Summary(ctx context.Context, now time.Time, lowStockThreshold int) (domain.OpsSummary, error) {
	session := r.db.session(ctx)
	b := newBounds(now)
	since := b.today.AddDate(0, 0, -b.days)
	out := domain.OpsSummary{GeneratedAt: now.UTC()}

	// --- orders: status snapshot + created/paid/revenue --------------------
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

	created, err := r.dailyOrderCounts(ctx, since, "")
	if err != nil {
		return out, err
	}
	paid, err := r.dailyOrderCounts(ctx, since,
		"status IN ('paid','shipped','completed')")
	if err != nil {
		return out, err
	}
	revenue, err := r.dailyOrderRevenue(ctx, since)
	if err != nil {
		return out, err
	}
	out.Orders.Created = metric(created, b)
	out.Orders.Paid = metric(paid, b)
	out.Orders.RevenueCents = metric(revenue, b)
	out.Orders.Series = series(created, b, 30)
	out.Orders.RevenueSeries = series(revenue, b, 30)

	// --- customers ---------------------------------------------------------
	if err := session.Model(&userModel{}).Where("role = ?", string(domain.RoleCustomer)).
		Count(&out.Customers.Total).Error; err != nil {
		return out, translate(err)
	}
	if err := session.Model(&userModel{}).Where("status = ?", string(domain.UserDisabled)).
		Count(&out.Customers.Disabled).Error; err != nil {
		return out, translate(err)
	}
	signups, err := r.dailyTableCounts(ctx, "users", "role = 'customer'", since)
	if err != nil {
		return out, err
	}
	out.Customers.New = metric(signups, b)
	out.Customers.Series = series(signups, b, 30)

	// --- products (an inventory snapshot, not a window) --------------------
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
	redemptions, err := r.dailyTableCounts(ctx, "coupon_redemptions", "", since)
	if err != nil {
		return out, err
	}
	out.Coupons.Redemptions = metric(redemptions, b)
	out.Coupons.Series = series(redemptions, b, 30)

	// --- tickets -----------------------------------------------------------
	ticketCounts, err := r.statusCounts(ctx, &ticketModel{})
	if err != nil {
		return out, err
	}
	out.Tickets.Open = ticketCounts["open"]
	out.Tickets.Pending = ticketCounts["pending"]
	if err := session.Model(&ticketModel{}).
		Where("assignee_id IS NULL AND status IN ?", []string{"open", "pending"}).
		Count(&out.Tickets.Unassigned).Error; err != nil {
		return out, translate(err)
	}
	tickets, err := r.dailyTableCounts(ctx, "tickets", "", since)
	if err != nil {
		return out, err
	}
	out.Tickets.Created = metric(tickets, b)
	out.Tickets.Series = series(tickets, b, 30)

	// --- returns -----------------------------------------------------------
	returnCounts, err := r.statusCounts(ctx, &returnModel{})
	if err != nil {
		return out, err
	}
	out.Returns.Requested = returnCounts["requested"]
	out.Returns.Approved = returnCounts["approved"]
	returns, err := r.dailyTableCounts(ctx, "return_requests", "", since)
	if err != nil {
		return out, err
	}
	out.Returns.Created = metric(returns, b)
	out.Returns.Series = series(returns, b, 30)

	// --- withdrawals -------------------------------------------------------
	withdrawalCounts, err := r.statusCounts(ctx, &withdrawalModel{})
	if err != nil {
		return out, err
	}
	out.Withdrawals.Requested = withdrawalCounts["requested"]
	out.Withdrawals.Approved = withdrawalCounts["approved"]
	out.Withdrawals.Paid = withdrawalCounts["paid"]
	withdrawals, err := r.dailyTableCounts(ctx, "withdrawals", "", since)
	if err != nil {
		return out, err
	}
	out.Withdrawals.Created = metric(withdrawals, b)
	out.Withdrawals.Series = series(withdrawals, b, 30)

	// --- reviews -----------------------------------------------------------
	if err := session.Model(&reviewModel{}).Count(&out.Reviews.Total).Error; err != nil {
		return out, translate(err)
	}
	reviews, err := r.dailyTableCounts(ctx, "reviews", "", since)
	if err != nil {
		return out, err
	}
	out.Reviews.Created = metric(reviews, b)
	out.Reviews.Series = series(reviews, b, 30)

	// --- commissions -------------------------------------------------------
	commissionCounts, err := r.statusCounts(ctx, &commissionModel{})
	if err != nil {
		return out, err
	}
	out.Commissions.Pending = commissionCounts[string(domain.CommissionPending)]
	out.Commissions.Approved = commissionCounts[string(domain.CommissionApproved)]
	commissions, err := r.dailyTableCounts(ctx, "commissions", "", since)
	if err != nil {
		return out, err
	}
	out.Commissions.Created = metric(commissions, b)
	out.Commissions.Series = series(commissions, b, 30)

	return out, nil
}

// statusCounts groups a table by status.
func (r *AnalyticsRepository) statusCounts(ctx context.Context, model any) (map[string]int64, error) {
	var rows []struct {
		Status string
		Count  int64
	}
	if err := r.db.session(ctx).Model(model).
		Select("status, count(*) as count").Group("status").Scan(&rows).Error; err != nil {
		return nil, translate(err)
	}
	out := make(map[string]int64, len(rows))
	for _, row := range rows {
		out[row.Status] = row.Count
	}
	return out, nil
}

// dailyOrderCounts buckets orders per day, newest rows included in today.
func (r *AnalyticsRepository) dailyOrderCounts(ctx context.Context, since time.Time, extraWhere string) (map[time.Time]int64, error) {
	q := `SELECT date_trunc('day', created_at AT TIME ZONE 'UTC') AS day, count(*) AS count
	      FROM orders WHERE created_at >= ?`
	args := []any{since}
	if extraWhere != "" {
		q += " AND " + extraWhere
	}
	q += " GROUP BY 1"

	var rows []dayCount
	if err := r.db.session(ctx).Raw(q, args...).Scan(&rows).Error; err != nil {
		return nil, translate(err)
	}
	return toDayMap(rows), nil
}

func (r *AnalyticsRepository) dailyOrderRevenue(ctx context.Context, since time.Time) (map[time.Time]int64, error) {
	var rows []dayTotal
	if err := r.db.session(ctx).Raw(`
		SELECT date_trunc('day', created_at AT TIME ZONE 'UTC') AS day,
		       coalesce(sum(total_cents), 0) AS total
		FROM orders
		WHERE created_at >= ? AND status IN ('paid','shipped','completed')
		GROUP BY 1`, since).Scan(&rows).Error; err != nil {
		return nil, translate(err)
	}
	out := make(map[time.Time]int64, len(rows))
	for _, row := range rows {
		out[row.Day] = row.Total
	}
	return out, nil
}

// dailyTableCounts buckets a module's rows per day. table and extraWhere are
// constants defined in this file, never user input.
func (r *AnalyticsRepository) dailyTableCounts(ctx context.Context, table, extraWhere string, since time.Time) (map[time.Time]int64, error) {
	q := "SELECT date_trunc('day', created_at AT TIME ZONE 'UTC') AS day, count(*) AS count FROM " +
		table + " WHERE created_at >= ?"
	args := []any{since}
	if extraWhere != "" {
		q += " AND " + extraWhere
	}
	q += " GROUP BY 1"

	var rows []dayCount
	if err := r.db.session(ctx).Raw(q, args...).Scan(&rows).Error; err != nil {
		return nil, translate(err)
	}
	return toDayMap(rows), nil
}

func toDayMap(rows []dayCount) map[time.Time]int64 {
	out := make(map[time.Time]int64, len(rows))
	for _, row := range rows {
		out[row.Day] = row.Count
	}
	return out
}
