package front

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/response"
	"github.com/holihur/openshop/internal/service"
)

// ListMyTokens lists the signed-in customer's tokens.
func (h *Handler) ListMyTokens(c *gin.Context) {
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

type createTokenRequest struct {
	Name          string   `json:"name"`
	Scopes        []string `json:"scopes"`
	CIDRs         []string `json:"cidrs"`
	ExpiresInDays int      `json:"expiresInDays"`
}

// CreateMyToken mints a storefront token. The secret is returned exactly once.
func (h *Handler) CreateMyToken(c *gin.Context) {
	var req createTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, domain.ErrInvalidArgument)
		return
	}
	in := service.CreatePATInput{
		UserID: middleware.UserID(c), Name: req.Name, Realm: "front",
		Scopes: toScopes(req.Scopes), CIDRs: req.CIDRs,
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

// RevokeMyToken disables one of the caller's tokens.
func (h *Handler) RevokeMyToken(c *gin.Context) {
	if err := h.PATs.Revoke(c.Request.Context(), c.Param("id"), middleware.UserID(c)); err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "token.revoke", "token", c.Param("id"), nil)
	response.OK(c, gin.H{"ok": true})
}

// toScopes converts request strings to scopes.
func toScopes(raw []string) []domain.Scope {
	out := make([]domain.Scope, 0, len(raw))
	for _, s := range raw {
		out = append(out, domain.Scope(s))
	}
	return out
}
