package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lab2/rest-api/internal/cluster/domain"
	"github.com/lab2/rest-api/internal/cluster/source"
)

func TestTopology(t *testing.T) {
	router := testRouter()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/cluster/topology", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body.String())
	}
	var snapshot domain.Snapshot
	if err := json.Unmarshal(response.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(snapshot.Services) != 6 {
		t.Errorf("services = %d, want 6", len(snapshot.Services))
	}
}

func TestRunScenario(t *testing.T) {
	router := testRouter()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/demo/scenarios/latency", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body.String())
	}
	var snapshot domain.Snapshot
	if err := json.Unmarshal(response.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if snapshot.Events[0].Status != domain.StatusDegraded {
		t.Errorf("latest event status = %q, want degraded", snapshot.Events[0].Status)
	}
}

func TestRunUnknownScenario(t *testing.T) {
	router := testRouter()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/demo/scenarios/unknown", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}

func testRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	demo := source.NewDemoSource(time.Second)
	router := gin.New()
	New(demo, demo).RegisterRoutes(router)
	return router
}
