package config

import (
	"fmt"
	"log"
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
	MerchantID  string
	CallbackURL string
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

	viper.AutomaticEnv()

	accessTTL, _ := time.ParseDuration(viper.GetString("JWT_ACCESS_TTL"))
	refreshTTL, _ := time.ParseDuration(viper.GetString("JWT_REFRESH_TTL"))
	rateTTL, _ := time.ParseDuration(viper.GetString("RATE_LIMIT_TTL"))

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
			MerchantID:  viper.GetString("PAYMENT_MERCHANT_ID"),
			CallbackURL: viper.GetString("PAYMENT_CALLBACK_URL"),
		},
		Rate: RateLimitConfig{
			MaxRequests: viper.GetInt("RATE_LIMIT_MAX"),
			TTL:         rateTTL,
		},
	}
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DB.Host, c.DB.Port, c.DB.User, c.DB.Password, c.DB.Name, c.DB.SSLMode,
	)
}
