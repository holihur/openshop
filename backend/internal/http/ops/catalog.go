package ops

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/response"
	"github.com/holihur/openshop/internal/service"
)

type createCategoryRequest struct {
	Name     string            `json:"name" binding:"required"`
	Names    map[string]string `json:"names"`
	Slug     string            `json:"slug"`
	ParentID string            `json:"parentId"`
	Sort     int               `json:"sort"`
}

type updateCategoryRequest struct {
	Name     *string            `json:"name"`
	Names    *map[string]string `json:"names"`
	Slug     *string            `json:"slug"`
	ParentID *string            `json:"parentId"`
	Sort     *int               `json:"sort"`
}

type createProductRequest struct {
	CategoryID  string            `json:"categoryId"`
	Title       string            `json:"title" binding:"required"`
	Names       map[string]string `json:"names"`
	Slug        string            `json:"slug"`
	Description string            `json:"description"`
	PriceCents  int64             `json:"priceCents" binding:"gte=0"`
	Currency    string            `json:"currency"`
	CoverImage  string            `json:"coverImage"`
	Images      []string          `json:"images"`
	Status      string            `json:"status"`
	Stock       int               `json:"stock" binding:"gte=0"`
	WeightGrams int               `json:"weightGrams" binding:"gte=0"`
}

type updateProductRequest struct {
	Title       *string           `json:"title"`
	Names       map[string]string `json:"names"`
	Description *string           `json:"description"`
	PriceCents  *int64            `json:"priceCents"`
	CoverImage  *string           `json:"coverImage"`
	Images      []string          `json:"images"`
	Status      *string           `json:"status"`
	Stock       *int              `json:"stock"`
	CategoryID  *string           `json:"categoryId"`
	WeightGrams *int              `json:"weightGrams"`
}

type createVariantRequest struct {
	SKU         string            `json:"sku"`
	Name        string            `json:"name" binding:"required"`
	PriceCents  int64             `json:"priceCents" binding:"gte=0"`
	Stock       int               `json:"stock" binding:"gte=0"`
	WeightGrams int               `json:"weightGrams" binding:"gte=0"`
	Attributes  map[string]string `json:"attributes"`
	Sort        int               `json:"sort"`
	Active      *bool             `json:"active"`
}

type updateVariantRequest struct {
	SKU         *string           `json:"sku"`
	Name        *string           `json:"name"`
	PriceCents  *int64            `json:"priceCents"`
	Stock       *int              `json:"stock"`
	WeightGrams *int              `json:"weightGrams"`
	Attributes  map[string]string `json:"attributes"`
	Sort        *int              `json:"sort"`
	Active      *bool             `json:"active"`
}

func (h *Handler) CreateCategory(c *gin.Context) {
	var req createCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, handler.WrapBind(err))
		return
	}
	cat, err := h.Catalog.CreateCategory(c.Request.Context(), service.CreateCategoryInput{
		Name: req.Name, Names: req.Names, Slug: req.Slug, ParentID: req.ParentID, Sort: req.Sort,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, handler.ToCategoryView(*cat, middleware.LocaleOf(c)))
}

func (h *Handler) UpdateCategory(c *gin.Context) {
	var req updateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, handler.WrapBind(err))
		return
	}
	cat, err := h.Catalog.UpdateCategory(c.Request.Context(), c.Param("id"), service.UpdateCategoryInput{
		Name: req.Name, Names: req.Names, Slug: req.Slug, ParentID: req.ParentID, Sort: req.Sort,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "category.update", "category", cat.ID, nil)
	response.OK(c, handler.ToCategoryView(*cat, middleware.LocaleOf(c)))
}

// GetProduct returns one product with its variants (admin detail view).
func (h *Handler) GetProduct(c *gin.Context) {
	p, err := h.Catalog.GetProduct(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, handler.ToProductView(*p))
}

func (h *Handler) CreateProduct(c *gin.Context) {
	var req createProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, handler.WrapBind(err))
		return
	}
	p, err := h.Catalog.CreateProduct(c.Request.Context(), service.CreateProductInput{
		CategoryID: req.CategoryID, Title: req.Title, Slug: req.Slug, Names: req.Names,
		Description: req.Description, PriceCents: req.PriceCents, Currency: req.Currency,
		CoverImage: req.CoverImage, Images: req.Images,
		Status: domain.ProductStatus(req.Status), Stock: req.Stock, WeightGrams: req.WeightGrams,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "product.create", "product", p.ID, nil)
	response.Created(c, handler.ToProductView(*p))
}

