package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// Stripe is a dependency-free Stripe Checkout adapter. It creates a hosted
// Checkout Session and verifies webhook signatures locally, so the service
// layer never sees Stripe-specific payloads. The secret key and webhook secret
// come from the environment, never from the settings table.
type Stripe struct {
	secretKey     string
	webhookSecret string
	baseURL       string
	client        *http.Client
}

func NewStripe(secretKey, webhookSecret string) *Stripe {
	return &Stripe{
		secretKey: secretKey, webhookSecret: webhookSecret,
		baseURL: "https://api.stripe.com", client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *Stripe) Name() string { return "stripe" }

func (s *Stripe) Charge(ctx context.Context, req port.ChargeRequest) (*port.ChargeResult, error) {
	if s.secretKey == "" {
		return nil, errors.New("stripe: secret key not configured")
	}
	form := url.Values{}
	form.Set("mode", "payment")
	form.Set("success_url", req.ReturnURL)
	form.Set("cancel_url", req.ReturnURL)
	form.Set("client_reference_id", req.OrderNo)
	form.Set("line_items[0][quantity]", "1")
	form.Set("line_items[0][price_data][currency]", strings.ToLower(req.Currency))
	form.Set("line_items[0][price_data][unit_amount]", strconv.FormatInt(req.AmountCents, 10))
	form.Set("line_items[0][price_data][product_data][name]", "Order "+req.OrderNo)
	form.Set("metadata[orderNo]", req.OrderNo)

	var out struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if err := s.post(ctx, "/v1/checkout/sessions", form, &out); err != nil {
		return nil, err
	}
	return &port.ChargeResult{
		ProviderRef: out.ID,
		Status:      domain.PaymentPending,
		RedirectURL: out.URL,
	}, nil
}

func (s *Stripe) Refund(ctx context.Context, req port.RefundRequest) error {
	if s.secretKey == "" {
		return errors.New("stripe: secret key not configured")
	}
	// Resolve the PaymentIntent behind a Checkout Session, then refund it.
	paymentIntent := req.ProviderRef
	if strings.HasPrefix(req.ProviderRef, "cs_") {
		var session struct {
			PaymentIntent string `json:"payment_intent"`
		}
		if err := s.get(ctx, "/v1/checkout/sessions/"+url.PathEscape(req.ProviderRef), &session); err != nil {
			return err
		}
		if session.PaymentIntent != "" {
			paymentIntent = session.PaymentIntent
		}
	}
	form := url.Values{}
	form.Set("payment_intent", paymentIntent)
	if req.AmountCents > 0 {
		form.Set("amount", strconv.FormatInt(req.AmountCents, 10))
	}
	return s.post(ctx, "/v1/refunds", form, nil)
}

func (s *Stripe) ParseWebhook(_ context.Context, headers map[string]string, body []byte) (*port.WebhookEvent, error) {
	if err := s.verify(headers["Stripe-Signature"], body); err != nil {
		return nil, err
	}
	var event struct {
		Type string `json:"type"`
		Data struct {
			Object struct {
				ID                string            `json:"id"`
				AmountTotal       int64             `json:"amount_total"`
				ClientReferenceID string            `json:"client_reference_id"`
				Metadata          map[string]string `json:"metadata"`
			} `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, err
	}
	obj := event.Data.Object
	var status domain.PaymentStatus
	switch event.Type {
	case "checkout.session.completed", "payment_intent.succeeded":
		status = domain.PaymentSucceeded
	case "payment_intent.payment_failed", "checkout.session.expired":
		status = domain.PaymentFailed
	case "charge.refunded":
		status = domain.PaymentRefunded
	default:
		return nil, fmt.Errorf("stripe: unhandled event %q", event.Type)
	}
	orderNo := obj.ClientReferenceID
	if orderNo == "" {
		orderNo = obj.Metadata["orderNo"]
	}
	return &port.WebhookEvent{
		ProviderRef: obj.ID, OrderNo: orderNo, Status: status, AmountCents: obj.AmountTotal,
	}, nil
}

// verify checks the Stripe-Signature header (t=timestamp,v1=signature) against
// an HMAC-SHA256 of "<timestamp>.<body>".
func (s *Stripe) verify(header string, body []byte) error {
	if s.webhookSecret == "" {
		return errors.New("stripe: webhook secret not configured")
	}
	var ts string
	var sigs []string
	for _, part := range strings.Split(header, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			ts = kv[1]
		case "v1":
			sigs = append(sigs, kv[1])
		}
	}
	if ts == "" || len(sigs) == 0 {
		return errors.New("stripe: invalid signature header")
	}
	mac := hmac.New(sha256.New, []byte(s.webhookSecret))
	mac.Write([]byte(ts))
	mac.Write([]byte("."))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	for _, sig := range sigs {
		if hmac.Equal([]byte(expected), []byte(sig)) {
			return nil
		}
	}
	return errors.New("stripe: signature verification failed")
}

func (s *Stripe) post(ctx context.Context, path string, form url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+path, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.secretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return s.do(req, out)
}

func (s *Stripe) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.secretKey)
	return s.do(req, out)
}

func (s *Stripe) do(req *http.Request, out any) error {
	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	data, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return fmt.Errorf("stripe: %s: %s", res.Status, strings.TrimSpace(string(data)))
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}
