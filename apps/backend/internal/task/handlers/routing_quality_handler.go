package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	dynamicruntime "github.com/kandev/kandev/internal/agent/runtime/dynamic"
	"github.com/kandev/kandev/internal/orchestrator"
)

type routeQualityCommand interface {
	RecordRouteQualityResult(context.Context, dynamicruntime.RouteQualityResult) (*orchestrator.RouteActionResult, error)
}

type routeQualityRequest struct {
	SessionID       string                       `json:"session_id" binding:"required"`
	RouteAttemptID  string                       `json:"route_attempt_id" binding:"required"`
	Result          dynamicruntime.QualityResult `json:"result" binding:"required"`
	Source          dynamicruntime.QualitySource `json:"source" binding:"required"`
	SourceReference string                       `json:"source_reference" binding:"required"`
}

func (h *TaskHandlers) httpRecordRoutingQualityResult(c *gin.Context) {
	taskID := c.Param("id")
	if h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "task service is not configured"})
		return
	}
	if err := h.service.AuthorizeTaskAccess(c.Request.Context(), taskID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	command, ok := h.orchestrator.(routeQualityCommand)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "routing quality command is not configured"})
		return
	}
	var request routeQualityRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid routing quality result"})
		return
	}
	result, err := command.RecordRouteQualityResult(c.Request.Context(), dynamicruntime.RouteQualityResult{
		TaskID: taskID, SessionID: request.SessionID, RouteAttemptID: request.RouteAttemptID,
		Result: request.Result, Source: request.Source, SourceReference: request.SourceReference,
	})
	if err != nil {
		switch {
		case errors.Is(err, dynamicruntime.ErrQualityResultOwnership):
			c.JSON(http.StatusNotFound, gin.H{"error": "routing attempt not found"})
		case errors.Is(err, dynamicruntime.ErrDuplicateQualityResult), errors.Is(err, dynamicruntime.ErrStaleQualityResult), errors.Is(err, dynamicruntime.ErrStaleGeneration):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"route": result})
}
