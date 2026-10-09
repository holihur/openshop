package payment

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"net/url"
	"strings"
	"testing"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// generateKeyPair returns a PEM private key and public key, so the tests
// exercise the real signing code without needing a gateway account.
func generateKeyPair(t *testing.T) (privatePEM, publicPEM string, privateKey *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	priv := pem.EncodeToMemory(&pem.Block{
		Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	pub, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pub})
	return string(priv), string(pubPEM), key
}

func TestAlipayChargeSignsTheRedirect(t *testing.T) {
	privatePEM, publicPEM, _ := generateKeyPair(t)
	a, err := NewAlipay(AlipayOptions{
		AppID: "2021000000000000", PrivateKey: privatePEM, AlipayPublicKey: publicPEM,
		NotifyURL: "https://shop.example.com/api/v1/webhooks/payments/alipay",
	})
	if err != nil {
		t.Fatalf("adapter: %v", err)
	}
	if !a.Configured() {
		t.Fatal("adapter should be configured")
	}

	result, err := a.Charge(context.Background(), port.ChargeRequest{
		OrderNo: "OS123", AmountCents: 12345, Currency: "CNY", Subject: "Desk lamp",
		ReturnURL: "https://shop.example.com/payment/result",
	})
	if err != nil {
		t.Fatalf("charge: %v", err)
	}
	if !strings.HasPrefix(result.RedirectURL, "https://openapi.alipay.com/gateway.do?") {
		t.Fatalf("unexpected redirect: %s", result.RedirectURL)
	}
	if result.Status != domain.PaymentPending {
		t.Errorf("status = %s, want pending", result.Status)
	}

	parsed, err := url.Parse(result.RedirectURL)
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	query := parsed.Query()
	// The amount must be a decimal string and the subject must survive encoding.
	if !strings.Contains(query.Get("biz_content"), `"total_amount":"123.45"`) {
		t.Errorf("biz_content = %s", query.Get("biz_content"))
	}
	// The signature must verify against the public key, using exactly the rule
	// the notify handler uses to verify.
	params := map[string]string{}
	for key := range query {
		params[key] = query.Get(key)
	}
	publicKey, err := ParseRSAPublicKey(publicPEM)
	if err != nil {
		t.Fatalf("parse public key: %v", err)
	}
	if err := verifySHA256RSA(publicKey, alipaySignatureBase(params), query.Get("sign")); err != nil {
		t.Fatalf("redirect signature does not verify: %v", err)
	}
	// sign_type is excluded from the signed string by the shared rule.
	if strings.Contains(alipaySignatureBase(params), "sign_type=") {
		t.Error("sign_type must not be part of the signature base")
	}
}

func TestAlipayWebhookVerifiesTheSignature(t *testing.T) {
	privatePEM, publicPEM, privateKey := generateKeyPair(t)
	a, err := NewAlipay(AlipayOptions{
		AppID: "2021000000000000", PrivateKey: privatePEM, AlipayPublicKey: publicPEM,
	})
	if err != nil {
		t.Fatalf("adapter: %v", err)
	}

	notify := map[string]string{
		"app_id":       "2021000000000000",
		"out_trade_no": "OS123",
		"trade_no":     "2026010922001400000000000001",
		"trade_status": "TRADE_SUCCESS",
		"total_amount": "123.45",
		"sign_type":    "RSA2",
	}
	signature, err := signSHA256RSA(privateKey, alipaySignatureBase(notify))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	body := url.Values{}
	for key, value := range notify {
		body.Set(key, value)
	}
	body.Set("sign", signature)

	event, err := a.ParseWebhook(context.Background(), nil, []byte(body.Encode()))
	if err != nil {
		t.Fatalf("webhook: %v", err)
	}
	if event.Status != domain.PaymentSucceeded || event.AmountCents != 12345 {
		t.Errorf("event = %+v, want a succeeded payment of 12345", event)
	}
	// The event is keyed by the identifier the charge was created with, so the
	// payment record can always be found.
	if event.OrderNo != "OS123" || event.ProviderRef != "OS123" {
		t.Errorf("event refs = %s / %s", event.OrderNo, event.ProviderRef)
	}
	if event.Raw["tradeNo"] != "2026010922001400000000000001" {
		t.Errorf("raw = %v, want the trade number", event.Raw)
	}

	// A tampered amount must not verify: that is the whole point of the check.
	tampered := url.Values{}
	for key, value := range notify {
		tampered.Set(key, value)
	}
	tampered.Set("total_amount", "1.00")
	tampered.Set("sign", signature)
	if _, err := a.ParseWebhook(context.Background(), nil, []byte(tampered.Encode())); err == nil {
		t.Fatal("a tampered notification must be rejected")
	}

	// A missing signature is refused outright.
	unsigned := url.Values{}
	for key, value := range notify {
		unsigned.Set(key, value)
	}
	if _, err := a.ParseWebhook(context.Background(), nil, []byte(unsigned.Encode())); err == nil {
		t.Fatal("an unsigned notification must be rejected")
	}
}

