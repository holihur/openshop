// Package paymentwire builds the Chinese gateway adapters from runtime settings.
//
// The identifiers (app id, merchant id, serial number) and the endpoints are
// edited in the ops console; only the signing material stays in the environment,
// because it is a secret rather than configuration. This mirrors how the OIDC
// and social sign-in providers are resolved, so all three behave the same way: a
// settings change takes effect on the next request, with no restart.
package payment

import (
	"context"
	"strings"
	"sync"

	"github.com/holihur/openshop/internal/port"
)

// Settings is the narrow view of the settings service this package needs.
type Settings interface {
	String(ctx context.Context, key string) string
}

// Secrets holds the environment-only half of the gateway configuration.
type Secrets struct {
	AlipayPrivateKey   string
	AlipayPublicKey    string
	WeChatPrivateKey   string
	WeChatAPIv3Key     string
	WeChatPlatformCert string
}

// Resolver returns the Chinese gateway adapters, rebuilt whenever their
// configuration changes.
type Resolver struct {
	settings Settings
	secrets  Secrets
	notify   func(provider string) string

	mu    sync.Mutex
	cache map[string]port.PaymentProvider
	key   string
}

// NewResolver builds a resolver. notify returns the absolute webhook URL for a
// provider, derived from the storefront URL so operators do not have to keep two
// settings in step.
func NewResolver(settings Settings, secrets Secrets, notify func(provider string) string) *Resolver {
	return &Resolver{settings: settings, secrets: secrets, notify: notify,
		cache: map[string]port.PaymentProvider{}}
}

// settingsKeys lists the settings a gateway reads, in the order they should be
// presented. Unlike the secrets, these are owned by the ops console.
func settingsKeys(name string) []string {
	if name == "alipay" {
		return []string{"payment.alipay.app_id"}
	}
	return []string{"payment.wechat.mch_id", "payment.wechat.app_id", "payment.wechat.serial_no"}
}

// NeedsEnv lists the environment variables the API process must provide for a
// gateway. The console cannot read them, so it reports the requirement instead
// of pretending to check it.
func NeedsEnv(name string) []string {
	if name == "alipay" {
		return []string{"ALIPAY_PRIVATE_KEY", "ALIPAY_PUBLIC_KEY"}
	}
	return []string{"WECHAT_PAY_PRIVATE_KEY", "WECHAT_PAY_API_V3_KEY", "WECHAT_PAY_PLATFORM_CERT"}
}

// configured reports the settings/env values for one provider.
func (r *Resolver) values(ctx context.Context, name string) map[string]string {
	if name == "alipay" {
		return map[string]string{
			"payment.alipay.app_id": strings.TrimSpace(r.settings.String(ctx, "payment.alipay.app_id")),
			"ALIPAY_PRIVATE_KEY":    strings.TrimSpace(r.secrets.AlipayPrivateKey),
			"ALIPAY_PUBLIC_KEY":     strings.TrimSpace(r.secrets.AlipayPublicKey),
		}
	}
	return map[string]string{
		"payment.wechat.mch_id":    strings.TrimSpace(r.settings.String(ctx, "payment.wechat.mch_id")),
		"payment.wechat.app_id":    strings.TrimSpace(r.settings.String(ctx, "payment.wechat.app_id")),
		"payment.wechat.serial_no": strings.TrimSpace(r.settings.String(ctx, "payment.wechat.serial_no")),
		"WECHAT_PAY_PRIVATE_KEY":   strings.TrimSpace(r.secrets.WeChatPrivateKey),
		"WECHAT_PAY_API_V3_KEY":    strings.TrimSpace(r.secrets.WeChatAPIv3Key),
		"WECHAT_PAY_PLATFORM_CERT": strings.TrimSpace(r.secrets.WeChatPlatformCert),
	}
}

// MissingSettings lists the settings an operator still has to fill in. Only
// settings are reported: the console cannot see the API's environment, so
// claiming a secret is absent when it is merely invisible would be wrong.
func (r *Resolver) MissingSettings(ctx context.Context, name string) []string {
	missing := make([]string, 0, 3)
	for _, key := range settingsKeys(name) {
		if strings.TrimSpace(r.settings.String(ctx, key)) == "" {
			missing = append(missing, key)
		}
	}
	sortStrings(missing)
	return missing
}

// Build returns the adapter for a gateway, or nil when it is not fully
// configured. The result is cached until any input changes.
func (r *Resolver) Build(ctx context.Context, name string) port.PaymentProvider {
	if name != "alipay" && name != "wechat" {
		return nil
	}
	values := r.values(ctx, name)
	for _, value := range values {
		if value == "" {
			return nil
		}
	}

	fingerprint := name + "|" + r.notify(name)
	for _, key := range sortedKeys(values) {
		fingerprint += "|" + values[key]
	}
	// The gateway endpoints are configuration too.
	if name == "alipay" {
		fingerprint += "|" + r.settings.String(ctx, "payment.alipay.gateway")
	} else {
		fingerprint += "|" + r.settings.String(ctx, "payment.wechat.gateway")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.key == fingerprint {
		if provider, ok := r.cache[name]; ok {
			return provider
		}
	}

	// A misconfigured key must not take the API down: the gateway is simply not
	// offered, and the ops console reports which value is wrong.
	r.cache = map[string]port.PaymentProvider{}
	r.key = fingerprint
	var provider port.PaymentProvider
	switch name {
	case "alipay":
		built, err := NewAlipay(AlipayOptions{
			AppID:           values["payment.alipay.app_id"],
			Gateway:         r.settings.String(ctx, "payment.alipay.gateway"),
			PrivateKey:      values["ALIPAY_PRIVATE_KEY"],
			AlipayPublicKey: values["ALIPAY_PUBLIC_KEY"],
			NotifyURL:       r.notify("alipay"),
		})
		if err != nil {
			return nil
		}
		provider = built
	case "wechat":
		built, err := NewWeChatPay(WeChatOptions{
			MchID:        values["payment.wechat.mch_id"],
			AppID:        values["payment.wechat.app_id"],
			SerialNo:     values["payment.wechat.serial_no"],
			Gateway:      r.settings.String(ctx, "payment.wechat.gateway"),
			PrivateKey:   values["WECHAT_PAY_PRIVATE_KEY"],
			APIv3Key:     values["WECHAT_PAY_API_V3_KEY"],
			PlatformCert: values["WECHAT_PAY_PLATFORM_CERT"],
			NotifyURL:    r.notify("wechat"),
		})
		if err != nil {
			return nil
		}
		provider = built
	}
	r.cache[name] = provider
	return provider
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sortStrings(keys)
	return keys
}
