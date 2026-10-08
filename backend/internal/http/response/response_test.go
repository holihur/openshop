package response

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/holihur/openshop/internal/domain"
)

// classify is the single place where a domain error becomes an HTTP status.
// A wrong mapping would either leak a failure as a 500 or, worse, report a
// refusal as a success path.
func TestClassifyMapsDomainErrors(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"not found", domain.ErrNotFound, http.StatusNotFound, "not_found"},
		{"conflict", domain.ErrConflict, http.StatusConflict, "conflict"},
		{"invalid argument", domain.ErrInvalidArgument, http.StatusBadRequest, "invalid_argument"},
		{"unauthorized", domain.ErrUnauthorized, http.StatusUnauthorized, "unauthorized"},
		{"expired token", domain.ErrTokenExpired, http.StatusUnauthorized, "unauthorized"},
		{"invalid token", domain.ErrTokenInvalid, http.StatusUnauthorized, "unauthorized"},
		{"forbidden", domain.ErrForbidden, http.StatusForbidden, "forbidden"},
		{"email not verified", domain.ErrEmailNotVerified, http.StatusForbidden, "email_not_verified"},
		{"insufficient stock", domain.ErrInsufficientStock, http.StatusConflict, "insufficient_stock"},
		{"busy", domain.ErrLockUnavailable, http.StatusTooManyRequests, "busy"},
		{"cart empty", domain.ErrCartEmpty, http.StatusBadRequest, "cart_empty"},
		{"order not payable", domain.ErrOrderNotPayable, http.StatusConflict, "order_not_payable"},
		{"order not refundable", domain.ErrOrderNotRefundable, http.StatusConflict, "order_not_refundable"},
		{"order not shippable", domain.ErrOrderNotShippable, http.StatusConflict, "order_not_shippable"},
		{"order not completable", domain.ErrOrderNotCompletable, http.StatusConflict, "order_not_completable"},
		{"coupon exhausted", domain.ErrCouponExhausted, http.StatusConflict, "coupon_exhausted"},
		{"payment failed", domain.ErrPaymentFailed, http.StatusBadGateway, "payment_failed"},
		{"invoice unavailable", domain.ErrInvoiceUnavailable, http.StatusConflict, "invoice_unavailable"},
		{"registration disabled", domain.ErrRegistrationDisabled, http.StatusForbidden, "registration_disabled"},
		{"insufficient funds", domain.ErrInsufficientFunds, http.StatusPaymentRequired, "insufficient_funds"},
		{"loyalty disabled", domain.ErrLoyaltyDisabled, http.StatusBadRequest, "loyalty_disabled"},
		{"account locked", domain.ErrAccountLocked, http.StatusLocked, "account_locked"},
		{"oidc unavailable", domain.ErrOIDCUnavailable, http.StatusBadGateway, "oidc_unavailable"},
		{"unknown error", errors.New("boom"), http.StatusInternalServerError, "internal_error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, code, _ := classify(tc.err)
			if status != tc.status || code != tc.code {
				t.Fatalf("classify(%v) = (%d, %q), want (%d, %q)", tc.err, status, code, tc.status, tc.code)
			}
		})
	}
}

// A sentinel wrapped with context must still classify as itself.
func TestClassifyUnwrapsContext(t *testing.T) {
	wrapped := fmt.Errorf("%w: account is not active", domain.ErrForbidden)
	status, code, _ := classify(wrapped)
	if status != http.StatusForbidden || code != "forbidden" {
		t.Fatalf("wrapped forbidden classified as (%d, %q)", status, code)
	}
}

// A domain.Error carries an explicit machine-readable code which wins over the
// sentinel mapping, so handlers can return precise codes.
func TestClassifyHonoursDomainErrorCode(t *testing.T) {
	err := domain.NewError("order_not_shippable", "order is not shippable", nil)
	status, code, message := classify(err)
	if status != http.StatusConflict || code != "order_not_shippable" {
		t.Fatalf("domain error classified as (%d, %q)", status, code)
	}
	if message != "order is not shippable" {
		t.Fatalf("message = %q", message)
	}

	// An unknown code must not be reported as success.
	unknown := domain.NewError("something_new", "x", nil)
	if status, _, _ := classify(unknown); status != http.StatusInternalServerError {
		t.Fatalf("an unmapped code should fall back to 500, got %d", status)
	}
}
