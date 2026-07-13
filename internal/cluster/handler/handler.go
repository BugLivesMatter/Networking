package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lab2/rest-api/internal/cluster/source"
)

type Handler struct {
	cluster   source.ClusterSource
	scenarios source.ScenarioRunner
}

func New(cluster source.ClusterSource, scenarios source.ScenarioRunner) *Handler {
	return &Handler{cluster: cluster, scenarios: scenarios}
}

func (h *Handler) RegisterRoutes(router gin.IRouter) {
	router.GET("/api/v1/cluster/topology", h.Topology)
	router.GET("/api/v1/cluster/services/:serviceID", h.Service)
	router.GET("/api/v1/cluster/events", h.Events)
	router.POST("/api/v1/demo/scenarios/:scenario", h.RunScenario)
}

// Service returns details for one cluster service.
// @Summary Получить сервис кластера
// @Tags cluster
// @Produce json
// @Param serviceID path string true "Идентификатор сервиса"
// @Success 200 {object} domain.Service
// @Failure 404 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /api/v1/cluster/services/{serviceID} [get]
func (h *Handler) Service(c *gin.Context) {
	snapshot, err := h.cluster.Snapshot(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "cluster topology unavailable"})
		return
	}
	serviceID := c.Param("serviceID")
	for _, service := range snapshot.Services {
		if service.ID == serviceID {
			c.JSON(http.StatusOK, service)
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
}

// Topology returns the complete cluster snapshot.
// @Summary Получить topology кластера
// @Tags cluster
// @Produce json
// @Success 200 {object} domain.Snapshot
// @Failure 503 {object} map[string]string
// @Router /api/v1/cluster/topology [get]
func (h *Handler) Topology(c *gin.Context) {
	snapshot, err := h.cluster.Snapshot(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "cluster topology unavailable"})
		return
	}
	c.JSON(http.StatusOK, snapshot)
}

// RunScenario applies a safe demo scenario.
// @Summary Запустить demo-сценарий
// @Tags cluster
// @Produce json
// @Param scenario path string true "Сценарий (latency, crash, scale, recover)"
// @Success 200 {object} domain.Snapshot
// @Failure 400 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /api/v1/demo/scenarios/{scenario} [post]
func (h *Handler) RunScenario(c *gin.Context) {
	snapshot, err := h.scenarios.RunScenario(c.Request.Context(), c.Param("scenario"))
	if err != nil {
		if errors.Is(err, source.ErrUnknownScenario) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "scenario unavailable"})
		return
	}
	c.JSON(http.StatusOK, snapshot)
}

// Events streams cluster events using Server-Sent Events.
// @Summary SSE-поток событий кластера
// @Tags cluster
// @Produce text/event-stream
// @Success 200 {string} string
// @Failure 503 {object} map[string]string
// @Router /api/v1/cluster/events [get]
func (h *Handler) Events(c *gin.Context) {
	ctx := c.Request.Context()
	events, err := h.cluster.Subscribe(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "event stream unavailable"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	heartbeatInterval := sseHeartbeatInterval
	if heartbeatInterval <= 0 {
		heartbeatInterval = 15 * time.Second
	}
	heartbeat := time.NewTicker(heartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			_, _ = c.Writer.WriteString(": heartbeat\n\n")
			c.Writer.Flush()
		case event, ok := <-events:
			if !ok {
				return
			}
			payload, marshalErr := json.Marshal(event)
			if marshalErr != nil {
				continue
			}
			c.SSEvent("cluster-event", string(payload))
			c.Writer.Flush()
		}
	}
}

// Kept as a variable so tests can use a short heartbeat interval without
// waiting for the production 15-second interval.
var sseHeartbeatInterval = 15 * time.Second
