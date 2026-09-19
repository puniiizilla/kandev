package handlers

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRoutingQualityEndpointIsRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	(&TaskHandlers{}).registerHTTP(router)
	if !registeredRoutes(router)["POST /api/v1/tasks/:id/routing-quality-results"] {
		t.Fatal("routing quality endpoint is not registered")
	}
	if !registeredRoutes(router)["GET /api/v1/tasks/:id/routing-evidence"] {
		t.Fatal("routing evidence endpoint is not registered")
	}
}
