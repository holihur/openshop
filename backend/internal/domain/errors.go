package domain

import "errors"

// Sentinel errors shared across layers. Handlers translate these into HTTP
// status codes, adapters may wrap them. Keeping them in the domain keeps the
// business rules independent from any transport or storage concern.
var (
	ErrNotFound             = errors.New("resource not found")
	ErrConflict             = errors.New("resource conflict")
	ErrInvalidArgument      = errors.New("invalid argument")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrForbidden            = errors.New("forbidden")
	ErrInsufficientStock    = errors.New("insufficient stock")
	ErrLockUnavailable      = errors.New("resource is busy, retry later")
	ErrPaymentFailed        = errors.New("payment failed")
	ErrOrderNotPayable      = errors.New("order is not payable")
	ErrOrderNotRefundable   = errors.New("order is not refundable")
	ErrOrderNotShippable    = errors.New("order is not shippable")
	ErrOrderNotCompletable  = errors.New("order is not completable")
	ErrCouponExhausted      = errors.New("coupon usage limit reached")
	ErrCartEmpty            = errors.New("cart is empty")
	ErrTokenInvalid         = errors.New("invalid token")
	ErrTokenExpired         = errors.New("token expired")
	ErrEmailNotVerified     = errors.New("email not verified")
	ErrInvoiceUnavailable   = errors.New("invoice is not available for this order")
	ErrRegistrationDisabled = errors.New("registration is disabled")
	ErrInsufficientFunds    = errors.New("insufficient wallet balance")
	ErrLoyaltyDisabled      = errors.New("wallet and points are disabled")
	ErrAccountLocked        = errors.New("account temporarily locked")
	ErrOIDCUnavailable      = errors.New("identity provider is unavailable")
)

// Error is a domain error carrying a stable machine-readable code alongside a
// human readable message. Adapters and the transport layer can inspect Code.
type Error struct {
	Code    string
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Err }

// NewError builds a domain error.
func NewError(code, message string, err error) *Error {
	return &Error{Code: code, Message: message, Err: err}
}
