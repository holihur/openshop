package social

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/holihur/openshop/internal/adapter/signing"
	"github.com/holihur/openshop/internal/port"
)

// Alipay implements "sign in with Alipay" (scope auth_user). The token call is
// an Alipay API request, so it is signed with the application's RSA2 key just
// like a payment request: Alipay does not accept an unsigned token exchange.
type Alipay struct {
	appID       string
	privateKey  string
	redirectURL string
	gateway     string
	authURL     string
	client      *http.Client
}

type AlipayOptions struct {
	AppID       string
	PrivateKey  string
	RedirectURL string
	// Gateway overrides the API endpoint; the tests point it at a stub.
	Gateway string
	// AuthURL overrides the shopper-facing authorization endpoint.
	AuthURL string
}

func NewAlipay(opts AlipayOptions) *Alipay {
	gateway := strings.TrimSpace(opts.Gateway)
	if gateway == "" {
		gateway = "https://openapi.alipay.com/gateway.do"
	}
	auth := strings.TrimSpace(opts.AuthURL)
	if auth == "" {
		auth = "https://openauth.alipay.com/oauth2/publicAppAuthorize.htm"
	}
	return &Alipay{
		appID: strings.TrimSpace(opts.AppID), privateKey: strings.TrimSpace(opts.PrivateKey),
		redirectURL: strings.TrimSpace(opts.RedirectURL), gateway: gateway, authURL: auth,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *Alipay) Name() string { return "alipay" }

// Configured reports whether the provider can be offered.
func (a *Alipay) Configured() bool {
	return a.appID != "" && a.privateKey != "" && a.redirectURL != ""
}

func (a *Alipay) AuthCodeURL(state string) string {
	query := url.Values{}
	query.Set("app_id", a.appID)
	query.Set("scope", "auth_user")
	query.Set("redirect_uri", a.redirectURL)
	query.Set("state", state)
	return a.authURL + "?" + query.Encode()
}

func (a *Alipay) Exchange(ctx context.Context, code string) (*port.Identity, error) {
	if !a.Configured() {
		return nil, fmt.Errorf("alipay: not configured")
	}
	key, err := signing.ParseRSAPrivateKey(a.privateKey)
	if err != nil {
		return nil, fmt.Errorf("alipay: private key: %w", err)
	}

	params := map[string]string{
		"app_id":     a.appID,
		"method":     "alipay.system.oauth.token",
		"format":     "JSON",
		"charset":    "utf-8",
		"sign_type":  "RSA2",
		"timestamp":  time.Now().UTC().Format("2006-01-02 15:04:05"),
		"version":    "1.0",
		"grant_type": "authorization_code",
		"code":       code,
	}
	signature, err := signing.SignSHA256RSA(key, signing.AlipaySignatureBase(params))
	if err != nil {
		return nil, fmt.Errorf("alipay: sign: %w", err)
	}
	params["sign"] = signature

	form := url.Values{}
	for name, value := range params {
		form.Set(name, value)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.gateway, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("alipay: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("alipay: %w", err)
	}

	var envelope struct {
		Response struct {
			UserID      string `json:"user_id"`
			AccessToken string `json:"access_token"`
			Code        string `json:"code"`
			Msg         string `json:"msg"`
			SubMsg      string `json:"sub_msg"`
		} `json:"alipay_system_oauth_token_response"`
		Error struct {
			Code   string `json:"code"`
			Msg    string `json:"msg"`
			SubMsg string `json:"sub_msg"`
		} `json:"error_response"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("alipay: unexpected response: %w", err)
	}
	if envelope.Error.Code != "" {
		return nil, fmt.Errorf("alipay: %s %s", envelope.Error.Msg, envelope.Error.SubMsg)
	}
	if envelope.Response.UserID == "" {
		return nil, fmt.Errorf("alipay: no account identifier returned (%s %s)",
			envelope.Response.Msg, envelope.Response.SubMsg)
	}

	identity := &port.Identity{
		Provider: a.Name(), Subject: envelope.Response.UserID,
	}
	// The profile call needs another signed request and is best effort: the user
	// id is enough to sign in.
	if name, avatar, err := a.profile(ctx, key, envelope.Response.AccessToken); err == nil {
		identity.Name, identity.AvatarURL = name, avatar
	}
	return identity, nil
}

// profile fetches the nickname and avatar with alipay.user.info.share.
func (a *Alipay) profile(ctx context.Context, key *rsa.PrivateKey, authToken string) (string, string, error) {
	if authToken == "" {
		return "", "", fmt.Errorf("alipay: no auth token")
	}
	params := map[string]string{
		"app_id":     a.appID,
		"method":     "alipay.user.info.share",
		"format":     "JSON",
		"charset":    "utf-8",
		"sign_type":  "RSA2",
		"timestamp":  time.Now().UTC().Format("2006-01-02 15:04:05"),
		"version":    "1.0",
		"auth_token": authToken,
	}
	signature, err := signing.SignSHA256RSA(key, signing.AlipaySignatureBase(params))
	if err != nil {
		return "", "", err
	}
	params["sign"] = signature

	form := url.Values{}
	for name, value := range params {
		form.Set(name, value)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.gateway, strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := a.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return "", "", err
	}
	var envelope struct {
		Response struct {
			NickName string `json:"nick_name"`
			Avatar   string `json:"avatar"`
		} `json:"alipay_user_info_share_response"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return "", "", err
	}
	return envelope.Response.NickName, envelope.Response.Avatar, nil
}
