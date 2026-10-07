package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/response"
)

type CategoryView struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Names    map[string]string `json:"names,omitempty"`
	Slug     string            `json:"slug"`
	ParentID string            `json:"parentId"`
	Sort     int               `json:"sort"`
}

type ProductView struct {
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
	WeightGrams int           `json:"weightGrams"`
	Variants    []VariantView `json:"variants,omitempty"`
	Rating      float64       `json:"rating,omitempty"`
	ReviewCount int64         `json:"reviewCount,omitempty"`
}

type VariantView struct {
	ID          string            `json:"id"`
	SKU         string            `json:"sku"`
	Name        string            `json:"name"`
	PriceCents  int64             `json:"priceCents"`
	Stock       int               `json:"stock"`
	WeightGrams int               `json:"weightGrams"`
	Attributes  map[string]string `json:"attributes,omitempty"`
	Sort        int               `json:"sort"`
	Active      bool              `json:"active"`
}

// ListCategories is shared: the storefront navigates by category and the ops
// console uses it for the product form.
func (h *Handler) ListCategories(c *gin.Context) {
	cats, err := h.Catalog.ListCategories(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]CategoryView, 0, len(cats))
	for _, cat := range cats {
		out = append(out, ToCategoryView(cat, middleware.LocaleOf(c)))
	}
	response.OK(c, out)
}

// ListProducts is shared: the public catalog filters to published products,
// while admins may list any status.
func (h *Handler) ListProducts(c *gin.Context) {
	filter := domain.ProductFilter{
		CategoryID: c.Query("categoryId"),
		Keyword:    c.Query("keyword"),
		Sort:       c.Query("sort"),
		Cursor:     c.Query("cursor"),
	}
	filter.Page, filter.PageSize = ParsePage(c, 20)
	_, filter.CursorMode = c.GetQuery("cursor")

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
	response.PaginatedCursor(c, ToProductViews(page.Items), page.Total, page.Page, page.PageSize, page.NextCursor)
}

func ToCategoryView(cat domain.Category, locale string) CategoryView {
	return CategoryView{
		ID: cat.ID, Name: cat.LocalizedName(locale), Names: cat.Names,
		Slug: cat.Slug, ParentID: cat.ParentID, Sort: cat.Sort,
	}
}

func ToProductView(p domain.Product) ProductView {
	images := p.Images
	if images == nil {
		images = []string{}
	}
	view := ProductView{
		ID: p.ID, CategoryID: p.CategoryID, Title: p.Title, Slug: p.Slug,
		Description: p.Description, PriceCents: p.PriceCents, Currency: p.Currency,
		CoverImage: p.CoverImage, Images: images, Status: string(p.Status), Stock: p.Stock,
		WeightGrams: p.WeightGrams,
	}
	if len(p.Variants) > 0 {
		view.Variants = make([]VariantView, 0, len(p.Variants))
		for _, v := range p.Variants {
			view.Variants = append(view.Variants, VariantView{
				ID: v.ID, SKU: v.SKU, Name: v.Name, PriceCents: v.PriceCents,
				Stock: v.Stock, Attributes: v.Attributes, Sort: v.Sort, Active: v.Active,
				WeightGrams: v.WeightGrams,
			})
		}
	}
	return view
}

func ToProductViews(items []domain.Product) []ProductView {
	out := make([]ProductView, 0, len(items))
	for _, p := range items {
		out = append(out, ToProductView(p))
	}
	return out
}
