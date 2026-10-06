package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/response"
	"github.com/holihur/openshop/internal/service"
)

type registerRequest struct {
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name"`
}

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

type forgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type resetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required,min=8"`
}

type verifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

type userView struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	Name          string `json:"name"`
	Role          string `json:"role"`
	EmailVerified bool   `json:"emailVerified"`
}

type authView struct {
	User         userView `json:"user"`
	AccessToken  string   `json:"accessToken"`
	RefreshToken string   `json:"refreshToken"`
	ExpiresIn    int64    `json:"expiresIn"`
}

func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, wrapBind(err))
		return
	}
	res, err := h.Auth.Register(c.Request.Context(), service.RegisterInput{
		Email: req.Email, Phone: req.Phone, Password: req.Password, Name: req.Name,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toAuthView(res))
}

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, wrapBind(err))
		return
	}
	res, err := h.Auth.Login(c.Request.Context(), service.LoginInput{
		Identifier: req.Identifier, Password: req.Password,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toAuthView(res))
}

func (h *Handler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, wrapBind(err))
		return
	}
	res, err := h.Auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toAuthView(res))
}

func (h *Handler) Logout(c *gin.Context) {
	var req logoutRequest
	_ = c.ShouldBindJSON(&req)
	if err := h.Auth.Logout(c.Request.Context(), middleware.Claims(c), req.RefreshToken); err != nil {
		response.Fail(c, err)
		return
	}
	response.NoContent(c)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.Auth.Me(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, userView{
		ID: user.ID, Email: user.Email, Phone: user.Phone, Name: user.Name, Role: string(user.Role),
		EmailVerified: user.EmailVerified,
	})
}

func (h *Handler) VerifyEmail(c *gin.Context) {
	var req verifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, wrapBind(err))
		return
	}
	if err := h.Auth.VerifyEmail(c.Request.Context(), req.Token); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"verified": true})
}

func (h *Handler) ResendVerification(c *gin.Context) {
	if err := h.Auth.ResendVerification(c.Request.Context(), middleware.UserID(c)); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"sent": true})
}

// ForgotPassword always returns success so accounts cannot be enumerated.
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, wrapBind(err))
		return
	}
	if err := h.Auth.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"sent": true})
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, wrapBind(err))
		return
	}
	if err := h.Auth.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"reset": true})
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, wrapBind(err))
		return
	}
	if err := h.Auth.ChangePassword(c.Request.Context(), middleware.UserID(c), req.CurrentPassword, req.NewPassword); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"changed": true})
}

func toAuthView(res *service.AuthResult) authView {
	return authView{
		User: userView{
			ID: res.User.ID, Email: res.User.Email, Phone: res.User.Phone,
			Name: res.User.Name, Role: string(res.User.Role), EmailVerified: res.User.EmailVerified,
		},
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		ExpiresIn:    res.ExpiresIn,
	}
}
