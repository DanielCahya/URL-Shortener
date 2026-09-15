package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration variables for the application.
type Config struct {
	AppEnv            string
	ServerPort        string
	BaseURL           string
	DatabaseURL       string
	DBMaxConns        int32
	DBMinConns        int32
	DBMaxConnLifetime time.Duration
	DBMaxConnIdleTime time.Duration
	LogLevel          string
	JWTSecret         string
	RedisAddr         string
	RateLimitAnon     int
	RateLimitAuth     int
	RateLimitWindow   int
	RabbitMQURL       string
}

// Load reads configuration from environment variables, falling back to sensible defaults.
// It also attempts to load a .env file if present.
func Load() (*Config, error) {
	// Attempt to load .env file if available (ignore error if file does not exist)
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:            getEnv("APP_ENV", "development"),
		ServerPort:        getEnv("SERVER_PORT", "8080"),
		BaseURL:           getEnv("BASE_URL", "http://localhost:8080"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/urlshortener?sslmode=disable"),
		DBMaxConns:        int32(getEnvAsInt("DB_MAX_CONNS", 25)),
		DBMinConns:        int32(getEnvAsInt("DB_MIN_CONNS", 5)),
		DBMaxConnLifetime: getEnvAsDuration("DB_MAX_CONN_LIFETIME", time.Hour),
		DBMaxConnIdleTime: getEnvAsDuration("DB_MAX_CONN_IDLE_TIME", 30*time.Minute),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		JWTSecret:         getEnv("JWT_SECRET", "supersecret-dev-key"),
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		RateLimitAnon:     getEnvAsInt("RATE_LIMIT_ANONYMOUS", 10),
		RateLimitAuth:     getEnvAsInt("RATE_LIMIT_AUTH", 100),
		RateLimitWindow:   getEnvAsInt("RATE_LIMIT_WINDOW_SECONDS", 60),
		RabbitMQURL:       getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
	}

	if cfg.ServerPort == "" {
		return nil, fmt.Errorf("SERVER_PORT cannot be empty")
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL cannot be empty")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET cannot be empty")
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	valStr := getEnv(key, "")
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return defaultVal
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	valStr := getEnv(key, "")
	if val, err := time.ParseDuration(valStr); err == nil {
		return val
	}
	return defaultVal
}