func (h *Handler) UpdateProduct(c *gin.Context) {
	var req updateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, handler.WrapBind(err))
		return
	}
	in := service.UpdateProductInput{
		Title: req.Title, Names: req.Names, Description: req.Description, PriceCents: req.PriceCents,
		CoverImage: req.CoverImage, Images: req.Images, Stock: req.Stock, CategoryID: req.CategoryID,
		WeightGrams: req.WeightGrams,
	}
	if req.Status != nil {
		st := domain.ProductStatus(*req.Status)
		in.Status = &st
	}
	p, err := h.Catalog.UpdateProduct(c.Request.Context(), c.Param("id"), in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "product.update", "product", p.ID, nil)
	response.OK(c, handler.ToProductView(*p))
}

func (h *Handler) ListVariants(c *gin.Context) {
	variants, err := h.Catalog.ListVariants(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]handler.VariantView, 0, len(variants))
	for _, v := range variants {
		out = append(out, handler.VariantView{
			ID: v.ID, SKU: v.SKU, Name: v.Name, PriceCents: v.PriceCents,
			Stock: v.Stock, Attributes: v.Attributes, Sort: v.Sort, Active: v.Active,
		})
	}
	response.OK(c, out)
}

func (h *Handler) CreateVariant(c *gin.Context) {
	var req createVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, handler.WrapBind(err))
		return
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	v, err := h.Catalog.CreateVariant(c.Request.Context(), service.CreateVariantInput{
		ProductID: c.Param("id"), SKU: req.SKU, Name: req.Name, PriceCents: req.PriceCents,
		Stock: req.Stock, WeightGrams: req.WeightGrams, Attributes: req.Attributes, Sort: req.Sort, Active: active,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "variant.create", "variant", v.ID, nil)
	response.Created(c, handler.VariantView{
		ID: v.ID, SKU: v.SKU, Name: v.Name, PriceCents: v.PriceCents,
		Stock: v.Stock, Attributes: v.Attributes, Sort: v.Sort, Active: v.Active,
	})
}

func (h *Handler) UpdateVariant(c *gin.Context) {
	var req updateVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, handler.WrapBind(err))
		return
	}
	v, err := h.Catalog.UpdateVariant(c.Request.Context(), c.Param("id"), service.UpdateVariantInput{
		SKU: req.SKU, Name: req.Name, PriceCents: req.PriceCents, Stock: req.Stock,
		WeightGrams: req.WeightGrams, Attributes: req.Attributes, Sort: req.Sort, Active: req.Active,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "variant.update", "variant", v.ID, nil)
	response.OK(c, handler.VariantView{
		ID: v.ID, SKU: v.SKU, Name: v.Name, PriceCents: v.PriceCents,
		Stock: v.Stock, Attributes: v.Attributes, Sort: v.Sort, Active: v.Active,
	})
}

// ListProductFAQs returns a product's FAQs (admin).
func (h *Handler) ListProductFAQs(c *gin.Context) {
	faqs, err := h.Catalog.ListFAQs(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]handler.FAQView, 0, len(faqs))
	for _, f := range faqs {
		out = append(out, handler.FAQView{ID: f.ID, Question: f.Question, Answer: f.Answer})
	}
	response.OK(c, out)
}

type replaceFAQsRequest struct {
	FAQs []struct {
		Question string `json:"question"`
		Answer   string `json:"answer"`
	} `json:"faqs"`
}

// ReplaceProductFAQs swaps a product's FAQs for the supplied list (admin).
func (h *Handler) ReplaceProductFAQs(c *gin.Context) {
	var req replaceFAQsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, handler.WrapBind(err))
		return
	}
	inputs := make([]service.FAQInput, 0, len(req.FAQs))
	for _, f := range req.FAQs {
		inputs = append(inputs, service.FAQInput{Question: f.Question, Answer: f.Answer})
	}
	faqs, err := h.Catalog.ReplaceFAQs(c.Request.Context(), c.Param("id"), inputs)
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "product.faqs", "product", c.Param("id"), nil)
	out := make([]handler.FAQView, 0, len(faqs))
	for _, f := range faqs {
		out = append(out, handler.FAQView{ID: f.ID, Question: f.Question, Answer: f.Answer})
	}
	response.OK(c, out)
}
