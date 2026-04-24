package health

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/lab2/rest-api/internal/cache"
	"github.com/lab2/rest-api/internal/category/domain"
	categoryrepo "github.com/lab2/rest-api/internal/category/repository"
	"github.com/lab2/rest-api/pkg/pagination"
)

// categoriesListCachePayload — тот же JSON, что кладёт categoryService.List в Redis (см. internal/category/service/service.go).
type categoriesListCachePayload struct {
	Categories []domain.Category `json:"categories"`
	Total      int64             `json:"total"`
	TotalPages int               `json:"total_pages"`
}

// DiagnosisResponse — сравнение латентности MongoDB и Redis и оценка выгоды от кеша.
type DiagnosisResponse struct {
	CheckedAt string `json:"checkedAt"`

	MongoDB DiagnosisMongoSection `json:"mongodb"`
	Redis   DiagnosisRedisSection `json:"redis"`

	// ComparisonSimple — ping MongoDB vs PING Redis.
	ComparisonSimple *DiagnosisComparison `json:"comparisonSimple,omitempty"`
	// ComparisonWorkload — repository.List (как при промахе кеша) vs cache.Get (как при попадании).
	ComparisonWorkload *DiagnosisComparison `json:"comparisonWorkload,omitempty"`
	// ComparisonMissVsHit — полный промах (List + cache.Set) vs cache hit (Get), как два последовательных запроса GET /categories.
	ComparisonMissVsHit *DiagnosisComparison `json:"comparisonMissVsHit,omitempty"`

	Notes string `json:"notes,omitempty"`
}

// DiagnosisMongoSection — замеры обращений к MongoDB.
type DiagnosisMongoSection struct {
	OK bool `json:"ok"`

	SimpleQuery string  `json:"simpleQuery,omitempty"`
	SimpleMs    float64 `json:"simpleLatencyMs,omitempty"`
	SimpleError string  `json:"simpleError,omitempty"`

	// Ниже — тот же путь данных, что GET /categories при промахе кеша.
	ListRepoMethod string `json:"listRepositoryMethod,omitempty"`
	Page           int    `json:"page"`
	Limit          int    `json:"limit"`
	CacheKey       string `json:"cacheKey,omitempty"`

	// ListMs — только CategoryRepository.List (CountDocuments + Find внутри репозитория).
	ListMs float64 `json:"listLatencyMs,omitempty"`
	// ListRowCount — число записей на странице.
	ListRowCount int   `json:"listRowCount,omitempty"`
	Total        int64 `json:"total,omitempty"`
	ListError    string `json:"listError,omitempty"`

	// ApproxPayloadBytes — размер JSON тела, как у cache.Service.Set (для справки).
	ApproxPayloadBytes int `json:"approxPayloadJsonBytes,omitempty"`
}

// DiagnosisRedisSection — замеры Redis / cache.Service.
type DiagnosisRedisSection struct {
	OK bool `json:"ok"`

	PingMs    float64 `json:"pingLatencyMs,omitempty"`
	PingError string  `json:"pingError,omitempty"`

	// CacheSetMs / CacheGetMs — те же вызовы, что в categoryService.List.
	CacheSetMs float64 `json:"cacheSetLatencyMs,omitempty"`
	CacheGetMs float64 `json:"cacheGetLatencyMs,omitempty"`
	CacheHit   bool    `json:"cacheGetHit,omitempty"`

	CacheError          string `json:"cacheError,omitempty"`
	ClientNotConfigured bool   `json:"clientNotConfigured,omitempty"`
}

// DiagnosisComparison — сравнение двух латентностей в миллисекундах.
type DiagnosisComparison struct {
	MongoDBMs float64 `json:"mongodbMs"`
	RedisMs   float64 `json:"redisMs"`
	// RedisFasterPercent — доля сокращения времени относительно MongoDB: (mongo-redis)/mongo*100 при mongo>redis.
	RedisFasterPercent *float64 `json:"redisFasterThanMongoPercent,omitempty"`
	RedisSpeedupFactor *float64 `json:"redisSpeedupFactor,omitempty"`
	Summary            string   `json:"summary"`
}

func msSince(t time.Time) float64 {
	return float64(time.Since(t).Microseconds()) / 1000.0
}

func buildComparison(mongoMs, redisMs float64, label string) *DiagnosisComparison {
	if mongoMs < 0 || redisMs < 0 {
		return nil
	}
	c := &DiagnosisComparison{
		MongoDBMs: mongoMs,
		RedisMs:   redisMs,
	}
	if mongoMs > 0 && redisMs < mongoMs {
		p := (mongoMs - redisMs) / mongoMs * 100
		c.RedisFasterPercent = &p
	}
	if redisMs > 0 {
		f := mongoMs / redisMs
		c.RedisSpeedupFactor = &f
	}
	if c.RedisFasterPercent != nil && c.RedisSpeedupFactor != nil {
		c.Summary = fmt.Sprintf(
			"%s: %.3f мс (Redis) против %.3f мс (MongoDB) — кеш примерно в %.2f раз быстрее по времени; латентность Redis короче на %.1f%% от времени БД",
			label, redisMs, mongoMs, *c.RedisSpeedupFactor, *c.RedisFasterPercent,
		)
	} else if c.RedisFasterPercent != nil {
		c.Summary = fmt.Sprintf("%s: %.3f мс (Redis) против %.3f мс (MongoDB); латентность Redis короче на %.1f%% от времени БД", label, redisMs, mongoMs, *c.RedisFasterPercent)
	} else {
		c.Summary = fmt.Sprintf("%s: при данных замерах кеш не быстрее (mongo=%.3f мс, redis=%.3f мс)", label, mongoMs, redisMs)
	}
	return c
}

