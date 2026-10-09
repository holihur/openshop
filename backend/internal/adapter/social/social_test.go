package social

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/holihur/openshop/internal/adapter/signing"
)

// generateKeyPair returns a PEM key pair for the signing tests.
func generateKeyPair(t *testing.T) (string, string, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	priv := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	pub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	return string(priv), string(pub), key
}

func TestWeChatAuthCodeURL(t *testing.T) {
	w := NewWeChat(WeChatOptions{
		AppID: "wx-app", AppSecret: "s", RedirectURL: "https://shop.example.com/cb",
	})
	got := w.AuthCodeURL("state-1")
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	query := parsed.Query()
	if query.Get("appid") != "wx-app" || query.Get("state") != "state-1" {
		t.Errorf("query = %v", query)
	}
	if query.Get("scope") != "snsapi_login" || query.Get("response_type") != "code" {
		t.Errorf("query = %v", query)
	}
	// WeChat requires the fragment marker, otherwise it shows an error page.
	if !strings.HasSuffix(got, "#wechat_redirect") {
		t.Errorf("URL must end with #wechat_redirect: %s", got)
	}
}

func TestWeChatExchangePrefersUnionID(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/sns/oauth2/access_token":
			// The secret must be sent, and the code echoed back.
			if r.URL.Query().Get("secret") != "shh" || r.URL.Query().Get("code") != "code-1" {
				t.Errorf("token query = %v", r.URL.Query())
			}
			_ = json.NewEncoder(rw).Encode(map[string]any{
				"access_token": "at", "openid": "OPENID", "unionid": "UNION", "expires_in": 7200,
			})
		case "/sns/userinfo":
			if r.URL.Query().Get("access_token") != "at" || r.URL.Query().Get("openid") != "OPENID" {
				t.Errorf("userinfo query = %v", r.URL.Query())
			}
			_ = json.NewEncoder(rw).Encode(map[string]any{
				"nickname": "微信用户", "headimgurl": "https://example.com/a.png", "unionid": "UNION",
			})
		default:
			http.NotFound(rw, r)
		}
	}))
	defer server.Close()

	w := NewWeChat(WeChatOptions{
		AppID: "wx-app", AppSecret: "shh", RedirectURL: "https://shop.example.com/cb",
		BaseURL: server.URL,
	})
	identity, err := w.Exchange(context.Background(), "code-1")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	// The unionid is stable across apps, so it wins over the openid.
	if identity.Subject != "UNION" {
		t.Errorf("subject = %q, want the union id", identity.Subject)
	}
	if identity.Name != "微信用户" || identity.AvatarURL == "" {
		t.Errorf("profile = %+v", identity)
	}
	if identity.Provider != "wechat" {
		t.Errorf("provider = %q", identity.Provider)
	}
	if len(paths) != 2 {
		t.Errorf("paths = %v", paths)
	}
}

func TestWeChatExchangeFallsBackToOpenID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sns/oauth2/access_token" {
			_ = json.NewEncoder(rw).Encode(map[string]any{"access_token": "at", "openid": "OPENID"})
			return
		}
		// A failing profile call must not block sign-in.
		http.Error(rw, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	w := NewWeChat(WeChatOptions{AppID: "a", AppSecret: "b", RedirectURL: "https://x/cb", BaseURL: server.URL})
	identity, err := w.Exchange(context.Background(), "code")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if identity.Subject != "OPENID" {
		t.Errorf("subject = %q, want the openid", identity.Subject)
	}
}

func TestWeChatExchangeRejectsProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(rw).Encode(map[string]any{"errcode": 40029, "errmsg": "invalid code"})
	}))
	defer server.Close()

	w := NewWeChat(WeChatOptions{AppID: "a", AppSecret: "b", RedirectURL: "https://x/cb", BaseURL: server.URL})
	if _, err := w.Exchange(context.Background(), "bad"); err == nil {
		t.Fatal("a provider error must fail the exchange")
	}
}