func TestAlipayRejectsAnotherAppID(t *testing.T) {
	privatePEM, publicPEM, privateKey := generateKeyPair(t)
	a, err := NewAlipay(AlipayOptions{
		AppID: "2021000000000000", PrivateKey: privatePEM, AlipayPublicKey: publicPEM,
	})
	if err != nil {
		t.Fatalf("adapter: %v", err)
	}
	notify := map[string]string{
		"app_id": "9999999999999999", "out_trade_no": "OS1",
		"trade_status": "TRADE_SUCCESS", "total_amount": "1.00",
	}
	signature, _ := signSHA256RSA(privateKey, alipaySignatureBase(notify))
	notify["sign"] = signature
	body := url.Values{}
	for key, value := range notify {
		body.Set(key, value)
	}
	if _, err := a.ParseWebhook(context.Background(), nil, []byte(body.Encode())); err == nil {
		t.Fatal("a notification for another app must be rejected")
	}
}

func TestParseAmountCents(t *testing.T) {
	cases := map[string]int64{"1": 100, "1.0": 100, "1.5": 150, "123.45": 12345, "0.01": 1, "19.9": 1990}
	for input, want := range cases {
		got, err := parseAmountCents(input)
		if err != nil {
			t.Fatalf("parseAmountCents(%q): %v", input, err)
		}
		if got != want {
			t.Errorf("parseAmountCents(%q) = %d, want %d", input, got, want)
		}
	}
	if _, err := parseAmountCents(""); err == nil {
		t.Error("an empty amount must be an error")
	}
	if _, err := parseAmountCents("abc"); err == nil {
		t.Error("a non-numeric amount must be an error")
	}
}

func TestFormatAmount(t *testing.T) {
	cases := map[int64]string{0: "0.00", 5: "0.05", 100: "1.00", 12345: "123.45", -250: "-2.50"}
	for cents, want := range cases {
		if got := formatAmount(cents); got != want {
			t.Errorf("formatAmount(%d) = %s, want %s", cents, got, want)
		}
	}
}

func TestParseKeyMaterialAcceptsPEMAndBase64(t *testing.T) {
	privatePEM, _, _ := generateKeyPair(t)
	if _, err := ParseRSAPrivateKey(privatePEM); err != nil {
		t.Fatalf("PEM private key: %v", err)
	}
	// Environment variables are often single-line base64 without the header.
	block, _ := pem.Decode([]byte(privatePEM))
	encoded := base64.StdEncoding.EncodeToString(pem.EncodeToMemory(block))
	key, err := ParseRSAPrivateKey(encoded)
	if err != nil {
		t.Fatalf("base64 private key: %v", err)
	}
	if key.N == nil {
		t.Error("parsed key is empty")
	}
	if _, err := ParseRSAPrivateKey("not a key"); err == nil {
		t.Error("garbage must be rejected")
	}
}