func normalizePageLimit(page, limit int) (int, int, int) {
	if page < 1 {
		page = pagination.DefaultPage
	}
	if limit < 1 {
		limit = pagination.DefaultLimit
	}
	if limit > pagination.MaxLimit {
		limit = pagination.MaxLimit
	}
	offset := (page - 1) * limit
	return page, limit, offset
}

// RunDiagnosisParams — параметры прогона (те же page/limit, что у GET /categories).
type RunDiagnosisParams struct {
	Page  int
	Limit int
}

// RunDiagnosis выполняет замеры. Перед записью в кеш удаляется ключ страницы — как холодный промах для этой пары page/limit.
func RunDiagnosis(ctx context.Context, mongoDB *mongo.Database, rdb *redis.Client, repo categoryrepo.CategoryRepository, cacheSvc cache.Service, cacheTTL time.Duration, p RunDiagnosisParams) DiagnosisResponse {
	out := DiagnosisResponse{
		CheckedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}

	page, limit, offset := normalizePageLimit(p.Page, p.Limit)
	out.MongoDB.Page = page
	out.MongoDB.Limit = limit
	cacheKey := cache.CategoriesListKey(page, limit)
	out.MongoDB.CacheKey = cacheKey
	out.MongoDB.ListRepoMethod = "CategoryRepository.List(ctx, offset, limit) — тот же вызов, что при промахе кеша в CategoryService.List"

	// --- MongoDB: ping через runCommand ---
	t0 := time.Now()
	pingResult := mongoDB.RunCommand(ctx, bson.D{{Key: "ping", Value: 1}})
	out.MongoDB.SimpleMs = msSince(t0)
	out.MongoDB.SimpleQuery = `db.runCommand({ping: 1})`
	if pingResult.Err() != nil {
		out.MongoDB.SimpleError = pingResult.Err().Error()
	} else {
		out.MongoDB.OK = true
	}

	// --- Redis PING (сырой клиент, как в cache.NewRedisClient) ---
	if rdb == nil {
		out.Redis.ClientNotConfigured = true
		out.Redis.PingError = "клиент Redis не инициализирован"
	} else {
		tPing := time.Now()
		if pErr := rdb.Ping(ctx).Err(); pErr != nil {
			out.Redis.PingError = pErr.Error()
		} else {
			out.Redis.OK = true
			out.Redis.PingMs = msSince(tPing)
		}
	}

	// Сброс кеша страницы — воспроизводим промах без обхода сервиса.
	_ = cacheSvc.Del(ctx, cacheKey)
	out.Notes = fmt.Sprintf("Перед замером выполнен cache.Del(%q) — для этой пары page/limit следующий GET /categories получит промах кеша.", cacheKey)

	// --- Тот же путь, что CategoryService.List при промахе ---
	tList := time.Now()
	categories, total, listErr := repo.List(ctx, offset, limit)
	out.MongoDB.ListMs = msSince(tList)
	if listErr != nil {
		out.MongoDB.ListError = listErr.Error()
		out.Redis.CacheError = "пропущено: ошибка List"
		return out
	}
	out.MongoDB.ListRowCount = len(categories)
	out.MongoDB.Total = total

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}
	payload := categoriesListCachePayload{
		Categories: categories,
		Total:      total,
		TotalPages: totalPages,
	}
	if b, jErr := json.Marshal(payload); jErr == nil {
		out.MongoDB.ApproxPayloadBytes = len(b)
	}

	tSet := time.Now()
	if sErr := cacheSvc.Set(ctx, cacheKey, payload, cacheTTL); sErr != nil {
		out.Redis.CacheError = "cache.Set: " + sErr.Error()
		return out
	}
	out.Redis.CacheSetMs = msSince(tSet)

	var warm categoriesListCachePayload
	tGet := time.Now()
	hit, gErr := cacheSvc.Get(ctx, cacheKey, &warm)
	out.Redis.CacheGetMs = msSince(tGet)
	if gErr != nil {
		out.Redis.CacheError = "cache.Get: " + gErr.Error()
		return out
	}
	out.Redis.CacheHit = hit

	if out.MongoDB.SimpleError == "" && out.Redis.PingError == "" && rdb != nil {
		out.ComparisonSimple = buildComparison(out.MongoDB.SimpleMs, out.Redis.PingMs, "Минимальный round-trip (MongoDB ping vs PING)")
	}

	// Ядро: стоимость чтения из БД vs чтения из кеша тем же cache.Service.Get.
	out.ComparisonWorkload = buildComparison(out.MongoDB.ListMs, out.Redis.CacheGetMs, "Список категорий: repository.List vs cache hit (cache.Service.Get)")

	missTotal := out.MongoDB.ListMs + out.Redis.CacheSetMs
	out.ComparisonMissVsHit = buildComparison(missTotal, out.Redis.CacheGetMs, "Полный промах кеша (List+Set) vs cache hit (Get), как два запроса GET /categories")

	return out
}