func TestWeChatUnconfigured(t *testing.T) {
	w := NewWeChat(WeChatOptions{AppID: "a"}) // no secret, no redirect
	if w.Configured() {
		t.Fatal("an incomplete provider must not report itself configured")
	}
	if _, err := w.Exchange(context.Background(), "x"); err == nil {
		t.Fatal("exchange must fail when unconfigured")
	}
}

func TestAlipayAuthCodeURL(t *testing.T) {
	a := NewAlipay(AlipayOptions{AppID: "2021", PrivateKey: "k", RedirectURL: "https://shop/cb"})
	got := a.AuthCodeURL("state-2")
	parsed, _ := url.Parse(got)
	query := parsed.Query()
	if query.Get("app_id") != "2021" || query.Get("scope") != "auth_user" || query.Get("state") != "state-2" {
		t.Errorf("query = %v", query)
	}
}

func TestAlipayExchangeSignsTheTokenRequest(t *testing.T) {
	privatePEM, publicPEM, _ := generateKeyPair(t)
	publicKey, err := signing.ParseRSAPublicKey(publicPEM)
	if err != nil {
		t.Fatalf("public key: %v", err)
	}

	var sawProfile bool
	gateway := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		form, err := url.ParseQuery(string(body))
		if err != nil {
			t.Errorf("body: %v", err)
		}
		params := map[string]string{}
		for key := range form {
			params[key] = form.Get(key)
		}
		// The request must carry a valid RSA2 signature: this is what Alipay
		// itself checks, and the reason the token call cannot be unsigned.
		if err := signing.VerifySHA256RSA(publicKey, signing.AlipaySignatureBase(params), params["sign"]); err != nil {
			t.Errorf("signature does not verify: %v", err)
		}
		if params["sign_type"] != "RSA2" {
			t.Errorf("sign_type = %q", params["sign_type"])
		}

		switch params["method"] {
		case "alipay.system.oauth.token":
			if params["code"] != "code-1" || params["grant_type"] != "authorization_code" {
				t.Errorf("token params = %v", params)
			}
			_ = json.NewEncoder(rw).Encode(map[string]any{
				"alipay_system_oauth_token_response": map[string]any{
					"user_id": "2088100000000001", "access_token": "ali-token",
				},
			})
		case "alipay.user.info.share":
			sawProfile = true
			if params["auth_token"] != "ali-token" {
				t.Errorf("profile params = %v", params)
			}
			_ = json.NewEncoder(rw).Encode(map[string]any{
				"alipay_user_info_share_response": map[string]any{
					"nick_name": "支付宝用户", "avatar": "https://example.com/b.png",
				},
			})
		default:
			t.Errorf("unexpected method %q", params["method"])
		}
	}))
	defer gateway.Close()

	a := NewAlipay(AlipayOptions{
		AppID: "2021", PrivateKey: privatePEM, RedirectURL: "https://shop/cb", Gateway: gateway.URL,
	})
	identity, err := a.Exchange(context.Background(), "code-1")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if identity.Subject != "2088100000000001" || identity.Provider != "alipay" {
		t.Errorf("identity = %+v", identity)
	}
	if identity.Name != "支付宝用户" || identity.AvatarURL == "" {
		t.Errorf("profile = %+v", identity)
	}
	if !sawProfile {
		t.Error("the profile call should have been made")
	}
}

func TestAlipayExchangeRejectsProviderError(t *testing.T) {
	gateway := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(rw).Encode(map[string]any{
			"error_response": map[string]any{"code": "40002", "msg": "Invalid Signature"},
		})
	}))
	defer gateway.Close()

	privatePEM, _, _ := generateKeyPair(t)
	a := NewAlipay(AlipayOptions{AppID: "1", PrivateKey: privatePEM, RedirectURL: "https://shop/cb", Gateway: gateway.URL})
	if _, err := a.Exchange(context.Background(), "x"); err == nil {
		t.Fatal("a provider error must fail the exchange")
	}
}
