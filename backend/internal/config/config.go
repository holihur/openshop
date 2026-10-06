package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the fully resolved runtime configuration. Everything is sourced
// from the environment so the same binary can be deployed to any number of
// instances without code changes (12-factor).
type Config struct {
	App      AppConfig
	HTTP     HTTPConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	NATS     NATSConfig
	JWT      JWTConfig
	Payment  PaymentConfig
	Storage  StorageConfig
	Mail     MailConfig
	SMS      SMSConfig
	Worker   WorkerConfig
}

type AppConfig struct {
	Name       string
	Env        string
	InstanceID string
	LogLevel   string
	Currency   string
	OrderTTL   time.Duration
}

type HTTPConfig struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	CORSOrigins     []string
	RateLimitRPS    int
}

type PostgresConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	AutoMigrate     bool
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	PoolSize int
}

type NATSConfig struct {
	URL            string
	Stream         string
	ConsumerPrefix string
}

type JWTConfig struct {
	Secret        string
	Issuer        string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
	DenyListCheck bool
}

type PaymentConfig struct {
	DefaultProvider string
	MockReturnURL   string
}

type StorageConfig struct {
	Driver    string // "local" | "s3"
	LocalDir  string
	PublicURL string
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
}

type MailConfig struct {
	Driver string // "log" | "smtp"
	From   string
	Host   string
	Port   int
	User   string
	Pass   string
}

type SMSConfig struct {
	Driver string // "log" | "aliyun"
	Sign   string
}

type WorkerConfig struct {
	Enabled            bool
	OrderSweepInterval time.Duration
	OrderSweepBatch    int
	OutboxInterval     time.Duration
	OutboxBatch        int
	// LeaderLock ensures only one instance runs a given scheduled job at a time.
	LeaderLockTTL time.Duration
}

// Load reads configuration from the environment, applying safe development
// defaults. It returns an error on invalid values rather than silently
// starting with a broken config.
func Load() (*Config, error) {
	cfg := &Config{
		App: AppConfig{
			Name:       env("APP_NAME", "openshop"),
			Env:        env("APP_ENV", "development"),
			InstanceID: env("INSTANCE_ID", hostname()),
			LogLevel:   env("LOG_LEVEL", "info"),
			Currency:   env("APP_CURRENCY", "CNY"),
			OrderTTL:   envDuration("ORDER_TTL", 30*time.Minute),
		},
		HTTP: HTTPConfig{
			Addr:            env("HTTP_ADDR", ":8080"),
			ReadTimeout:     envDuration("HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    envDuration("HTTP_WRITE_TIMEOUT", 30*time.Second),
			ShutdownTimeout: envDuration("HTTP_SHUTDOWN_TIMEOUT", 15*time.Second),
			CORSOrigins:     envList("HTTP_CORS_ORIGINS", []string{"http://localhost:5173"}),
			RateLimitRPS:    envInt("HTTP_RATE_LIMIT_RPS", 50),
		},
		Postgres: PostgresConfig{
			DSN:             env("POSTGRES_DSN", "host=localhost port=5432 user=openshop password=openshop dbname=openshop sslmode=disable TimeZone=UTC"),
			MaxOpenConns:    envInt("POSTGRES_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    envInt("POSTGRES_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: envDuration("POSTGRES_CONN_MAX_LIFETIME", time.Hour),
			AutoMigrate:     envBool("POSTGRES_AUTO_MIGRATE", false),
		},
		Redis: RedisConfig{
			Addr:     env("REDIS_ADDR", "localhost:6379"),
			Password: env("REDIS_PASSWORD", ""),
			DB:       envInt("REDIS_DB", 0),
			PoolSize: envInt("REDIS_POOL_SIZE", 20),
		},
		NATS: NATSConfig{
			URL:            env("NATS_URL", "nats://localhost:4222"),
			Stream:         env("NATS_STREAM", "OPENSHOP"),
			ConsumerPrefix: env("NATS_CONSUMER_PREFIX", "openshop"),
		},
		JWT: JWTConfig{
			Secret:        env("JWT_SECRET", "dev-insecure-change-me"),
			Issuer:        env("JWT_ISSUER", "openshop"),
			AccessTTL:     envDuration("JWT_ACCESS_TTL", 2*time.Hour),
			RefreshTTL:    envDuration("JWT_REFRESH_TTL", 30*24*time.Hour),
			DenyListCheck: envBool("JWT_DENYLIST", true),
		},
		Payment: PaymentConfig{
			DefaultProvider: env("PAYMENT_PROVIDER", "mock"),
			MockReturnURL:   env("PAYMENT_MOCK_RETURN_URL", "http://localhost:5173/payment/result"),
		},
		Storage: StorageConfig{
			Driver:    env("STORAGE_DRIVER", "local"),
			LocalDir:  env("STORAGE_LOCAL_DIR", "./data/uploads"),
			PublicURL: env("STORAGE_PUBLIC_URL", "http://localhost:8080/uploads"),
			Endpoint:  env("S3_ENDPOINT", ""),
			Bucket:    env("S3_BUCKET", ""),
			AccessKey: env("S3_ACCESS_KEY", ""),
			SecretKey: env("S3_SECRET_KEY", ""),
		},
		Mail: MailConfig{
			Driver: env("MAIL_DRIVER", "log"),
			From:   env("MAIL_FROM", "noreply@openshop.local"),
			Host:   env("MAIL_HOST", ""),
			Port:   envInt("MAIL_PORT", 587),
			User:   env("MAIL_USER", ""),
			Pass:   env("MAIL_PASS", ""),
		},
		SMS: SMSConfig{
			Driver: env("SMS_DRIVER", "log"),
			Sign:   env("SMS_SIGN", "OpenShop"),
		},
		Worker: WorkerConfig{
			Enabled:            envBool("WORKER_ENABLED", true),
			OrderSweepInterval: envDuration("ORDER_SWEEP_INTERVAL", time.Minute),
			OrderSweepBatch:    envInt("ORDER_SWEEP_BATCH", 100),
			OutboxInterval:     envDuration("OUTBOX_INTERVAL", time.Second),
			OutboxBatch:        envInt("OUTBOX_BATCH", 100),
			LeaderLockTTL:      envDuration("LEADER_LOCK_TTL", 30*time.Second),
		},
	}

	if cfg.App.Env == "production" {
		if cfg.JWT.Secret == "dev-insecure-change-me" {
			return nil, fmt.Errorf("JWT_SECRET must be set in production")
		}
	}
	return cfg, nil
}

func (a AppConfig) IsProduction() bool { return a.Env == "production" }

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func envList(key string, def []string) []string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	return def
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil || h == "" {
		return "unknown"
	}
	return h
}
