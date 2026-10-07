package ops

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/response"
	"github.com/holihur/openshop/internal/service"
)

// ListOpsTokens lists the signed-in operator's tokens.
func (h *Handler) ListOpsTokens(c *gin.Context) {
	tokens, err := h.PATs.List(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]handler.PersonalAccessTokenView, 0, len(tokens))
	for _, t := range tokens {
		out = append(out, handler.ToPATView(t))
	}
	response.OK(c, out)
}

type createOpsTokenRequest struct {
	Name          string   `json:"name"`
	Scopes        []string `json:"scopes"`
	CIDRs         []string `json:"cidrs"`
	ExpiresInDays int      `json:"expiresInDays"`
}

// CreateOpsToken mints an operations token. Its scopes are intersected with the
// caller's role, so a token can never exceed the permissions of its owner.
func (h *Handler) CreateOpsToken(c *gin.Context) {
	var req createOpsTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, domain.ErrInvalidArgument)
		return
	}
	role := middleware.Role(c)
	scopes := make([]domain.Scope, 0, len(req.Scopes))
	for _, raw := range req.Scopes {
		scope := domain.Scope(raw)
		if scope == domain.ScopeAll {
			if role != domain.RoleAdmin {
				response.Fail(c, domain.ErrForbidden)
				return
			}
			scopes = append(scopes, scope)
			continue
		}
		if !domain.HasPermission(role, domain.Permission(scope)) {
			response.Fail(c, domain.ErrForbidden)
			return
		}
		scopes = append(scopes, scope)
	}

	in := service.CreatePATInput{
		UserID: middleware.UserID(c), Name: req.Name, Realm: "ops",
		Scopes: scopes, CIDRs: req.CIDRs,
	}
	if req.ExpiresInDays > 0 {
		expires := time.Now().UTC().AddDate(0, 0, req.ExpiresInDays)
		in.ExpiresAt = &expires
	}
	record, raw, err := h.PATs.Create(c.Request.Context(), in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "token.create", "token", record.ID, nil)
	response.Created(c, handler.CreatedPATView{PersonalAccessTokenView: handler.ToPATView(*record), Token: raw})
}

// RevokeOpsToken disables one of the caller's tokens.
func (h *Handler) RevokeOpsToken(c *gin.Context) {
	if err := h.PATs.Revoke(c.Request.Context(), c.Param("id"), middleware.UserID(c)); err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "token.revoke", "token", c.Param("id"), nil)
	response.OK(c, gin.H{"ok": true})
}
