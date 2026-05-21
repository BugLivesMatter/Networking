package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	amqp091 "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type checkResult struct {
	Status    string  `json:"status"`
	LatencyMs float64 `json:"latency_ms,omitempty"`
	Error     string  `json:"error,omitempty"`
}

type readyResponse struct {
	Status    string                 `json:"status"`
	Timestamp string                 `json:"timestamp"`
	Checks    map[string]checkResult `json:"checks"`
}

// LiveHandler — минимальный liveness-зонд без проверки внешних зависимостей.
// Kubernetes перезапускает контейнер только при ошибке этого хэндлера.
//
// @Summary Liveness probe (K8s)
// @Description Минимальная проверка: процесс жив. Без обращений к внешним сервисам.
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health/live [get]
func LiveHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// ReadyHandler — readiness-зонд: проверяет MongoDB, Redis и RabbitMQ.
// Возвращает 503, если хотя бы одна критическая зависимость недоступна.
// Redis: если nil — статус "disabled" (не блокирует readiness).
//
// @Summary Readiness probe (K8s)
// @Description Проверяет подключение к MongoDB, Redis и RabbitMQ. 503 при деградации.
// @Tags health
// @Produce json
// @Success 200 {object} readyResponse
// @Failure 503 {object} readyResponse
// @Router /health/ready [get]
func ReadyHandler(mongoDB *mongo.Database, rdb *redis.Client, amqpConn *amqp091.Connection) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		checks := make(map[string]checkResult)
		allOK := true

		// MongoDB
		t0 := time.Now()
		if err := mongoDB.RunCommand(ctx, bson.D{{Key: "ping", Value: 1}}).Err(); err != nil {
			checks["mongodb"] = checkResult{Status: "error", Error: err.Error()}
			allOK = false
		} else {
			checks["mongodb"] = checkResult{Status: "ok", LatencyMs: msSince(t0)}
		}

		// Redis (опционально — если не настроен, не блокирует readiness)
		if rdb != nil {
			t1 := time.Now()
			if err := rdb.Ping(ctx).Err(); err != nil {
				checks["redis"] = checkResult{Status: "error", Error: err.Error()}
				allOK = false
			} else {
				checks["redis"] = checkResult{Status: "ok", LatencyMs: msSince(t1)}
			}
		} else {
			checks["redis"] = checkResult{Status: "disabled"}
		}

		// RabbitMQ
		if amqpConn != nil && !amqpConn.IsClosed() {
			checks["rabbitmq"] = checkResult{Status: "ok"}
		} else {
			checks["rabbitmq"] = checkResult{Status: "error", Error: "connection closed or not initialized"}
			allOK = false
		}

		status := "ok"
		code := http.StatusOK
		if !allOK {
			status = "degraded"
			code = http.StatusServiceUnavailable
		}

		c.JSON(code, readyResponse{
			Status:    status,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Checks:    checks,
		})
	}
}
