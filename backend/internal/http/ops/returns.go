package ops

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/response"
)

// ListReturns lists return requests for moderation (admin only).
func (h *Handler) ListReturns(c *gin.Context) {
	page, size := handler.ParsePage(c, 20)
	filter := domain.ReturnFilter{Page: page, PageSize: size}
	if s := c.Query("status"); s != "" {
		st := domain.ReturnStatus(s)
		filter.Status = &st
	}
	result, err := h.Returns.List(c.Request.Context(), filter)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]handler.ReturnView, 0, len(result.Items))
	for _, r := range result.Items {
		out = append(out, handler.ToReturnView(r))
	}
	response.Paginated(c, out, result.Total, result.Page, result.PageSize)
}

// ApproveReturn marks a return request approved.
func (h *Handler) ApproveReturn(c *gin.Context) {
	req, err := h.Returns.Approve(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "return.approve", "return", req.ID, nil)
	response.OK(c, handler.ToReturnView(*req))
}

// RejectReturn marks a return request rejected.
func (h *Handler) RejectReturn(c *gin.Context) {
	req, err := h.Returns.Reject(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "return.reject", "return", req.ID, nil)
	response.OK(c, handler.ToReturnView(*req))
}
