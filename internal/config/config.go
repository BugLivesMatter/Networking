package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	Port       int
	AppEnv     string

	// JWT конфигурация
	JWTAccessSecret      string
	JWTRefreshSecret     string
	JWTAccessExpiration  string
	JWTRefreshExpiration string

	// OAuth2 Yandex
	YandexClientID     string
	YandexClientSecret string
	YandexCallbackURL  string

	// Redis и кеш
	RedisHost       string
	RedisPort       int
	RedisPassword   string
	CacheTTLDefault time.Duration
	CacheEnabled    bool
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	port, _ := strconv.Atoi(getEnv("PORT", "4200"))
	dbPort, _ := strconv.Atoi(getEnv("DB_PORT", "5432"))
	redisPort, _ := strconv.Atoi(getEnv("REDIS_PORT", "6379"))
	cacheTTLSeconds, _ := strconv.Atoi(getEnv("CACHE_TTL_DEFAULT", "300"))
	cacheEnabled, _ := strconv.ParseBool(getEnv("CACHE_ENABLED", "true"))

	cfg := &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     dbPort,
		DBUser:     getEnv("DB_USER", "student"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "wp_labs"),
		Port:       port,
		AppEnv:     getEnv("APP_ENV", "development"),

		// JWT
		JWTAccessSecret:      getEnv("JWT_ACCESS_SECRET", ""),
		JWTRefreshSecret:     getEnv("JWT_REFRESH_SECRET", ""),
		JWTAccessExpiration:  getEnv("JWT_ACCESS_EXPIRATION", "15m"),
		JWTRefreshExpiration: getEnv("JWT_REFRESH_EXPIRATION", "7d"),

		// OAuth2
		YandexClientID:     getEnv("YANDEX_CLIENT_ID", ""),
		YandexClientSecret: getEnv("YANDEX_CLIENT_SECRET", ""),
		YandexCallbackURL:  getEnv("YANDEX_CALLBACK_URL", ""),

		// Redis и кеш
		RedisHost:       getEnv("REDIS_HOST", "localhost"),
		RedisPort:       redisPort,
		RedisPassword:   getEnv("REDIS_PASSWORD", ""),
		CacheTTLDefault: time.Duration(cacheTTLSeconds) * time.Second,
		CacheEnabled:    cacheEnabled,
	}
	return cfg, nil
}

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName,
	)
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// MigrationDSN возвращает строку подключения в формате URL для golang-migrate
// Формат: postgres://user:password@host:port/dbname?sslmode=disable
func (c *Config) MigrationDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.DBUser,
		c.DBPassword,
		c.DBHost,
		c.DBPort,
		c.DBName,
	)
}
