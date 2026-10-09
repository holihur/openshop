package payment

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/holihur/openshop/internal/adapter/signing"
	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// Alipay implements the Alipay "page pay" (电脑网站支付) gateway. The charge is a
// signed redirect to Alipay; the outcome arrives as a signed POST to the notify
// URL, whose signature is verified locally with Alipay's public key.
//
// The app id and the notify URL are configuration; the private key and Alipay's
// public key come from the environment and are never stored in the database.
type Alipay struct {
	appID        string
	gateway      string
	privateKey   *rsa.PrivateKey
	alipayPublic *rsa.PublicKey
	client       *http.Client
	notifyURL    string
}

// AlipayOptions is the configuration the adapter needs. PrivateKey and
// AlipayPublicKey accept PEM or base64.
type AlipayOptions struct {
	AppID           string
	Gateway         string
	PrivateKey      string
	AlipayPublicKey string
	NotifyURL       string
}

func NewAlipay(opts AlipayOptions) (*Alipay, error) {
	gateway := strings.TrimSpace(opts.Gateway)
	if gateway == "" {
		gateway = "https://openapi.alipay.com/gateway.do"
	}
	a := &Alipay{
		appID: strings.TrimSpace(opts.AppID), gateway: gateway,
		notifyURL: strings.TrimSpace(opts.NotifyURL),
		client:    &http.Client{Timeout: 15 * time.Second},
	}
	if opts.PrivateKey != "" {
		key, err := signing.ParseRSAPrivateKey(opts.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("alipay: private key: %w", err)
		}
		a.privateKey = key
	}
	if opts.AlipayPublicKey != "" {
		key, err := signing.ParseRSAPublicKey(opts.AlipayPublicKey)
		if err != nil {
			return nil, fmt.Errorf("alipay: platform public key: %w", err)
		}
		a.alipayPublic = key
	}
	return a, nil
}

func (a *Alipay) Name() string { return "alipay" }

// Configured reports whether the adapter has everything it needs to take money.
func (a *Alipay) Configured() bool { return a.appID != "" && a.privateKey != nil }

func (a *Alipay) Charge(_ context.Context, req port.ChargeRequest) (*port.ChargeResult, error) {
	if !a.Configured() {
		return nil, errors.New("alipay: not configured")
	}

	// alipay.trade.page.pay returns a redirect the shopper's browser follows.
	bizContent := map[string]string{
		"out_trade_no": req.OrderNo,
		"total_amount": formatAmount(req.AmountCents),
		"subject":      subjectOr(req),
		"product_code": "FAST_INSTANT_TRADE_PAY",
	}
	params, err := a.signedParams("alipay.trade.page.pay", bizContent, req.ReturnURL)
	if err != nil {
		return nil, err
	}
	return &port.ChargeResult{
		// The trade number is only known after the shopper pays, so the order
		// number identifies the charge until the notify arrives.
		ProviderRef: req.OrderNo,
		Status:      domain.PaymentPending,
		RedirectURL: a.gateway + "?" + encodeParams(params),
		Raw:         map[string]any{"outTradeNo": req.OrderNo},
	}, nil
}

