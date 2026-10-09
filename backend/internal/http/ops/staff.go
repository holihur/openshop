package ops

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/response"
	"github.com/holihur/openshop/internal/service"
)

type createStaffRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Name     string `json:"name"`
	Role     string `json:"role" binding:"required"`
	Password string `json:"password"`
}

type updateStaffRequest struct {
	Name   *string `json:"name"`
	Role   *string `json:"role"`
	Status *string `json:"status"`
}

type staffView struct {
	ID        string   `json:"id"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
	Role      string   `json:"role"`
	Status    string   `json:"status"`
	CreatedAt string   `json:"createdAt"`
	Perms     []string `json:"permissions"`
}

func toStaffView(u domain.User) staffView {
	perms := make([]string, 0)
	for _, p := range domain.Permissions(u.Role) {
		perms = append(perms, string(p))
	}
	return staffView{
		ID: u.ID, Email: u.Email, Name: u.Name, Role: string(u.Role),
		Status: string(u.Status), CreatedAt: u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		Perms: perms,
	}
}

// ListStaff returns the console users, never shoppers.
func (h *Handler) ListStaff(c *gin.Context) {
	page, size := handler.ParsePage(c, 20)
	filter := domain.UserFilter{Keyword: c.Query("keyword"), OpsOnly: true, Page: page, PageSize: size}
	if s := c.Query("role"); s != "" {
		role := domain.UserRole(s)
		filter.Role = &role
	}
	if s := c.Query("status"); s != "" {
		status := domain.UserStatus(s)
		filter.Status = &status
	}
	result, err := h.Staff.List(c.Request.Context(), filter)
	if err != nil {
		response.Fail(c, err)
		return
	}
	items := make([]staffView, 0, len(result.Items))
	for _, u := range result.Items {
		items = append(items, toStaffView(u))
	}
	response.Paginated(c, items, result.Total, result.Page, result.PageSize)
}

// ListRoles returns the console roles and their permissions, which is the role
// matrix the console displays.
func (h *Handler) ListRoles(c *gin.Context) {
	roles, err := h.Staff.Roles(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, roles)
}

// CreateStaff adds a console user. A generated password is returned once.
func (h *Handler) CreateStaff(c *gin.Context) {
	var req createStaffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, handler.WrapBind(err))
		return
	}
	result, err := h.Staff.Create(c.Request.Context(), service.CreateStaffInput{
		Email: req.Email, Name: req.Name, Role: domain.UserRole(req.Role), Password: req.Password,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "staff.created", "user", result.User.ID, map[string]string{"role": string(result.User.Role)})
	response.Created(c, gin.H{
		"staff":             toStaffView(*result.User),
		"generatedPassword": result.GeneratedPassword,
	})
}

// UpdateStaff changes a console user's name, role or status.
func (h *Handler) UpdateStaff(c *gin.Context) {
	var req updateStaffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, handler.WrapBind(err))
		return
	}
	in := service.UpdateStaffInput{Name: req.Name}
	if req.Role != nil {
		role := domain.UserRole(*req.Role)
		in.Role = &role
	}
	if req.Status != nil {
		status := domain.UserStatus(*req.Status)
		in.Status = &status
	}
	user, err := h.Staff.Update(c.Request.Context(), middleware.UserID(c), c.Param("id"), in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "staff.updated", "user", user.ID, map[string]string{"role": string(user.Role)})
	response.OK(c, toStaffView(*user))
}

// ResetStaffPassword issues a new password and signs the user out everywhere.
func (h *Handler) ResetStaffPassword(c *gin.Context) {
	result, err := h.Staff.ResetPassword(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "staff.password_reset", "user", result.User.ID, nil)
	response.OK(c, gin.H{
		"staff":             toStaffView(*result.User),
		"generatedPassword": result.GeneratedPassword,
	})
}
