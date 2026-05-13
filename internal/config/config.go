package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
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

	// MinIO / Object Storage
	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOBucket    string
	MinIOUseSSL    bool
	MaxFileSize    int64

	// RabbitMQ (ЛР8)
	RabbitMQHost    string
	RabbitMQPort    int
	RabbitMQUser    string
	RabbitMQPass    string
	QueueRegistered string

	// SMTP (ЛР8)
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPass     string
	SMTPFrom     string
	SMTPSecure   bool
	AppPublicURL string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	port, _ := strconv.Atoi(getEnv("PORT", "4200"))
	redisPort, _ := strconv.Atoi(getEnv("REDIS_PORT", "6379"))
	cacheTTLSeconds, _ := strconv.Atoi(getEnv("CACHE_TTL_DEFAULT", "300"))
	cacheEnabled, _ := strconv.ParseBool(getEnv("CACHE_ENABLED", "true"))
	minioUseSSL, _ := strconv.ParseBool(getEnv("MINIO_USE_SSL", "false"))
	maxFileSize, _ := strconv.ParseInt(getEnv("MAX_FILE_SIZE", "10485760"), 10, 64)
	rmqPort, _ := strconv.Atoi(getEnv("RABBITMQ_PORT", "5672"))
	smtpPort, _ := strconv.Atoi(getEnv("SMTP_PORT", "587"))
	smtpSecure, _ := strconv.ParseBool(getEnv("SMTP_SECURE", "false"))

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

		// MinIO / Object Storage
		MinIOEndpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey: getEnv("MINIO_ACCESS_KEY", ""),
		MinIOSecretKey: getEnv("MINIO_SECRET_KEY", ""),
		MinIOBucket:    getEnv("MINIO_BUCKET", "wp-labs-files"),
		MinIOUseSSL:    minioUseSSL,
		MaxFileSize:    maxFileSize,

		RabbitMQHost:    getEnv("RABBITMQ_HOST", "localhost"),
		RabbitMQPort:    rmqPort,
		RabbitMQUser:    getEnv("RABBITMQ_USER", ""),
		RabbitMQPass:    getEnv("RABBITMQ_PASS", ""),
		QueueRegistered: getEnv("QUEUE_USER_REGISTERED", "wp.auth.user.registered"),

		SMTPHost:     getEnv("SMTP_HOST", ""),
		SMTPPort:     smtpPort,
		SMTPUser:     getEnv("SMTP_USER", ""),
		SMTPPass:     getEnv("SMTP_PASS", ""),
		SMTPFrom:     getEnv("SMTP_FROM", ""),
		SMTPSecure:   smtpSecure,
		AppPublicURL: strings.TrimRight(getEnv("APP_PUBLIC_URL", "http://localhost:4200"), "/"),
	}
	if err := cfg.validateMessaging(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// validateMessaging проверяет обязательные параметры RabbitMQ и SMTP (ЛР8).
func (c *Config) validateMessaging() error {
	if strings.TrimSpace(c.RabbitMQUser) == "" {
		return fmt.Errorf("конфигурация: не задан RABBITMQ_USER")
	}
	if strings.TrimSpace(c.RabbitMQPass) == "" {
		return fmt.Errorf("конфигурация: не задан RABBITMQ_PASS")
	}
	if strings.EqualFold(c.RabbitMQUser, "guest") && strings.EqualFold(c.RabbitMQPass, "guest") {
		return fmt.Errorf("конфигурация: запрещена связка guest/guest для RabbitMQ")
	}
	if strings.TrimSpace(c.SMTPHost) == "" {
		return fmt.Errorf("конфигурация: не задан SMTP_HOST")
	}
	if strings.TrimSpace(c.SMTPUser) == "" {
		return fmt.Errorf("конфигурация: не задан SMTP_USER")
	}
	if strings.TrimSpace(c.SMTPPass) == "" {
		return fmt.Errorf("конфигурация: не задан SMTP_PASS")
	}
	if strings.TrimSpace(c.SMTPFrom) == "" {
		return fmt.Errorf("конфигурация: не задан SMTP_FROM")
	}
	if c.SMTPPort <= 0 || c.SMTPPort > 65535 {
		return fmt.Errorf("конфигурация: некорректный SMTP_PORT")
	}
	if err := c.validateYandexSMTPFrom(); err != nil {
		return err
	}
	return nil
}

// smtpMailboxFromField извлекает адрес из поля From (поддержка вида "Имя <user@host>").
func smtpMailboxFromField(from string) string {
	from = strings.TrimSpace(from)
	if i := strings.LastIndex(from, "<"); i >= 0 {
		if j := strings.LastIndex(from, ">"); j > i {
			return strings.TrimSpace(from[i+1 : j])
		}
	}
	return from
}

// validateYandexSMTPFrom: у Яндекс.Pочты отправитель MAIL FROM должен совпадать с учётной записью SMTP,
// иначе типичная ошибка 535 5.7.8 "This user does not have access rights to this service".
func (c *Config) validateYandexSMTPFrom() error {
	host := strings.ToLower(strings.TrimSpace(c.SMTPHost))
	if !strings.Contains(host, "yandex") {
		return nil
	}
	fromBox := strings.ToLower(smtpMailboxFromField(c.SMTPFrom))
	userBox := strings.ToLower(strings.TrimSpace(c.SMTPUser))
	if fromBox != userBox {
		return fmt.Errorf(
			"конфигурация SMTP (Яндекс): адрес в SMTP_FROM (%q → %s) должен совпадать с SMTP_USER (%q); "+
				"иначе сервер отвечает 535 «нет прав на сервис». Укажите один и тот же ящик или добавьте алиас в настройках Почты",
			c.SMTPFrom, fromBox, c.SMTPUser,
		)
	}
	return nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// AMQPURI возвращает URI подключения к RabbitMQ.
func (c *Config) AMQPURI() string {
	u := url.URL{
		Scheme: "amqp",
		Host:   fmt.Sprintf("%s:%d", c.RabbitMQHost, c.RabbitMQPort),
		User:   url.UserPassword(c.RabbitMQUser, c.RabbitMQPass),
		Path:   "/",
	}
	return u.String()
}
