package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoURI    string
	MongoDBName string
	Port        int
	AppEnv      string

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
	redisPort, _ := strconv.Atoi(getEnv("REDIS_PORT", "6379"))
	cacheTTLSeconds, _ := strconv.Atoi(getEnv("CACHE_TTL_DEFAULT", "300"))
	cacheEnabled, _ := strconv.ParseBool(getEnv("CACHE_ENABLED", "true"))

	cfg := &Config{
		MongoURI:    getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDBName: getEnv("MONGO_DB_NAME", "wp_labs"),
		Port:        port,
		AppEnv:      getEnv("APP_ENV", "development"),

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

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
