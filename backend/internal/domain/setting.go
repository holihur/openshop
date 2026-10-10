package domain

// SettingType is the value type of a runtime setting.
type SettingType string

const (
	SettingString SettingType = "string"
	SettingInt    SettingType = "int"
	SettingBool   SettingType = "bool"
	SettingColor  SettingType = "color"
	// SettingJSON is a JSON document edited as text; it must parse.
	SettingJSON SettingType = "json"
)

// SettingDef describes one configurable runtime setting. Defaults are the
// fallback when nothing is stored; at startup they are seeded from the
// environment so an existing deployment keeps its behaviour until an operator
// changes a value in the console.
type SettingDef struct {
	Key         string
	Group       string
	Type        SettingType
	Default     string
	Description string
	// Min/Max bound integer settings (0 means unbounded on that side).
	Min int
	Max int
}

// SettingDefs is the registry of everything the ops console can change at
// runtime. Anything not listed here is startup-only: database/Redis/NATS
// addresses, listen addresses, secrets, storage credentials, TLS, etc.
var SettingDefs = []SettingDef{
	{Key: "store.public_url", Group: "store", Type: SettingString,
		Default: "http://localhost:8080", Description: "Public site URL used in the sitemap and emails"},
	{Key: "store.hero_title", Group: "store", Type: SettingString,
		Default:     "Everyday goods, thoughtfully made",
		Description: "Homepage banner title (empty uses the built-in copy)"},
	{Key: "store.hero_subtitle", Group: "store", Type: SettingString,
		Default:     "Free shipping over ¥99, a 30-day return window and support that answers.",
		Description: "Homepage banner subtitle"},
	{Key: "store.hero_image_url", Group: "store", Type: SettingString,
		Default: "", Description: "Homepage banner background image URL"},
	{Key: "store.hero_cta_url", Group: "store", Type: SettingString,
		Default: "/products", Description: "Homepage banner button target"},
	{Key: "store.announcement", Group: "store", Type: SettingString,
		Default: "", Description: "Site-wide announcement shown on every storefront page (empty hides it)"},
	{Key: "store.announcement_url", Group: "store", Type: SettingString,
		Default: "", Description: "Optional link the announcement points to"},
	{Key: "store.feature1_title", Group: "store", Type: SettingString,
		Default:     "Free shipping over ¥99",
		Description: "Homepage feature card 1 title (empty hides the card)"},
	{Key: "store.feature1_text", Group: "store", Type: SettingString,
		Default: "Standard delivery arrives in 3–5 business days.", Description: "Homepage feature card 1 text"},
	{Key: "store.feature2_title", Group: "store", Type: SettingString,
		Default: "30-day returns", Description: "Homepage feature card 2 title (empty hides the card)"},
	{Key: "store.feature2_text", Group: "store", Type: SettingString,
		Default: "Changed your mind? Send it back within 30 days.", Description: "Homepage feature card 2 text"},
	{Key: "store.feature3_title", Group: "store", Type: SettingString,
		Default: "Real support", Description: "Homepage feature card 3 title (empty hides the card)"},
	{Key: "store.feature3_text", Group: "store", Type: SettingString,
		Default: "Questions go to a person, not a queue.", Description: "Homepage feature card 3 text"},
	{Key: "store.tagline", Group: "store", Type: SettingString,
		Default: "openshop", Description: "Store name in the header and footer (empty hides it)"},
	{Key: "store.theme_color", Group: "store", Type: SettingColor,
		Default: "#4f46e5", Description: "Brand color (hex, e.g. #6366f1); empty keeps the default theme"},
	{Key: "checkout.tax_rate_bps", Group: "checkout", Type: SettingInt,
		Default: "0", Description: "Tax rate in basis points (600 = 6%)", Min: 0, Max: 10000},
	{Key: "checkout.order_ttl_minutes", Group: "checkout", Type: SettingInt,
		Default: "30", Description: "Minutes before an unpaid order expires", Min: 1, Max: 10080},
	{Key: "search.synonyms", Group: "search", Type: SettingJSON,
		Default:     "{}",
		Description: `Alternative search terms, e.g. {"sofa":["couch","settee"]}. Matching is case-insensitive.`},
	{Key: "inventory.low_stock_threshold", Group: "inventory", Type: SettingInt,
		Default: "5", Description: "Flag products at or below this stock level", Min: 0, Max: 1000000},
	{Key: "social.wechat.enabled", Group: "auth", Type: SettingBool,
		Default:     "false",
		Description: "Offer sign in with WeChat. Requires SOCIAL WeChat app id and WECHAT_APP_SECRET."},
	{Key: "social.wechat.app_id", Group: "auth", Type: SettingString,
		Default: "", Description: "WeChat Open Platform app id (the secret is WECHAT_APP_SECRET)"},
	{Key: "social.alipay.enabled", Group: "auth", Type: SettingBool,
		Default:     "false",
		Description: "Offer sign in with Alipay. Requires an Alipay app id and ALIPAY_PRIVATE_KEY."},
	{Key: "social.alipay.app_id", Group: "auth", Type: SettingString,
		Default: "", Description: "Alipay app id for sign in (the key is ALIPAY_PRIVATE_KEY)"},
	{Key: "auth.require_email_verification", Group: "auth", Type: SettingBool,
		Default: "false", Description: "Block sign-in until the email address is verified"},
	{Key: "auth.password_reset_url", Group: "auth", Type: SettingString,
		Default: "", Description: "Base URL of the password reset page"},
	{Key: "auth.email_verify_url", Group: "auth", Type: SettingString,
		Default: "", Description: "Base URL of the email verification page"},
	{Key: "auth.allow_registration", Group: "auth", Type: SettingBool,
		Default: "true", Description: "Allow public sign-up (disable to close the store)"},
	{Key: "oidc.enabled", Group: "auth", Type: SettingBool,
		Default: "false", Description: "Offer single sign-on via OpenID Connect providers"},
	{Key: "oidc.providers", Group: "auth", Type: SettingJSON,
		Default: "[]", Description: "JSON array of providers: id, name, issuer, clientId, redirectUrl, scopes, enabled"},
	{Key: "oidc.issuer", Group: "auth", Type: SettingString,
		Default: "", Description: "OIDC issuer URL (used for discovery)"},
	{Key: "oidc.client_id", Group: "auth", Type: SettingString,
		Default: "", Description: "OIDC client id"},
	{Key: "oidc.redirect_url", Group: "auth", Type: SettingString,
		Default: "", Description: "OIDC callback URL (must match the provider registration)"},
	{Key: "oidc.scopes", Group: "auth", Type: SettingString,
		Default: "openid profile email", Description: "Space-separated OIDC scopes"},
	{Key: "wallet.enabled", Group: "loyalty", Type: SettingBool,
		Default: "true", Description: "Enable the stored-value wallet"},
	{Key: "wallet.min_topup_cents", Group: "loyalty", Type: SettingInt,
		Default: "100", Description: "Smallest wallet top-up, in cents", Min: 1, Max: 100000000},
	{Key: "wallet.max_topup_cents", Group: "loyalty", Type: SettingInt,
		Default: "1000000", Description: "Largest wallet top-up, in cents (0 = unlimited)", Min: 0, Max: 100000000},
	{Key: "wallet.withdrawal_enabled", Group: "loyalty", Type: SettingBool,
		Default: "true", Description: "Allow customers to request wallet withdrawals"},
	{Key: "wallet.min_withdrawal_cents", Group: "loyalty", Type: SettingInt,
		Default: "1000", Description: "Smallest withdrawal, in cents", Min: 1, Max: 100000000},
	{Key: "wallet.withdrawal_instructions", Group: "loyalty", Type: SettingString,
		Default: "", Description: "Note shown to customers before they request a withdrawal"},
	{Key: "points.enabled", Group: "loyalty", Type: SettingBool,
		Default: "true", Description: "Enable loyalty points"},
	{Key: "points.earn_per_unit", Group: "loyalty", Type: SettingInt,
		Default: "1", Description: "Points earned per whole currency unit spent", Min: 0, Max: 100000},
	{Key: "points.redeem_cents_per_point", Group: "loyalty", Type: SettingInt,
		Default: "1", Description: "Discount in cents for one redeemed point", Min: 0, Max: 100000},
	{Key: "points.max_redeem_percent", Group: "loyalty", Type: SettingInt,
		Default: "50", Description: "Maximum share of an order payable with points", Min: 0, Max: 100},
	{Key: "commission.enabled", Group: "loyalty", Type: SettingBool,
		Default: "true", Description: "Enable referral commissions"},
	{Key: "commission.rate_bps", Group: "loyalty", Type: SettingInt,
		Default: "500", Description: "Commission rate in basis points (500 = 5%)", Min: 0, Max: 10000},
	{Key: "commission.hold_days", Group: "loyalty", Type: SettingInt,
		Default: "15", Description: "Days after order completion before a commission is paid", Min: 0, Max: 3650},
	{Key: "commission.base", Group: "loyalty", Type: SettingString,
		Default: "total", Description: "Commission base: total or subtotal"},
	{Key: "security.rate_limit_rps", Group: "security", Type: SettingInt,
		Default: "50", Description: "Per-IP requests per second", Min: 1, Max: 1000000},
	{Key: "security.auth_rate_limit_rps", Group: "security", Type: SettingInt,
		Default: "10", Description: "Per-IP requests per second on sign-in and password endpoints", Min: 1, Max: 1000000},
	{Key: "security.max_failed_attempts", Group: "security", Type: SettingInt,
		Default: "10", Description: "Consecutive failed sign-ins before an account is locked", Min: 1, Max: 1000},
	{Key: "security.lockout_minutes", Group: "security", Type: SettingInt,
		Default: "15", Description: "Minutes an account stays locked after too many failures", Min: 1, Max: 100000},
	{Key: "security.rate_limit_user_rps", Group: "security", Type: SettingInt,
		Default: "100", Description: "Per-authenticated-user requests per second", Min: 1, Max: 1000000},
	{Key: "payment.default_provider", Group: "payment", Type: SettingString,
		Default: "mock", Description: "Active payment gateway (mock, offline, stripe, alipay, wechat)"},
	{Key: "payment.enabled_providers", Group: "payment", Type: SettingString,
		Default: "",
		Description: "Comma-separated channels offered at checkout: mock, offline, stripe, alipay, wechat " +
			"(empty = every registered gateway). A gateway only appears once its credentials are set."},
	{Key: "payment.stripe_public_key", Group: "payment", Type: SettingString,
		Default: "", Description: "Stripe publishable key (public)"},
	{Key: "payment.stripe_return_url", Group: "payment", Type: SettingString,
		Default: "", Description: "Where Stripe returns the shopper after payment"},
	{Key: "mail.driver", Group: "mail", Type: SettingString,
		Default: "log", Description: "Mail transport: log (dev) or smtp"},
	{Key: "mail.from", Group: "mail", Type: SettingString,
		Default: "noreply@openshop.local", Description: "From address on outgoing email"},
	{Key: "mail.host", Group: "mail", Type: SettingString,
		Default: "", Description: "SMTP relay host"},
	{Key: "mail.port", Group: "mail", Type: SettingInt,
		Default: "587", Description: "SMTP relay port", Min: 1, Max: 65535},
	{Key: "mail.user", Group: "mail", Type: SettingString,
		Default: "", Description: "SMTP username (the password stays in the environment)"},
}

// SettingDefByKey returns the definition for a key.
func SettingDefByKey(key string) (SettingDef, bool) {
	for _, d := range SettingDefs {
		if d.Key == key {
			return d, true
		}
	}
	return SettingDef{}, false
}

// Setting is a resolved setting with its current value, for the console.
type Setting struct {
	Key         string
	Group       string
	Type        SettingType
	Value       string
	Default     string
	Description string
	Min         int
	Max         int
}
