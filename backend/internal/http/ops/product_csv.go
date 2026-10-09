package ops

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/response"
	"github.com/holihur/openshop/internal/service"
)

// ExportProducts streams the whole catalogue as CSV (one row per variant).
func (h *Handler) ExportProducts(c *gin.Context) {
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="products.csv"`)
	c.Status(http.StatusOK)
	if err := h.Catalog.ExportProductCSV(c.Request.Context(), c.Writer); err != nil {
		// The headers are already sent, so the error can only be logged.
		h.Logger.Error("product export failed", "error", err)
		return
	}
	h.RecordAudit(c, "product.export", "product", "", nil)
}

// ImportProducts applies a CSV upload. Use ?dry_run=true to validate a file and
// see what it would change without writing anything, and ?create=false to refuse
// handles that do not exist yet.
func (h *Handler) ImportProducts(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, fmt.Errorf("%w: a CSV file is required", domain.ErrInvalidArgument))
		return
	}
	if fileHeader.Size > 32<<20 {
		response.Fail(c, fmt.Errorf("%w: the file exceeds 32MiB", domain.ErrInvalidArgument))
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		response.Fail(c, err)
		return
	}
	defer f.Close()

	// Validation-only is the safe default for a first look at a file, so
	// creating new products requires an explicit create=true.
	opts := service.ImportOptions{
		DryRun: c.Query("dry_run") == "true",
		Create: c.Query("create") != "false",
	}
	report, err := h.Catalog.ImportProductCSV(c.Request.Context(), f, opts)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if !opts.DryRun {
		h.RecordAudit(c, "product.import", "product", "", map[string]string{
			"created": fmt.Sprint(report.ProductsCreated),
			"updated": fmt.Sprint(report.ProductsUpdated),
			"errors":  fmt.Sprint(len(report.Errors)),
		})
	}
	response.OK(c, report)
}
