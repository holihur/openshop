// Package payment holds payment-gateway adapters. Each adapter implements
// port.PaymentProvider; the service layer only ever sees that interface, so a
// real gateway (Stripe, Alipay, WeChat Pay) can be added without changes.
package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// Mock is a deterministic provider for local development and tests. It signs
// callbacks with HMAC-SHA256 so the webhook verification path is exercised.
type Mock struct {
	returnURL string
	secret    []byte
}

func NewMock(returnURL, secret string) *Mock {
	if secret == "" {
		secret = "mock-secret"
	}
	return &Mock{returnURL: returnURL, secret: []byte(secret)}
}

func (m *Mock) Name() string { return "mock" }

func (m *Mock) Charge(_ context.Context, req port.ChargeRequest) (*port.ChargeResult, error) {
	ref := "mock_" + uuid.NewString()
	return &port.ChargeResult{
		ProviderRef: ref,
		Status:      domain.PaymentPending,
		RedirectURL: fmt.Sprintf("%s?payment_ref=%s&order_no=%s", m.returnURL, ref, req.OrderNo),
		Raw: map[string]any{
			"provider": "mock",
			"orderNo":  req.OrderNo,
			"amount":   req.AmountCents,
		},
	}, nil
}

func (m *Mock) Refund(_ context.Context, req port.RefundRequest) error {
	if req.ProviderRef == "" {
		return domain.ErrInvalidArgument
	}
	return nil
}

type mockWebhook struct {
	ProviderRef string `json:"providerRef"`
	OrderNo     string `json:"orderNo"`
	Status      string `json:"status"`
	AmountCents int64  `json:"amountCents"`
}

func (m *Mock) ParseWebhook(_ context.Context, headers map[string]string, body []byte) (*port.WebhookEvent, error) {
	if sig := header(headers, "X-Mock-Signature"); sig != "" {
		if !m.verify(body, sig) {
			return nil, fmt.Errorf("mock webhook: %w", domain.ErrPaymentFailed)
		}
	}
	var w mockWebhook
	if err := json.Unmarshal(body, &w); err != nil {
		return nil, fmt.Errorf("mock webhook decode: %w", domain.ErrInvalidArgument)
	}
	if w.OrderNo == "" {
		return nil, domain.ErrInvalidArgument
	}
	status := parseStatus(w.Status)
	return &port.WebhookEvent{
		ProviderRef: w.ProviderRef,
		OrderNo:     w.OrderNo,
		Status:      status,
		AmountCents: w.AmountCents,
	}, nil
}

func (m *Mock) verify(body []byte, sig string) bool {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sig))
}

// Sign is exposed so tests and the mock checkout page can produce a valid
// signature.
func (m *Mock) Sign(body []byte) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// BuildWebhook implements port.SandboxProvider: it fabricates a correctly
// signed callback so the full payment flow can be exercised locally.
func (m *Mock) BuildWebhook(orderNo, providerRef string, status domain.PaymentStatus, amountCents int64) (map[string]string, []byte) {
	body, _ := json.Marshal(mockWebhook{
		ProviderRef: providerRef,
		OrderNo:     orderNo,
		Status:      string(status),
		AmountCents: amountCents,
	})
	return map[string]string{"X-Mock-Signature": m.Sign(body)}, body
}

func parseStatus(s string) domain.PaymentStatus {
	switch strings.ToLower(s) {
	case "succeeded", "success", "paid":
		return domain.PaymentSucceeded
	case "failed", "failure":
		return domain.PaymentFailed
	case "refunded":
		return domain.PaymentRefunded
	default:
		return domain.PaymentPending
	}
}

func header(h map[string]string, key string) string {
	for k, v := range h {
		if strings.EqualFold(k, key) {
			return v
		}
	}
	return ""
}

var _ port.PaymentProvider = (*Mock)(nil)
var _ port.SandboxProvider = (*Mock)(nil)

// ErrNoProvider indicates a registry lookup failed.
var ErrNoProvider = errors.New("payment provider not found")

// Registry resolves providers by name.
type Registry struct {
	providers map[string]port.PaymentProvider
	def       string
}

func NewRegistry(def string, providers ...port.PaymentProvider) *Registry {
	m := make(map[string]port.PaymentProvider, len(providers))
	for _, p := range providers {
		m[p.Name()] = p
	}
	return &Registry{providers: m, def: def}
}

func (r *Registry) Get(name string) (port.PaymentProvider, error) {
	if p, ok := r.providers[name]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrNoProvider, name)
}

func (r *Registry) Default() port.PaymentProvider {
	if p, ok := r.providers[r.def]; ok {
		return p
	}
	for _, p := range r.providers {
		return p
	}
	return nil
}

var _ port.PaymentRegistry = (*Registry)(nil)
