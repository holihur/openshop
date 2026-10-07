package front

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/response"
)

type returnRequestBody struct {
	Reason string `json:"reason"`
}

// RequestReturn lets a customer ask to return a delivered order.
func (h *Handler) RequestReturn(c *gin.Context) {
	var req returnRequestBody
	_ = c.ShouldBindJSON(&req)
	result, err := h.Returns.Request(c.Request.Context(), middleware.UserID(c), c.Param("id"), req.Reason)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, handler.ToReturnView(*result))
}

// ListOrderReturns lists the return requests of an order (owner or admin).
func (h *Handler) ListOrderReturns(c *gin.Context) {
	list, err := h.Returns.ListByOrder(c.Request.Context(), middleware.UserID(c), c.Param("id"), middleware.IsAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]handler.ReturnView, 0, len(list))
	for _, r := range list {
		out = append(out, handler.ToReturnView(r))
	}
	response.OK(c, out)
}
