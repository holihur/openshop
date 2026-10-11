package ops

import (
	"context"
	"strconv"

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

type roleRequest struct {
	Key         string   `json:"key"`
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Permissions []string `json:"permissions"`
}

func toPermissions(values []string) []domain.Permission {
	out := make([]domain.Permission, 0, len(values))
	for _, v := range values {
		out = append(out, domain.Permission(v))
	}
	return out
}

func toRoleView(role domain.Role) gin.H {
	return gin.H{
		"key": role.Key, "name": role.Name, "description": role.Description,
		"permissions": role.Permissions, "custom": !role.Builtin,
	}
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

func (h *Handler) toStaffView(ctx context.Context, u domain.User) staffView {
	perms := make([]string, 0)
	for _, p := range h.rolePermissions(ctx, u.Role) {
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
		items = append(items, h.toStaffView(c.Request.Context(), u))
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
	result, err := h.Staff.Create(c.Request.Context(), string(middleware.Role(c)), service.CreateStaffInput{
		Email: req.Email, Name: req.Name, Role: domain.UserRole(req.Role), Password: req.Password,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "staff.created", "user", result.User.ID, map[string]string{"role": string(result.User.Role)})
	response.Created(c, gin.H{
		"staff":             h.toStaffView(c.Request.Context(), *result.User),
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
	user, err := h.Staff.Update(c.Request.Context(), middleware.UserID(c), string(middleware.Role(c)), c.Param("id"), in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "staff.updated", "user", user.ID, map[string]string{"role": string(user.Role)})
	response.OK(c, h.toStaffView(c.Request.Context(), *user))
}

// CreateRole adds an operator-defined role. The permissions it may grant are
// capped by the caller's own, so a role manager cannot escalate.
func (h *Handler) CreateRole(c *gin.Context) {
	var req roleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, handler.WrapBind(err))
		return
	}
	name := ""
	if req.Name != nil {
		name = *req.Name
	}
	description := ""
	if req.Description != nil {
		description = *req.Description
	}
	role, err := h.RoleSvc.Create(c.Request.Context(), string(middleware.Role(c)), service.CreateRoleInput{
		Key: req.Key, Name: name, Description: description,
		Permissions: toPermissions(req.Permissions),
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "role.created", "role", role.Key, map[string]string{"permissions": strconv.Itoa(len(role.Permissions))})
	response.Created(c, toRoleView(*role))
}

// UpdateRole edits an operator-defined role.
func (h *Handler) UpdateRole(c *gin.Context) {
	var req roleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, handler.WrapBind(err))
		return
	}
	// Omitted fields stay nil, which leaves them untouched; an empty string
	// clears them.
	in := service.UpdateRoleInput{Name: req.Name, Description: req.Description}
	if req.Permissions != nil {
		perms := toPermissions(req.Permissions)
		in.Permissions = &perms
	}
	role, err := h.RoleSvc.Update(c.Request.Context(), string(middleware.Role(c)), c.Param("key"), in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "role.updated", "role", role.Key, map[string]string{"permissions": strconv.Itoa(len(role.Permissions))})
	response.OK(c, toRoleView(*role))
}

// DeleteRole removes an operator-defined role that nobody holds.
func (h *Handler) DeleteRole(c *gin.Context) {
	key := c.Param("key")
	members, err := h.Staff.MembersOf(c.Request.Context(), key)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.RoleSvc.Delete(c.Request.Context(), key, members); err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "role.deleted", "role", key, nil)
	response.NoContent(c)
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
		"staff":             h.toStaffView(c.Request.Context(), *result.User),
		"generatedPassword": result.GeneratedPassword,
	})
}

// rolePermissions resolves a role, including the ones an operator defined.
func (h *Handler) rolePermissions(ctx context.Context, role domain.UserRole) []domain.Permission {
	if h.Roles != nil {
		return h.Roles.Permissions(ctx, string(role))
	}
	return domain.Permissions(role)
}
