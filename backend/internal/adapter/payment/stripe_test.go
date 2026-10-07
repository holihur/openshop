package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"testing"
	"time"

	"github.com/holihur/openshop/internal/domain"
)

func TestStripeParseWebhookVerifiesSignature(t *testing.T) {
	provider := NewStripe("sk_test", "whsec_test")
	body := []byte(`{"type":"checkout.session.completed","data":{"object":{"id":"cs_1","amount_total":1234,"client_reference_id":"ORD-1"}}}`)

	ts := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte("whsec_test"))
	mac.Write([]byte(ts))
	mac.Write([]byte("."))
	mac.Write(body)
	header := "t=" + ts + ",v1=" + hex.EncodeToString(mac.Sum(nil))

	evt, err := provider.ParseWebhook(context.Background(), map[string]string{"Stripe-Signature": header}, body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if evt.Status != domain.PaymentSucceeded || evt.ProviderRef != "cs_1" || evt.OrderNo != "ORD-1" || evt.AmountCents != 1234 {
		t.Fatalf("unexpected event: %+v", evt)
	}

	// A tampered body must fail verification.
	if _, err := provider.ParseWebhook(context.Background(), map[string]string{"Stripe-Signature": header}, []byte(`{"type":"x"}`)); err == nil {
		t.Fatal("expected signature verification to fail for a tampered body")
	}
	// A missing header must fail too.
	if _, err := provider.ParseWebhook(context.Background(), map[string]string{}, body); err == nil {
		t.Fatal("expected signature verification to fail without a header")
	}
}
