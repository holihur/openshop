package payment

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/holihur/openshop/internal/adapter/signing"
	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// WeChatPay implements WeChat Pay v3 "Native" payment: the API returns a
// code_url that the shopper's WeChat app scans.
//
// Every request is signed with the merchant private key (SHA-256 with RSA) and
// every notification is verified against WeChat's platform certificate and then
// decrypted with the API v3 key, so an attacker cannot forge a paid order.
type WeChatPay struct {
	mchID       string
	appID       string
	serialNo    string
	gateway     string
	privateKey  *rsa.PrivateKey
	platformKey *rsa.PublicKey
	apiV3Key    []byte
	notifyURL   string
	client      *http.Client
}

// WeChatOptions is the configuration the adapter needs.
type WeChatOptions struct {
	MchID        string
	AppID        string
	SerialNo     string
	Gateway      string
	PrivateKey   string
	PlatformCert string
	APIv3Key     string
	NotifyURL    string
}

func NewWeChatPay(opts WeChatOptions) (*WeChatPay, error) {
	gateway := strings.TrimSpace(opts.Gateway)
	if gateway == "" {
		gateway = "https://api.mch.weixin.qq.com"
	}
	w := &WeChatPay{
		mchID: strings.TrimSpace(opts.MchID), appID: strings.TrimSpace(opts.AppID),
		serialNo: strings.TrimSpace(opts.SerialNo), gateway: strings.TrimSuffix(gateway, "/"),
		notifyURL: strings.TrimSpace(opts.NotifyURL),
		client:    &http.Client{Timeout: 15 * time.Second},
	}
	if opts.PrivateKey != "" {
		key, err := signing.ParseRSAPrivateKey(opts.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("wechat pay: private key: %w", err)
		}
		w.privateKey = key
	}
	if opts.PlatformCert != "" {
		key, err := signing.ParseRSAPublicKey(opts.PlatformCert)
		if err != nil {
			return nil, fmt.Errorf("wechat pay: platform certificate: %w", err)
		}
		w.platformKey = key
	}
	if opts.APIv3Key != "" {
		// The API v3 key is exactly 32 bytes and is used as an AES-256 key.
		w.apiV3Key = []byte(opts.APIv3Key)
	}
	return w, nil
}

func (w *WeChatPay) Name() string { return "wechat" }

// Configured reports whether the adapter has everything it needs to take money.
func (w *WeChatPay) Configured() bool {
	return w.mchID != "" && w.appID != "" && w.serialNo != "" && w.privateKey != nil
}

