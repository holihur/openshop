package port

import (
	"context"

	"github.com/holihur/openshop/internal/domain"
)

// ChargeRequest describes a charge to perform through a payment provider.
type ChargeRequest struct {
	OrderNo     string
	AmountCents int64
	Currency    string
	Subject     string // customer identifier shown on receipts
	ReturnURL   string
	Metadata    map[string]string
}

// ChargeResult is the provider response. RedirectURL is set for hosted
// checkout flows; ProviderRef is the id used for later reconciliation.
type ChargeResult struct {
	ProviderRef string
	Status      domain.PaymentStatus
	RedirectURL string
	Raw         map[string]any
}

// RefundRequest describes a refund.
type RefundRequest struct {
	ProviderRef string
	AmountCents int64
	Reason      string
}

// PaymentProvider abstracts a payment gateway. The mock adapter is used in
// development; Alipay/Stripe adapters can be dropped in without touching the
// service layer.
type PaymentProvider interface {
	Name() string
	Charge(ctx context.Context, req ChargeRequest) (*ChargeResult, error)
	Refund(ctx context.Context, req RefundRequest) error
	// ParseWebhook verifies the signature and normalises the callback so the
	// service layer never sees provider-specific payloads.
	ParseWebhook(ctx context.Context, headers map[string]string, body []byte) (*WebhookEvent, error)
}

// WebhookEvent is a verified, normalised provider callback.
type WebhookEvent struct {
	ProviderRef string
	OrderNo     string
	Status      domain.PaymentStatus
	AmountCents int64
	Raw         map[string]any
}

// PaymentRegistry resolves providers by name so multiple gateways can coexist.
type PaymentRegistry interface {
	Get(name string) (PaymentProvider, error)
	Default() PaymentProvider
	// Names lists the registered providers, sorted.
	Names() []string
}

// ReadinessProvider is implemented by gateways whose credentials may be missing.
// The ops console shows the state, so an operator can tell a gateway that is
// switched off from one that was never configured.
type ReadinessProvider interface {
	Configured() bool
}

// SandboxProvider is implemented by test/sandbox gateways that can fabricate a
// signed callback. It lets the storefront exercise the full payment flow in
// development without a real gateway. Real providers do not implement it.
type SandboxProvider interface {
	PaymentProvider
	BuildWebhook(orderNo, providerRef string, status domain.PaymentStatus, amountCents int64) (headers map[string]string, body []byte)
}
