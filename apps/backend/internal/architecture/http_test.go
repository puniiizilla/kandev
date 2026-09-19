package architecture

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestArchitectureRoutesFailClosedWithoutWorkspace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router, NewService(fakeRepositories{}, t.TempDir()))
	for _, request := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/architecture"},
		{http.MethodPost, "/api/v1/architecture/refresh"},
		{http.MethodGet, "/api/v1/architecture/diagrams/SYSTEM_OVERVIEW/source?revision=abc"},
		{http.MethodGet, "/api/v1/architecture/diagrams/SYSTEM_OVERVIEW/render?revision=abc"},
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(request.method, request.path, nil))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s %s status = %d", request.method, request.path, recorder.Code)
		}
	}
}
