// Package response centralises HTTP envelope formatting and error mapping so
// handlers stay small and consistent.
package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
)

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type envelope struct {
	Data  any        `json:"data,omitempty"`
	Meta  any        `json:"meta,omitempty"`
	Error *errorBody `json:"error,omitempty"`
}

// OK writes a 200 response with a data payload.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, envelope{Data: data})
}

// Created writes a 201 response.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, envelope{Data: data})
}

// NoContent writes a 204 response.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Meta is pagination metadata returned alongside list responses.
type Meta struct {
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
	// NextCursor is present for keyset (cursor) listings.
	NextCursor string `json:"nextCursor,omitempty"`
}

// Paginated writes a list response with pagination metadata.
func Paginated(c *gin.Context, items any, total int64, page, pageSize int) {
	PaginatedCursor(c, items, total, page, pageSize, "")
}

// PaginatedCursor writes a list response that also carries a keyset cursor.
func PaginatedCursor(c *gin.Context, items any, total int64, page, pageSize int, nextCursor string) {
	c.JSON(http.StatusOK, envelope{
		Data: items,
		Meta: Meta{Total: total, Page: page, PageSize: pageSize, NextCursor: nextCursor},
	})
}

// Fail maps a domain error to an HTTP status code and writes the envelope.
func Fail(c *gin.Context, err error) {
	status, code, msg := classify(err)
	c.AbortWithStatusJSON(status, envelope{Error: &errorBody{Code: code, Message: msg}})
}

func classify(err error) (int, string, string) {
	var de *domain.Error
	if errors.As(err, &de) {
		return statusFor(de.Code), de.Code, de.Message
	}

	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, "not_found", "resource not found"
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict, "conflict", err.Error()
	case errors.Is(err, domain.ErrInvalidArgument):
		return http.StatusBadRequest, "invalid_argument", err.Error()
	case errors.Is(err, domain.ErrUnauthorized), errors.Is(err, domain.ErrTokenExpired), errors.Is(err, domain.ErrTokenInvalid):
		return http.StatusUnauthorized, "unauthorized", "authentication required"
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden, "forbidden", "permission denied"
	case errors.Is(err, domain.ErrEmailNotVerified):
		return http.StatusForbidden, "email_not_verified", "please verify your email first"
	case errors.Is(err, domain.ErrInsufficientStock):
		return http.StatusConflict, "insufficient_stock", "insufficient stock"
	case errors.Is(err, domain.ErrLockUnavailable):
		return http.StatusTooManyRequests, "busy", "the resource is busy, please retry"
	case errors.Is(err, domain.ErrCartEmpty):
		return http.StatusBadRequest, "cart_empty", "your cart is empty"
	case errors.Is(err, domain.ErrOrderNotPayable):
		return http.StatusConflict, "order_not_payable", "order is not payable"
	case errors.Is(err, domain.ErrOrderNotRefundable):
		return http.StatusConflict, "order_not_refundable", "order is not refundable"
	case errors.Is(err, domain.ErrOrderNotShippable):
		return http.StatusConflict, "order_not_shippable", "order is not shippable"
	case errors.Is(err, domain.ErrOrderNotCompletable):
		return http.StatusConflict, "order_not_completable", "order is not completable"
	case errors.Is(err, domain.ErrCouponExhausted):
		return http.StatusConflict, "coupon_exhausted", "coupon usage limit reached"
	case errors.Is(err, domain.ErrPaymentFailed):
		return http.StatusBadGateway, "payment_failed", "payment failed"
	case errors.Is(err, domain.ErrInvoiceUnavailable):
		return http.StatusConflict, "invoice_unavailable", "an invoice is only available for paid orders"
	default:
		return http.StatusInternalServerError, "internal_error", "something went wrong"
	}
}

func statusFor(code string) int {
	switch code {
	case "not_found":
		return http.StatusNotFound
	case "conflict":
		return http.StatusConflict
	case "invalid_argument":
		return http.StatusBadRequest
	case "unauthorized":
		return http.StatusUnauthorized
	case "forbidden":
		return http.StatusForbidden
	case "email_not_verified":
		return http.StatusForbidden
	case "insufficient_stock":
		return http.StatusConflict
	case "cart_empty":
		return http.StatusBadRequest
	case "order_not_payable":
		return http.StatusConflict
	case "order_not_refundable":
		return http.StatusConflict
	case "order_not_shippable":
		return http.StatusConflict
	case "order_not_completable":
		return http.StatusConflict
	case "coupon_exhausted":
		return http.StatusConflict
	case "coupon_invalid":
		return http.StatusBadRequest
	case "payment_failed":
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}
