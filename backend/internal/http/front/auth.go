package front

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/response"
	"github.com/holihur/openshop/internal/service"
)

type registerRequest struct {
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	Password     string `json:"password" binding:"required,min=8"`
	Name         string `json:"name"`
	ReferralCode string `json:"referralCode"`
}

type forgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type resetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8"`
}

type verifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

type resendVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
}

func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, handler.WrapBind(err))
		return
	}
	res, err := h.Auth.Register(c.Request.Context(), service.RegisterInput{
		Email: req.Email, Phone: req.Phone, Password: req.Password, Name: req.Name,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	// Link the new account to a referrer when a code was supplied. A bad code is
	// ignored so registration never fails because of a typo.
	if h.Commission != nil && req.ReferralCode != "" {
		_ = h.Commission.ApplyReferral(c.Request.Context(), req.ReferralCode, res.User.ID)
	}
	response.Created(c, handler.ToAuthView(res))
}

func (h *Handler) VerifyEmail(c *gin.Context) {
	var req verifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, handler.WrapBind(err))
		return
	}
	if err := h.Auth.VerifyEmail(c.Request.Context(), req.Token); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"verified": true})
}

func (h *Handler) ResendVerification(c *gin.Context) {
	var req resendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, handler.WrapBind(err))
		return
	}
	if err := h.Auth.ResendVerificationByEmail(c.Request.Context(), req.Email); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"sent": true})
}

// ForgotPassword always returns success so accounts cannot be enumerated.
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, handler.WrapBind(err))
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
		response.Fail(c, handler.WrapBind(err))
		return
	}
	if err := h.Auth.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "auth.password_reset", "user", "", nil)
	response.OK(c, gin.H{"reset": true})
}
