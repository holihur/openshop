package payment

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/holihur/openshop/internal/adapter/signing"
	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

func newWeChatForTest(t *testing.T) (*WeChatPay, *rsa.PrivateKey) {
	t.Helper()
	merchantKeyPEM, _, _ := generateKeyPair(t)
	// The platform's private key signs notifications (as Tencent does) and the
	// matching public key is what the adapter verifies with.
	_, platformPublicPEM, platformKey := generateKeyPair(t)
	w, err := NewWeChatPay(WeChatOptions{
		MchID: "1900000109", AppID: "wxd678efh567hg6787", SerialNo: "5157F09EFDC096DE15EBE81A47057A72",
		PrivateKey: merchantKeyPEM, PlatformCert: platformPublicPEM,
		APIv3Key:  "0123456789abcdef0123456789abcdef",
		NotifyURL: "https://shop.example.com/api/v1/webhooks/payments/wechat",
	})
	if err != nil {
		t.Fatalf("adapter: %v", err)
	}
	if !w.Configured() {
		t.Fatal("adapter should be configured")
	}
	return w, platformKey
}

func TestWeChatAuthorizationIsSigned(t *testing.T) {
	w, _ := newWeChatForTest(t)

	body := []byte(`{"appid":"wxd678efh567hg6787"}`)
	authorization, err := w.authorization("POST", "/v3/pay/transactions/native", body)
	if err != nil {
		t.Fatalf("authorization: %v", err)
	}
	if !strings.HasPrefix(authorization, "WECHATPAY2-SHA256-RSA2048 ") {
		t.Fatalf("unexpected scheme: %s", authorization)
	}
	fields := parseAuthorization(t, authorization)
	for _, required := range []string{"mchid", "nonce_str", "signature", "timestamp", "serial_no"} {
		if fields[required] == "" {
			t.Errorf("authorization is missing %s", required)
		}
	}
	if fields["mchid"] != "1900000109" || fields["serial_no"] != "5157F09EFDC096DE15EBE81A47057A72" {
		t.Errorf("authorization fields = %v", fields)
	}

	// The signature must cover method, path, timestamp, nonce and body, exactly
	// as WeChat reconstructs it, so verify with the merchant public key.
	merchantPublic, err := signing.ParseRSAPublicKey(mustPublicFromPrivate(t, w.privateKey))
	if err != nil {
		t.Fatalf("public key: %v", err)
	}
	message := strings.Join([]string{
		"POST", "/v3/pay/transactions/native", fields["timestamp"], fields["nonce_str"], string(body), "",
	}, "\n")
	if err := signing.VerifySHA256RSA(merchantPublic, message, fields["signature"]); err != nil {
		t.Fatalf("authorization signature does not verify: %v", err)
	}
}

func TestWeChatChargeRequiresConfiguration(t *testing.T) {
	// An adapter without credentials must refuse rather than pretend.
	empty, err := NewWeChatPay(WeChatOptions{})
	if err != nil {
		t.Fatalf("adapter: %v", err)
	}
	if empty.Configured() {
		t.Fatal("an adapter without credentials must not report itself configured")
	}
	if _, err := empty.Charge(context.Background(), port.ChargeRequest{OrderNo: "OS1"}); err == nil {
		t.Fatal("charging without credentials must fail")
	}
}

func TestWeChatNotificationIsVerifiedAndDecrypted(t *testing.T) {
	w, platformKey := newWeChatForTest(t)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := "n0nc3"

	resource, err := json.Marshal(map[string]any{
		"out_trade_no":   "OS456",
		"transaction_id": "4200001234202601091234567890",
		"trade_state":    "SUCCESS",
		"amount":         map[string]any{"total": 19900},
	})
	if err != nil {
		t.Fatalf("marshal resource: %v", err)
	}
	ciphertext, err := w.encrypt(string(resource), "abcdefghijkl", "transaction")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	body, err := json.Marshal(map[string]any{
		"id": "EV-1", "event_type": "TRANSACTION.SUCCESS",
		"resource": map[string]any{
			"algorithm": "AEAD_AES_256_GCM", "ciphertext": ciphertext,
			"nonce": "abcdefghijkl", "associated_data": "transaction",
		},
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	signature, err := signing.SignSHA256RSA(platformKey, strings.Join([]string{timestamp, nonce, string(body), ""}, "\n"))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	headers := map[string]string{
		"Wechatpay-Timestamp": timestamp, "Wechatpay-Nonce": nonce, "Wechatpay-Signature": signature,
	}

	event, err := w.ParseWebhook(context.Background(), headers, body)
	if err != nil {
		t.Fatalf("webhook: %v", err)
	}
	if event.Status != domain.PaymentSucceeded {
		t.Errorf("status = %s, want succeeded", event.Status)
	}
	if event.OrderNo != "OS456" || event.ProviderRef != "OS456" {
		t.Errorf("refs = %s / %s", event.OrderNo, event.ProviderRef)
	}
	if event.Raw["transactionId"] != "4200001234202601091234567890" {
		t.Errorf("raw = %v, want the transaction id", event.Raw)
	}
	if event.AmountCents != 19900 {
		t.Errorf("amount = %d, want 19900", event.AmountCents)
	}

	// A body that does not match the signature must be rejected.
	tampered := append([]byte{}, body...)
	tampered[0] = 'X'
	if _, err := w.ParseWebhook(context.Background(), headers, tampered); err == nil {
		t.Fatal("a tampered notification must be rejected")
	}

	// Missing signature headers are refused.
	if _, err := w.ParseWebhook(context.Background(), map[string]string{}, body); err == nil {
		t.Fatal("a notification without signature headers must be rejected")
	}
}

func TestWeChatNotificationRejectsStaleTimestamp(t *testing.T) {
	w, platformKey := newWeChatForTest(t)
	stale := strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10)
	nonce := "n0nc3"
	body := []byte(`{"event_type":"TRANSACTION.SUCCESS","resource":{}}`)
	signature, err := signing.SignSHA256RSA(platformKey, strings.Join([]string{stale, nonce, string(body), ""}, "\n"))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	_, err = w.ParseWebhook(context.Background(), map[string]string{
		"Wechatpay-Timestamp": stale, "Wechatpay-Nonce": nonce, "Wechatpay-Signature": signature,
	}, body)
	if err == nil {
		t.Fatal("a replay of an old notification must be rejected")
	}
}

// parseAuthorization extracts the key="value" pairs of a WECHATPAY2 header.
func parseAuthorization(t *testing.T, value string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, part := range strings.Split(strings.TrimPrefix(value, "WECHATPAY2-SHA256-RSA2048 "), ",") {
		key, raw, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		out[key] = strings.Trim(raw, `"`)
	}
	return out
}

func mustPublicFromPrivate(t *testing.T, key *rsa.PrivateKey) string {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}