func (w *WeChatPay) Charge(ctx context.Context, req port.ChargeRequest) (*port.ChargeResult, error) {
	if !w.Configured() {
		return nil, errors.New("wechat pay: not configured")
	}

	payload := map[string]any{
		"appid":        w.appID,
		"mchid":        w.mchID,
		"description":  subjectOr(req),
		"out_trade_no": req.OrderNo,
		"notify_url":   w.notifyURL,
		"amount": map[string]any{
			"total":    req.AmountCents,
			"currency": currencyOr(req.Currency, "CNY"),
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	var out struct {
		CodeURL string `json:"code_url"`
		Prepay  string `json:"prepay_id"`
	}
	if err := w.call(ctx, http.MethodPost, "/v3/pay/transactions/native", body, &out); err != nil {
		return nil, err
	}
	return &port.ChargeResult{
		// The transaction id only exists once the shopper pays; the order number
		// identifies the charge until the notification arrives.
		ProviderRef: req.OrderNo,
		Status:      domain.PaymentPending,
		// The code URL is what the shopper's WeChat app scans.
		RedirectURL: out.CodeURL,
		Raw:         map[string]any{"prepayId": out.Prepay, "codeUrl": out.CodeURL},
	}, nil
}

func (w *WeChatPay) Refund(ctx context.Context, req port.RefundRequest) error {
	if !w.Configured() {
		return errors.New("wechat pay: not configured")
	}
	payload := map[string]any{
		"out_trade_no":  req.ProviderRef,
		"out_refund_no": fmt.Sprintf("r-%s-%d", req.ProviderRef, time.Now().UnixNano()),
		"reason":        req.Reason,
		"amount": map[string]any{
			"refund":   req.AmountCents,
			"total":    req.AmountCents,
			"currency": "CNY",
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return w.call(ctx, http.MethodPost, "/v3/refund/domestic/refunds", body, nil)
}

// ParseWebhook verifies WeChat's notification signature and decrypts the
// resource. Both steps are mandatory: the body is encrypted, and the signature
// covers the ciphertext.
func (w *WeChatPay) ParseWebhook(_ context.Context, headers map[string]string, body []byte) (*port.WebhookEvent, error) {
	timestamp := header(headers, "Wechatpay-Timestamp")
	nonce := header(headers, "Wechatpay-Nonce")
	signature := header(headers, "Wechatpay-Signature")
	if timestamp == "" || nonce == "" || signature == "" {
		return nil, errors.New("wechat pay notify: missing signature headers")
	}
	if w.platformKey == nil {
		return nil, errors.New("wechat pay notify: platform certificate not configured")
	}
	message := strings.Join([]string{timestamp, nonce, string(body), ""}, "\n")
	if err := signing.VerifySHA256RSA(w.platformKey, message, signature); err != nil {
		return nil, errors.New("wechat pay notify: invalid signature")
	}
	// Replay protection: a notification older than five minutes is refused.
	if ts, err := strconv.ParseInt(timestamp, 10, 64); err == nil {
		if drift := time.Since(time.Unix(ts, 0)); drift > 5*time.Minute || drift < -5*time.Minute {
			return nil, errors.New("wechat pay notify: timestamp outside the tolerance window")
		}
	}

	var envelope struct {
		EventType string `json:"event_type"`
		Resource  struct {
			Algorithm      string `json:"algorithm"`
			Ciphertext     string `json:"ciphertext"`
			Nonce          string `json:"nonce"`
			AssociatedData string `json:"associated_data"`
		} `json:"resource"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("wechat pay notify: %w", err)
	}
	if envelope.Resource.Algorithm != "AEAD_AES_256_GCM" {
		return nil, fmt.Errorf("wechat pay notify: unsupported algorithm %q", envelope.Resource.Algorithm)
	}

	plain, err := w.decrypt(envelope.Resource.Ciphertext, envelope.Resource.Nonce, envelope.Resource.AssociatedData)
	if err != nil {
		return nil, err
	}

	var resource struct {
		OutTradeNo    string `json:"out_trade_no"`
		TransactionID string `json:"transaction_id"`
		TradeState    string `json:"trade_state"`
		Amount        struct {
			Total int64 `json:"total"`
		} `json:"amount"`
	}
	if err := json.Unmarshal(plain, &resource); err != nil {
		return nil, fmt.Errorf("wechat pay notify: %w", err)
	}

	status := domain.PaymentPending
	switch strings.ToUpper(resource.TradeState) {
	case "SUCCESS":
		status = domain.PaymentSucceeded
	case "CLOSED", "REVOKED", "PAYERROR":
		status = domain.PaymentFailed
	}
	// Keyed by out_trade_no for the same reason as Alipay: it is the identifier
	// the charge was created with. The transaction id goes in Raw for
	// reconciliation.
	return &port.WebhookEvent{
		ProviderRef: resource.OutTradeNo,
		OrderNo:     resource.OutTradeNo,
		Status:      status,
		AmountCents: resource.Amount.Total,
		Raw: map[string]any{
			"eventType":     envelope.EventType,
			"tradeState":    resource.TradeState,
			"transactionId": resource.TransactionID,
		},
	}, nil
}

// decrypt opens the AEAD-protected resource of a notification.
func (w *WeChatPay) decrypt(ciphertext, nonce, associatedData string) ([]byte, error) {
	if len(w.apiV3Key) != 32 {
		return nil, errors.New("wechat pay notify: API v3 key must be 32 bytes")
	}
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("wechat pay notify: ciphertext is not base64: %w", err)
	}
	block, err := aes.NewCipher(w.apiV3Key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(nonce) != aead.NonceSize() {
		return nil, errors.New("wechat pay notify: invalid nonce length")
	}
	plain, err := aead.Open(nil, []byte(nonce), raw, []byte(associatedData))
	if err != nil {
		return nil, errors.New("wechat pay notify: could not decrypt the resource")
	}
	return plain, nil
}

// encrypt is the inverse of decrypt; the tests use it to build a notification
// exactly as WeChat would, so the verification path is exercised end to end.
func (w *WeChatPay) encrypt(plaintext, nonce, associatedData string) (string, error) {
	block, err := aes.NewCipher(w.apiV3Key)
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	out := aead.Seal(nil, []byte(nonce), []byte(plaintext), []byte(associatedData))
	return base64.StdEncoding.EncodeToString(out), nil
}

// call performs a signed WeChat Pay v3 request.
func (w *WeChatPay) call(ctx context.Context, method, path string, body []byte, out any) error {
	httpReq, err := http.NewRequestWithContext(ctx, method, w.gateway+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	authorization, err := w.authorization(method, path, body)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", authorization)

	res, err := w.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("wechat pay: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("wechat pay: %w", err)
	}
	if res.StatusCode >= 300 {
		return fmt.Errorf("wechat pay rejected the request (%d): %s", res.StatusCode, summarise(raw))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("wechat pay: unexpected response: %w", err)
	}
	return nil
}

// authorization builds the WECHATPAY2-SHA256-RSA2048 header.
func (w *WeChatPay) authorization(method, path string, body []byte) (string, error) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonce, err := randomNonce()
	if err != nil {
		return "", err
	}
	message := strings.Join([]string{method, path, timestamp, nonce, string(body), ""}, "\n")
	signature, err := signing.SignSHA256RSA(w.privateKey, message)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		`WECHATPAY2-SHA256-RSA2048 mchid="%s",nonce_str="%s",signature="%s",timestamp="%s",serial_no="%s"`,
		w.mchID, nonce, signature, timestamp, w.serialNo,
	), nil
}

func randomNonce() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return strings.ToUpper(base64.StdEncoding.EncodeToString(buf)), nil
}

func summarise(body []byte) string {
	if len(body) > 300 {
		body = body[:300]
	}
	return strings.TrimSpace(string(body))
}

func currencyOr(currency, fallback string) string {
	if strings.TrimSpace(currency) == "" {
		return fallback
	}
	return strings.ToUpper(currency)
}
