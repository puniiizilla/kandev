package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	dynamicruntime "github.com/kandev/kandev/internal/agent/runtime/dynamic"
)

type routingEvidenceReader interface {
	ListTaskRoutingEvidence(context.Context, string) (dynamicruntime.RoutingEvidenceResult, error)
}

func (h *TaskHandlers) httpGetRoutingEvidence(c *gin.Context) {
	taskID := c.Param("id")
	if h.service == nil || h.routingEvidence == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "routing evidence is not configured"})
		return
	}
	if err := h.service.AuthorizeTaskAccess(c.Request.Context(), taskID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	evidence, err := h.routingEvidence.ListTaskRoutingEvidence(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read routing evidence"})
		return
	}
	c.JSON(http.StatusOK, evidence)
}
