package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/response"
	"github.com/holihur/openshop/internal/service"
)

type categoryView struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	ParentID string `json:"parentId"`
	Sort     int    `json:"sort"`
}

type productView struct {
	ID          string        `json:"id"`
	CategoryID  string        `json:"categoryId"`
	Title       string        `json:"title"`
	Slug        string        `json:"slug"`
	Description string        `json:"description"`
	PriceCents  int64         `json:"priceCents"`
	Currency    string        `json:"currency"`
	CoverImage  string        `json:"coverImage"`
	Images      []string      `json:"images"`
	Status      string        `json:"status"`
	Stock       int           `json:"stock"`
	Variants    []variantView `json:"variants,omitempty"`
	Rating      float64       `json:"rating,omitempty"`
	ReviewCount int64         `json:"reviewCount,omitempty"`
}

type variantView struct {
	ID         string            `json:"id"`
	SKU        string            `json:"sku"`
	Name       string            `json:"name"`
	PriceCents int64             `json:"priceCents"`
	Stock      int               `json:"stock"`
	Attributes map[string]string `json:"attributes,omitempty"`
	Sort       int               `json:"sort"`
	Active     bool              `json:"active"`
}

type createCategoryRequest struct {
	Name     string `json:"name" binding:"required"`
	Slug     string `json:"slug"`
	ParentID string `json:"parentId"`
	Sort     int    `json:"sort"`
}

type createProductRequest struct {
	CategoryID  string   `json:"categoryId"`
	Title       string   `json:"title" binding:"required"`
	Slug        string   `json:"slug"`
	Description string   `json:"description"`
	PriceCents  int64    `json:"priceCents" binding:"gte=0"`
	Currency    string   `json:"currency"`
	CoverImage  string   `json:"coverImage"`
	Images      []string `json:"images"`
	Status      string   `json:"status"`
	Stock       int      `json:"stock" binding:"gte=0"`
}

type updateProductRequest struct {
	Title       *string  `json:"title"`
	Description *string  `json:"description"`
	PriceCents  *int64   `json:"priceCents"`
	CoverImage  *string  `json:"coverImage"`
	Images      []string `json:"images"`
	Status      *string  `json:"status"`
	Stock       *int     `json:"stock"`
	CategoryID  *string  `json:"categoryId"`
}

func (h *Handler) ListCategories(c *gin.Context) {
	cats, err := h.Catalog.ListCategories(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]categoryView, 0, len(cats))
	for _, cat := range cats {
		out = append(out, toCategoryView(cat))
	}
	response.OK(c, out)
}

func (h *Handler) CreateCategory(c *gin.Context) {
	var req createCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, wrapBind(err))
		return
	}
	cat, err := h.Catalog.CreateCategory(c.Request.Context(), service.CreateCategoryInput{
		Name: req.Name, Slug: req.Slug, ParentID: req.ParentID, Sort: req.Sort,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toCategoryView(*cat))
}

func (h *Handler) ListProducts(c *gin.Context) {
	filter := domain.ProductFilter{
		CategoryID: c.Query("categoryId"),
		Keyword:    c.Query("keyword"),
		Sort:       c.Query("sort"),
	}
	filter.Page, filter.PageSize = parsePage(c, 20)

	if middleware.IsAdmin(c) {
		// Admins may filter by any status (including drafts and archives).
		if s := c.Query("status"); s != "" {
			st := domain.ProductStatus(s)
			filter.Status = &st
		}
	} else {
		// The public catalog only ever exposes published products.
		published := domain.ProductPublished
		filter.Status = &published
	}

	page, err := h.Catalog.ListProducts(c.Request.Context(), filter)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Paginated(c, toProductViews(page.Items), page.Total, page.Page, page.PageSize)
}

func (h *Handler) GetProduct(c *gin.Context) {
	p, err := h.Catalog.GetProduct(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	view := toProductView(*p)
	if h.Reviews != nil {
		if summary, err := h.Reviews.Summary(c.Request.Context(), p.ID); err == nil {
			view.Rating = summary.Average
			view.ReviewCount = summary.Count
		}
	}
	response.OK(c, view)
}

func (h *Handler) CreateProduct(c *gin.Context) {
	var req createProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, wrapBind(err))
		return
	}
	p, err := h.Catalog.CreateProduct(c.Request.Context(), service.CreateProductInput{
		CategoryID: req.CategoryID, Title: req.Title, Slug: req.Slug,
		Description: req.Description, PriceCents: req.PriceCents, Currency: req.Currency,
		CoverImage: req.CoverImage, Images: req.Images,
		Status: domain.ProductStatus(req.Status), Stock: req.Stock,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toProductView(*p))
}