func (a *Alipay) Refund(ctx context.Context, req port.RefundRequest) error {
	if !a.Configured() {
		return errors.New("alipay: not configured")
	}
	bizContent := map[string]string{
		"out_trade_no":  req.ProviderRef,
		"refund_amount": formatAmount(req.AmountCents),
	}
	if req.Reason != "" {
		bizContent["refund_reason"] = req.Reason
	}
	params, err := a.signedParams("alipay.trade.refund", bizContent, "")
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.gateway,
		strings.NewReader(encodeParams(params)))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := a.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("alipay refund: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("alipay refund: %w", err)
	}

	var envelope struct {
		Response struct {
			Code   string `json:"code"`
			Msg    string `json:"msg"`
			SubMsg string `json:"sub_msg"`
		} `json:"alipay_trade_refund_response"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("alipay refund: unexpected response: %w", err)
	}
	if envelope.Response.Code != "10000" {
		return fmt.Errorf("alipay refund rejected: %s %s",
			envelope.Response.Msg, envelope.Response.SubMsg)
	}
	return nil
}

// ParseWebhook verifies Alipay's asynchronous notification. A notification that
// does not carry a valid signature is rejected: without this check anybody could
// POST a paid order.
func (a *Alipay) ParseWebhook(_ context.Context, _ map[string]string, body []byte) (*port.WebhookEvent, error) {
	form, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, fmt.Errorf("alipay notify: %w", err)
	}
	params := make(map[string]string, len(form))
	for key := range form {
		params[key] = form.Get(key)
	}

	if a.alipayPublic == nil {
		return nil, errors.New("alipay notify: platform public key not configured")
	}
	signature := params["sign"]
	if signature == "" {
		return nil, errors.New("alipay notify: missing signature")
	}
	if err := signing.VerifySHA256RSA(a.alipayPublic, signing.AlipaySignatureBase(params), signature); err != nil {
		return nil, fmt.Errorf("alipay notify: invalid signature")
	}
	if appID := params["app_id"]; appID != "" && a.appID != "" && appID != a.appID {
		return nil, errors.New("alipay notify: unexpected app id")
	}

	status := domain.PaymentPending
	switch params["trade_status"] {
	case "TRADE_SUCCESS", "TRADE_FINISHED":
		status = domain.PaymentSucceeded
	case "TRADE_CLOSED":
		status = domain.PaymentFailed
	}

	amountCents, err := parseAmountCents(params["total_amount"])
	if err != nil {
		return nil, err
	}
	// The payment record is keyed by the identifier we sent when charging
	// (out_trade_no), not by the gateway's own trade number, so the callback can
	// always find the payment it belongs to.
	return &port.WebhookEvent{
		ProviderRef: params["out_trade_no"],
		OrderNo:     params["out_trade_no"],
		Status:      status,
		AmountCents: amountCents,
		Raw: map[string]any{
			"tradeStatus": params["trade_status"],
			"tradeNo":     params["trade_no"],
		},
	}, nil
}

// signedParams builds a signed Alipay request. returnURL belongs in the query
// string (it is not part of the business content for page pay).
func (a *Alipay) signedParams(method string, bizContent map[string]string, returnURL string) (map[string]string, error) {
	content, err := json.Marshal(bizContent)
	if err != nil {
		return nil, err
	}
	params := map[string]string{
		"app_id":      a.appID,
		"method":      method,
		"format":      "JSON",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().UTC().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"biz_content": string(content),
	}
	if returnURL != "" {
		params["return_url"] = returnURL
	}
	if a.notifyURL != "" {
		params["notify_url"] = a.notifyURL
	}
	signature, err := signing.SignSHA256RSA(a.privateKey, signing.AlipaySignatureBase(params))
	if err != nil {
		return nil, fmt.Errorf("alipay: sign: %w", err)
	}
	params["sign"] = signature
	return params, nil
}

// encodeParams renders the parameters in a stable order so the request is
// reproducible and the signature stays valid.
func encodeParams(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	values := url.Values{}
	for _, key := range keys {
		values.Set(key, params[key])
	}
	// url.Values sorts by key and escapes exactly as Alipay expects.
	return values.Encode()
}

// formatAmount renders cents as the gateway's decimal string.
func formatAmount(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}

func parseAmountCents(amount string) (int64, error) {
	amount = strings.TrimSpace(amount)
	if amount == "" {
		return 0, errors.New("missing amount")
	}
	parts := strings.SplitN(amount, ".", 2)
	units, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount %q", amount)
	}
	var cents int64
	if len(parts) == 2 {
		fraction := parts[1]
		if len(fraction) > 2 {
			fraction = fraction[:2]
		}
		for len(fraction) < 2 {
			fraction += "0"
		}
		cents, err = strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid amount %q", amount)
		}
	}
	return units*100 + cents, nil
}

func subjectOr(req port.ChargeRequest) string {
	if req.Subject != "" {
		return req.Subject
	}
	return "Order " + req.OrderNo
}
