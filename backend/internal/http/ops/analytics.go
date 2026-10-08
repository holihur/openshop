package ops

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/http/handler"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/response"
)

type lowStockView struct {
	Type        string `json:"type"`
	ID          string `json:"id"`
	ProductID   string `json:"productId"`
	Title       string `json:"title"`
	VariantName string `json:"variantName,omitempty"`
	SKU         string `json:"sku,omitempty"`
	Stock       int    `json:"stock"`
}

type dashboardView struct {
	RevenueCents      int64               `json:"revenueCents"`
	RevenueByCurrency map[string]int64    `json:"revenueByCurrency,omitempty"`
	LowStock          []lowStockView      `json:"lowStock"`
	PaidOrders        int64               `json:"paidOrders"`
	PendingOrders     int64               `json:"pendingOrders"`
	CancelledOrders   int64               `json:"cancelledOrders"`
	TotalOrders       int64               `json:"totalOrders"`
	TotalProducts     int64               `json:"totalProducts"`
	TotalUsers        int64               `json:"totalUsers"`
	RecentOrders      []handler.OrderView `json:"recentOrders"`
}

func toLowStockViews(items []domain.LowStockItem) []lowStockView {
	out := make([]lowStockView, 0, len(items))
	for _, it := range items {
		out = append(out, lowStockView{
			Type: it.Type, ID: it.ID, ProductID: it.ProductID, Title: it.Title,
			VariantName: it.VariantName, SKU: it.SKU, Stock: it.Stock,
		})
	}
	return out
}

// Summary returns the per-module counters shown at the top of each console
// page, bucketed into day/week/fortnight/month windows.
func (h *Handler) Summary(c *gin.Context) {
	summary, err := h.Analytics.Summary(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, summary)
}

func (h *Handler) Dashboard(c *gin.Context) {
	dashboard, err := h.Analytics.Dashboard(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	recent := make([]handler.OrderView, 0, len(dashboard.RecentOrders))
	for _, o := range dashboard.RecentOrders {
		recent = append(recent, handler.ToOrderView(o))
	}
	response.OK(c, dashboardView{
		RevenueCents:      dashboard.RevenueCents,
		RevenueByCurrency: dashboard.RevenueByCurrency,
		LowStock:          toLowStockViews(dashboard.LowStock),
		PaidOrders:        dashboard.PaidOrders,
		PendingOrders:     dashboard.PendingOrders,
		CancelledOrders:   dashboard.CancelledOrders,
		TotalOrders:       dashboard.TotalOrders,
		TotalProducts:     dashboard.TotalProducts,
		TotalUsers:        dashboard.TotalUsers,
		RecentOrders:      recent,
	})
}

func (h *Handler) LowStock(c *gin.Context) {
	items, err := h.Analytics.LowStock(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toLowStockViews(items))
}
