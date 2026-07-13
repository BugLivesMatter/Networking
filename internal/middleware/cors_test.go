package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSAllowlistAndPreflight(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS([]string{"https://dashboard.example", "*"}))
	router.GET("/resource", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	request := httptest.NewRequest(http.MethodGet, "/resource", nil)
	request.Header.Set("Origin", "https://dashboard.example")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "https://dashboard.example" {
		t.Fatalf("allow origin = %q", got)
	}
	if got := response.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("allow credentials = %q", got)
	}
	if got := response.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("vary = %q", got)
	}

	request = httptest.NewRequest(http.MethodOptions, "/resource", nil)
	request.Header.Set("Origin", "https://dashboard.example")
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodGet, "/resource", nil)
	request.Header.Set("Origin", "https://evil.example")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected allow origin for denied origin: %q", got)
	}
}
