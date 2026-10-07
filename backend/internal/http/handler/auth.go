package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/response"
	"github.com/holihur/openshop/internal/service"
)

type loginRequest struct {
	Identifier string `json:"identifier" binding:"required"`
	Password   string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type logoutRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required,min=8"`
}

type UserView struct {
	ID            string   `json:"id"`
	Email         string   `json:"email"`
	Phone         string   `json:"phone"`
	Name          string   `json:"name"`
	Role          string   `json:"role"`
	EmailVerified bool     `json:"emailVerified"`
	Permissions   []string `json:"permissions,omitempty"`
}

type AuthView struct {
	User         UserView `json:"user"`
	AccessToken  string   `json:"accessToken"`
	RefreshToken string   `json:"refreshToken"`
	ExpiresIn    int64    `json:"expiresIn"`
}

// Login is shared: both the storefront and the ops console sign in here.
func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, WrapBind(err))
		return
	}
	res, err := h.Auth.Login(c.Request.Context(), service.LoginInput{
		Identifier: req.Identifier, Password: req.Password,
	})
	if err != nil {
		if h.Audit != nil {
			h.Audit.Record(c.Request.Context(), service.Entry{
				Action: "auth.login_failed", ResourceType: "user",
				Metadata: map[string]string{"identifier": maskIdentifier(req.Identifier)},
				IP:       c.ClientIP(),
			})
		}
		response.Fail(c, err)
		return
	}
	if h.Audit != nil {
		h.Audit.Record(c.Request.Context(), service.Entry{
			ActorID: res.User.ID, ActorRole: res.User.Role, Action: "auth.login",
			ResourceType: "user", ResourceID: res.User.ID, IP: c.ClientIP(),
		})
	}
	response.OK(c, ToAuthView(res))
}

func (h *Handler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, WrapBind(err))
		return
	}
	res, err := h.Auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, ToAuthView(res))
}

func (h *Handler) Logout(c *gin.Context) {
	var req logoutRequest
	_ = c.ShouldBindJSON(&req)
	if err := h.Auth.Logout(c.Request.Context(), middleware.Claims(c), req.RefreshToken); err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "auth.logout", "user", middleware.UserID(c), nil)
	response.NoContent(c)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.Auth.Me(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, UserView{
		ID: user.ID, Email: user.Email, Phone: user.Phone, Name: user.Name, Role: string(user.Role),
		EmailVerified: user.EmailVerified, Permissions: permissionStrings(user.Role),
	})
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, WrapBind(err))
		return
	}
	if err := h.Auth.ChangePassword(c.Request.Context(), middleware.UserID(c), req.CurrentPassword, req.NewPassword); err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "auth.password_change", "user", middleware.UserID(c), nil)
	response.OK(c, gin.H{"changed": true})
}

// maskIdentifier keeps audit entries useful without storing full PII.
func maskIdentifier(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	if at := strings.IndexByte(id, '@'); at >= 0 {
		local := id[:at]
		if len(local) > 2 {
			local = local[:2] + "***"
		}
		return local + id[at:]
	}
	if len(id) > 4 {
		return id[:2] + "****" + id[len(id)-2:]
	}
	return "***"
}

func permissionStrings(role domain.UserRole) []string {
	perms := domain.Permissions(role)
	if len(perms) == 0 {
		return nil
	}
	out := make([]string, len(perms))
	for i, p := range perms {
		out[i] = string(p)
	}
	return out
}

func ToAuthView(res *service.AuthResult) AuthView {
	return AuthView{
		User: UserView{
			ID: res.User.ID, Email: res.User.Email, Phone: res.User.Phone,
			Name: res.User.Name, Role: string(res.User.Role), EmailVerified: res.User.EmailVerified,
			Permissions: permissionStrings(res.User.Role),
		},
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		ExpiresIn:    res.ExpiresIn,
	}
}
