package config

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	App     AppConfig
	DB      DBConfig
	Redis   RedisConfig
	JWT     JWTConfig
	Payment PaymentConfig
	Rate    RateLimitConfig
}

type AppConfig struct {
	Name  string
	Env   string
	Port  string
	Debug bool
}

type DBConfig struct {
	Host      string
	Port      string
	User      string
	Password  string
	Name      string
	SSLMode   string
	MaxConns  int
	IdleConns int
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret        string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

type PaymentConfig struct {
	// Provider is the active gateway ("mock" today; "zarinpal" later).
	Provider string

	MerchantID  string
	CallbackURL string

	// GatewayURL is the local mock page URL. Only used by the mock provider.
	GatewayURL string

	// CommissionPercent is the platform's cut per order, in whole percents.
	// Clamped to [0, 100] in Load so a misconfigured env can never produce
	// nonsensical values.
	CommissionPercent int
}

type RateLimitConfig struct {
	MaxRequests int
	TTL         time.Duration
}

func Load() *Config {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Set defaults
	viper.SetDefault("APP_PORT", "8080")
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("DB_SSL_MODE", "disable")
	viper.SetDefault("JWT_ACCESS_TTL", "15m")
	viper.SetDefault("JWT_REFRESH_TTL", "168h")
	viper.SetDefault("RATE_LIMIT_MAX", 100)
	viper.SetDefault("RATE_LIMIT_TTL", "1m")
	viper.SetDefault("COMMISSION_PERCENT", 20)

	viper.SetDefault("PAYMENT_PROVIDER", "mock")
	viper.SetDefault("PAYMENT_CALLBACK_URL", "http://localhost:8080/api/v1/payment/callback")
	viper.SetDefault("PAYMENT_GATEWAY_URL", "http://localhost:5500/mock-gateway.html")
	viper.SetDefault("FRONTEND_URL", "http://localhost:5500")

	viper.AutomaticEnv()

	accessTTL, _ := time.ParseDuration(viper.GetString("JWT_ACCESS_TTL"))
	refreshTTL, _ := time.ParseDuration(viper.GetString("JWT_REFRESH_TTL"))
	rateTTL, _ := time.ParseDuration(viper.GetString("RATE_LIMIT_TTL"))

	// Clamp commission to a sane range.
	commission := viper.GetInt("COMMISSION_PERCENT")
	if commission < 0 {
		commission = 0
	}
	if commission > 100 {
		commission = 100
	}

	return &Config{
		App: AppConfig{
			Name:  viper.GetString("APP_NAME"),
			Env:   viper.GetString("APP_ENV"),
			Port:  viper.GetString("APP_PORT"),
			Debug: viper.GetBool("APP_DEBUG"),
		},
		DB: DBConfig{
			Host:      viper.GetString("DB_HOST"),
			Port:      viper.GetString("DB_PORT"),
			User:      viper.GetString("DB_USER"),
			Password:  viper.GetString("DB_PASSWORD"),
			Name:      viper.GetString("DB_NAME"),
			SSLMode:   viper.GetString("DB_SSL_MODE"),
			MaxConns:  viper.GetInt("DB_MAX_CONNECTIONS"),
			IdleConns: viper.GetInt("DB_IDLE_CONNECTIONS"),
		},
		Redis: RedisConfig{
			Host:     viper.GetString("REDIS_HOST"),
			Port:     viper.GetString("REDIS_PORT"),
			Password: viper.GetString("REDIS_PASSWORD"),
			DB:       viper.GetInt("REDIS_DB"),
		},
		JWT: JWTConfig{
			Secret:        viper.GetString("JWT_SECRET"),
			RefreshSecret: viper.GetString("JWT_REFRESH_SECRET"),
			AccessTTL:     accessTTL,
			RefreshTTL:    refreshTTL,
		},
		Payment: PaymentConfig{
			Provider:          viper.GetString("PAYMENT_PROVIDER"),
			MerchantID:        viper.GetString("PAYMENT_MERCHANT_ID"),
			CallbackURL:       viper.GetString("PAYMENT_CALLBACK_URL"),
			GatewayURL:        viper.GetString("PAYMENT_GATEWAY_URL"),
			CommissionPercent: commission,
		},
		Rate: RateLimitConfig{
			MaxRequests: viper.GetInt("RATE_LIMIT_MAX"),
			TTL:         rateTTL,
		},
	}
}

// GetDSN builds a libpq-style "key=value key=value ..." connection string.
//
// Previously this used a plain fmt.Sprintf with unescaped values. libpq's
// key=value format requires any value containing whitespace, a backslash,
// or a single quote to be wrapped in single quotes with backslash/quote
// characters escaped — otherwise:
//   - a password containing a space silently truncates the DSN at that
//     space, so lib/pq/pgx parses the rest as bogus extra keys and the
//     connection fails (or, worse, connects with a different, unintended
//     set of parameters if the leftover token happens to look like a
//     valid key=value pair).
//   - a value containing a single quote can break out of the intended
//     field entirely.
//
// escapeDSNValue below applies the escaping libpq itself documents.
func (c *Config) GetDSN() string {
	pairs := []struct{ key, val string }{
		{"host", c.DB.Host},
		{"port", c.DB.Port},
		{"user", c.DB.User},
		{"password", c.DB.Password},
		{"dbname", c.DB.Name},
		{"sslmode", c.DB.SSLMode},
	}

	parts := make([]string, 0, len(pairs))
	for _, p := range pairs {
		parts = append(parts, fmt.Sprintf("%s=%s", p.key, escapeDSNValue(p.val)))
	}
	return strings.Join(parts, " ")
}

// escapeDSNValue quotes and escapes a single libpq DSN value. Values with
// no special characters are returned as-is (matches libpq's own behavior
// and keeps simple DSNs readable in logs/debuggers); anything containing
// a space, single quote, backslash, or that is empty is wrapped in single
// quotes with backslashes and quotes escaped.
func escapeDSNValue(v string) string {
	needsQuoting := v == "" || strings.ContainsAny(v, ` '\`+"\t\n")
	if !needsQuoting {
		return v
	}
	escaped := strings.ReplaceAll(v, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `'`, `\'`)
	return "'" + escaped + "'"
}