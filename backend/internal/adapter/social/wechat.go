package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/holihur/openshop/internal/port"
)

// WeChat implements "sign in with WeChat" through the WeChat Open Platform
// (open.weixin.qq.com). The shopper scans a QR code, WeChat redirects back with
// a code, and the code is exchanged for the account's openid.
//
// WeChat does not return an email address, so the identity carries a stable
// subject plus a display name and avatar.
type WeChat struct {
	appID       string
	appSecret   string
	redirectURL string
	baseURL     string
	authURL     string
	client      *http.Client
}

type WeChatOptions struct {
	AppID       string
	AppSecret   string
	RedirectURL string
	// BaseURL overrides the API host; the tests point it at a stub.
	BaseURL string
	// AuthURL overrides the QR connect endpoint.
	AuthURL string
}

func NewWeChat(opts WeChatOptions) *WeChat {
	base := strings.TrimSuffix(opts.BaseURL, "/")
	if base == "" {
		base = "https://api.weixin.qq.com"
	}
	auth := opts.AuthURL
	if auth == "" {
		auth = "https://open.weixin.qq.com/connect/qrconnect"
	}
	w := &WeChat{
		appID: strings.TrimSpace(opts.AppID), appSecret: strings.TrimSpace(opts.AppSecret),
		redirectURL: strings.TrimSpace(opts.RedirectURL),
		client:      &http.Client{Timeout: 10 * time.Second},
	}
	w.baseURL, w.authURL = base, auth
	return w
}

func (w *WeChat) Name() string { return "wechat" }

// Configured reports whether the provider can be offered. It is used by the
// sign-in page: an unconfigured provider is hidden rather than broken.
func (w *WeChat) Configured() bool {
	return w.appID != "" && w.appSecret != "" && w.redirectURL != ""
}

func (w *WeChat) AuthCodeURL(state string) string {
	query := url.Values{}
	query.Set("appid", w.appID)
	query.Set("redirect_uri", w.redirectURL)
	query.Set("response_type", "code")
	query.Set("scope", "snsapi_login")
	query.Set("state", state)
	// WeChat expects the fragment marker at the end of the URL.
	return w.authURL + "?" + query.Encode() + "#wechat_redirect"
}

func (w *WeChat) Exchange(ctx context.Context, code string) (*port.Identity, error) {
	if !w.Configured() {
		return nil, fmt.Errorf("wechat: not configured")
	}

	tokenQuery := url.Values{}
	tokenQuery.Set("appid", w.appID)
	tokenQuery.Set("secret", w.appSecret)
	tokenQuery.Set("code", code)
	tokenQuery.Set("grant_type", "authorization_code")

	var token struct {
		AccessToken string `json:"access_token"`
		OpenID      string `json:"openid"`
		UnionID     string `json:"unionid"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := w.get(ctx, "/sns/oauth2/access_token", tokenQuery, &token); err != nil {
		return nil, err
	}
	if token.ErrCode != 0 || token.AccessToken == "" {
		return nil, fmt.Errorf("wechat: token exchange failed: %s", token.ErrMsg)
	}

	// The profile call is best effort: a missing nickname must not block sign-in.
	var profile struct {
		Nickname string `json:"nickname"`
		HeadIMG  string `json:"headimgurl"`
		UnionID  string `json:"unionid"`
		OpenID   string `json:"openid"`
	}
	infoQuery := url.Values{}
	infoQuery.Set("access_token", token.AccessToken)
	infoQuery.Set("openid", token.OpenID)
	infoQuery.Set("lang", "zh_CN")
	if err := w.get(ctx, "/sns/userinfo", infoQuery, &profile); err != nil {
		profile.Nickname = ""
	}

	// A unionid is stable across every app of the same Open Platform account, so
	// it is the better subject when present.
	subject := strings.TrimSpace(token.UnionID)
	if subject == "" {
		subject = strings.TrimSpace(profile.UnionID)
	}
	if subject == "" {
		subject = strings.TrimSpace(token.OpenID)
	}
	if subject == "" {
		return nil, fmt.Errorf("wechat: no account identifier returned")
	}
	return &port.Identity{
		Subject: subject, Name: profile.Nickname, AvatarURL: profile.HeadIMG,
		Provider: w.Name(),
	}, nil
}

func (w *WeChat) get(ctx context.Context, path string, query url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, w.baseURL+path+"?"+query.Encode(), nil)
	if err != nil {
		return err
	}
	res, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("wechat: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("wechat: %w", err)
	}
	if res.StatusCode >= 300 {
		return fmt.Errorf("wechat: unexpected status %d", res.StatusCode)
	}
	// WeChat returns 200 with an errcode for many failures, so the caller checks
	// the envelope after decoding.
	return json.Unmarshal(body, out)
}
