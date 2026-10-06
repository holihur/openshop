package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/http/response"
)

type dashboardView struct {
	RevenueCents      int64            `json:"revenueCents"`
	RevenueByCurrency map[string]int64 `json:"revenueByCurrency,omitempty"`
	PaidOrders        int64            `json:"paidOrders"`
	PendingOrders     int64            `json:"pendingOrders"`
	CancelledOrders   int64            `json:"cancelledOrders"`
	TotalOrders       int64            `json:"totalOrders"`
	TotalProducts     int64            `json:"totalProducts"`
	TotalUsers        int64            `json:"totalUsers"`
	RecentOrders      []orderView      `json:"recentOrders"`
}

func (h *Handler) Dashboard(c *gin.Context) {
	dashboard, err := h.Analytics.Dashboard(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	recent := make([]orderView, 0, len(dashboard.RecentOrders))
	for _, o := range dashboard.RecentOrders {
		recent = append(recent, toOrderView(o))
	}
	response.OK(c, dashboardView{
		RevenueCents:      dashboard.RevenueCents,
		RevenueByCurrency: dashboard.RevenueByCurrency,
		PaidOrders:        dashboard.PaidOrders,
		PendingOrders:     dashboard.PendingOrders,
		CancelledOrders:   dashboard.CancelledOrders,
		TotalOrders:       dashboard.TotalOrders,
		TotalProducts:     dashboard.TotalProducts,
		TotalUsers:        dashboard.TotalUsers,
		RecentOrders:      recent,
	})
}
