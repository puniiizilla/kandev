package architecture

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine, service *Service) {
	api := router.Group("/api/v1/architecture")
	api.GET("", func(c *gin.Context) { inventoryHTTP(c, service, false) })
	api.POST("/refresh", func(c *gin.Context) { inventoryHTTP(c, service, true) })
	api.GET("/diagrams/:diagramID/source", func(c *gin.Context) {
		body, err := service.Source(c.Request.Context(), c.Query("workspace_id"), c.Param("diagramID"), c.Query("revision"))
		if err != nil {
			writeError(c, err)
			return
		}
		c.Data(http.StatusOK, "application/json; charset=utf-8", body)
	})
	api.GET("/diagrams/:diagramID/render", func(c *gin.Context) {
		body, err := service.Render(c.Request.Context(), c.Query("workspace_id"), c.Param("diagramID"), c.Query("revision"), c.Query("refresh") == "true")
		if err != nil {
			writeError(c, err)
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", body)
	})
}

func inventoryHTTP(c *gin.Context, service *Service, refresh bool) {
	inventory, err := service.Inventory(c.Request.Context(), c.Query("workspace_id"), refresh)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, inventory)
}

func writeError(c *gin.Context, err error) {
	code := ErrorCode(err)
	status := http.StatusUnprocessableEntity
	if code == CodeBindingMissing || code == CodeBindingInvalid || code == CodePathInvalid {
		status = http.StatusBadRequest
	}
	if code == CodeDiagramNotFound {
		status = http.StatusNotFound
	}
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": err.Error()}})
}