func (h *Handler) UpdateProduct(c *gin.Context) {
	var req updateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, wrapBind(err))
		return
	}
	in := service.UpdateProductInput{
		Title: req.Title, Description: req.Description, PriceCents: req.PriceCents,
		CoverImage: req.CoverImage, Images: req.Images, Stock: req.Stock, CategoryID: req.CategoryID,
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
	response.OK(c, toProductView(*p))
}

func toCategoryView(cat domain.Category) categoryView {
	return categoryView{ID: cat.ID, Name: cat.Name, Slug: cat.Slug, ParentID: cat.ParentID, Sort: cat.Sort}
}

func toProductView(p domain.Product) productView {
	images := p.Images
	if images == nil {
		images = []string{}
	}
	view := productView{
		ID: p.ID, CategoryID: p.CategoryID, Title: p.Title, Slug: p.Slug,
		Description: p.Description, PriceCents: p.PriceCents, Currency: p.Currency,
		CoverImage: p.CoverImage, Images: images, Status: string(p.Status), Stock: p.Stock,
	}
	if len(p.Variants) > 0 {
		view.Variants = make([]variantView, 0, len(p.Variants))
		for _, v := range p.Variants {
			view.Variants = append(view.Variants, variantView{
				ID: v.ID, SKU: v.SKU, Name: v.Name, PriceCents: v.PriceCents,
				Stock: v.Stock, Attributes: v.Attributes, Sort: v.Sort, Active: v.Active,
			})
		}
	}
	return view
}

func toProductViews(items []domain.Product) []productView {
	out := make([]productView, 0, len(items))
	for _, p := range items {
		out = append(out, toProductView(p))
	}
	return out
}

type createVariantRequest struct {
	SKU        string            `json:"sku"`
	Name       string            `json:"name" binding:"required"`
	PriceCents int64             `json:"priceCents" binding:"gte=0"`
	Stock      int               `json:"stock" binding:"gte=0"`
	Attributes map[string]string `json:"attributes"`
	Sort       int               `json:"sort"`
	Active     *bool             `json:"active"`
}

type updateVariantRequest struct {
	SKU        *string           `json:"sku"`
	Name       *string           `json:"name"`
	PriceCents *int64            `json:"priceCents"`
	Stock      *int              `json:"stock"`
	Attributes map[string]string `json:"attributes"`
	Sort       *int              `json:"sort"`
	Active     *bool             `json:"active"`
}

func (h *Handler) ListVariants(c *gin.Context) {
	variants, err := h.Catalog.ListVariants(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]variantView, 0, len(variants))
	for _, v := range variants {
		out = append(out, variantView{
			ID: v.ID, SKU: v.SKU, Name: v.Name, PriceCents: v.PriceCents,
			Stock: v.Stock, Attributes: v.Attributes, Sort: v.Sort, Active: v.Active,
		})
	}
	response.OK(c, out)
}

func (h *Handler) CreateVariant(c *gin.Context) {
	var req createVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, wrapBind(err))
		return
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	v, err := h.Catalog.CreateVariant(c.Request.Context(), service.CreateVariantInput{
		ProductID: c.Param("id"), SKU: req.SKU, Name: req.Name, PriceCents: req.PriceCents,
		Stock: req.Stock, Attributes: req.Attributes, Sort: req.Sort, Active: active,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, variantView{
		ID: v.ID, SKU: v.SKU, Name: v.Name, PriceCents: v.PriceCents,
		Stock: v.Stock, Attributes: v.Attributes, Sort: v.Sort, Active: v.Active,
	})
}

func (h *Handler) UpdateVariant(c *gin.Context) {
	var req updateVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, wrapBind(err))
		return
	}
	v, err := h.Catalog.UpdateVariant(c.Request.Context(), c.Param("id"), service.UpdateVariantInput{
		SKU: req.SKU, Name: req.Name, PriceCents: req.PriceCents, Stock: req.Stock,
		Attributes: req.Attributes, Sort: req.Sort, Active: req.Active,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, variantView{
		ID: v.ID, SKU: v.SKU, Name: v.Name, PriceCents: v.PriceCents,
		Stock: v.Stock, Attributes: v.Attributes, Sort: v.Sort, Active: v.Active,
	})
}
