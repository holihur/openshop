package domain

// SettingType is the value type of a runtime setting.
type SettingType string

const (
	SettingString SettingType = "string"
	SettingInt    SettingType = "int"
	SettingBool   SettingType = "bool"
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
		Default: "", Description: "Homepage banner title (empty uses the built-in copy)"},
	{Key: "store.hero_subtitle", Group: "store", Type: SettingString,
		Default: "", Description: "Homepage banner subtitle"},
	{Key: "store.hero_image_url", Group: "store", Type: SettingString,
		Default: "", Description: "Homepage banner background image URL"},
	{Key: "store.hero_cta_url", Group: "store", Type: SettingString,
		Default: "/products", Description: "Homepage banner button target"},
	{Key: "checkout.tax_rate_bps", Group: "checkout", Type: SettingInt,
		Default: "0", Description: "Tax rate in basis points (600 = 6%)", Min: 0, Max: 10000},
	{Key: "checkout.order_ttl_minutes", Group: "checkout", Type: SettingInt,
		Default: "30", Description: "Minutes before an unpaid order expires", Min: 1, Max: 10080},
	{Key: "inventory.low_stock_threshold", Group: "inventory", Type: SettingInt,
		Default: "5", Description: "Flag products at or below this stock level", Min: 0, Max: 1000000},
	{Key: "auth.require_email_verification", Group: "auth", Type: SettingBool,
		Default: "false", Description: "Block sign-in until the email address is verified"},
	{Key: "auth.password_reset_url", Group: "auth", Type: SettingString,
		Default: "", Description: "Base URL of the password reset page"},
	{Key: "auth.email_verify_url", Group: "auth", Type: SettingString,
		Default: "", Description: "Base URL of the email verification page"},
	{Key: "security.rate_limit_rps", Group: "security", Type: SettingInt,
		Default: "50", Description: "Per-IP requests per second", Min: 1, Max: 1000000},
	{Key: "security.rate_limit_user_rps", Group: "security", Type: SettingInt,
		Default: "100", Description: "Per-authenticated-user requests per second", Min: 1, Max: 1000000},
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
