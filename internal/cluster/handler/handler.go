package handler

import (
	"encoding/json"
	"errors"
	"net/http"

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
	router.GET("/api/v1/cluster/events", h.Events)
	router.POST("/api/v1/demo/scenarios/:scenario", h.RunScenario)
}

func (h *Handler) Topology(c *gin.Context) {
	snapshot, err := h.cluster.Snapshot(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "cluster topology unavailable"})
		return
	}
	c.JSON(http.StatusOK, snapshot)
}

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

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-events:
			payload, marshalErr := json.Marshal(event)
			if marshalErr != nil {
				continue
			}
			c.SSEvent("cluster-event", string(payload))
			c.Writer.Flush()
		}
	}
}
