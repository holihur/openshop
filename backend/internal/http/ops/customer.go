package ops

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/response"
	"github.com/holihur/openshop/internal/service"
)

type customerView struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	Name          string `json:"name"`
	Role          string `json:"role"`
	Status        string `json:"status"`
	EmailVerified bool   `json:"emailVerified"`
	CreatedAt     string `json:"createdAt"`
}

func toCustomerView(u domain.User) customerView {
	return customerView{
		ID: u.ID, Email: u.Email, Phone: u.Phone, Name: u.Name,
		Role: string(u.Role), Status: string(u.Status), EmailVerified: u.EmailVerified,
		CreatedAt: u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

// ListCustomers lists customer accounts (admin only).
func (h *Handler) ListCustomers(c *gin.Context) {
	page, size := handler.ParsePage(c, 20)
	f := domain.UserFilter{Keyword: c.Query("keyword"), Page: page, PageSize: size}
	// Customer management is about shoppers, not staff.
	role := domain.RoleCustomer
	f.Role = &role
	if s := c.Query("status"); s != "" {
		st := domain.UserStatus(s)
		f.Status = &st
	}
	result, err := h.Customers.List(c.Request.Context(), f)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]customerView, 0, len(result.Items))
	for _, u := range result.Items {
		out = append(out, toCustomerView(u))
	}
	response.Paginated(c, out, result.Total, result.Page, result.PageSize)
}

// GetCustomer returns one customer account.
func (h *Handler) GetCustomer(c *gin.Context) {
	u, err := h.Customers.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toCustomerView(*u))
}

type updateCustomerRequest struct {
	Name   *string `json:"name"`
	Status *string `json:"status"`
}

// UpdateCustomer renames or enables/disables a customer account.
func (h *Handler) UpdateCustomer(c *gin.Context) {
	var req updateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, handler.WrapBind(err))
		return
	}
	in := service.UpdateCustomerInput{Name: req.Name}
	if req.Status != nil {
		st := domain.UserStatus(*req.Status)
		in.Status = &st
	}
	u, err := h.Customers.Update(c.Request.Context(), c.Param("id"), in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "customer.update", "user", u.ID, nil)
	response.OK(c, toCustomerView(*u))
}
