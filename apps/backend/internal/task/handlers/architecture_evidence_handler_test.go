package handlers

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestArchitectureEvidenceRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	(&TaskHandlers{}).registerHTTP(router)
	requireRoutes(t, router,
		"GET /api/v1/tasks/:id/architecture-evidence",
		"POST /api/v1/tasks/:id/architecture-evidence/refresh",
		"POST /api/v1/tasks/:id/architecture-evidence/override",
		"GET /api/v1/tasks/:id/architecture-evidence/delta/:diagramID",
	)
}
