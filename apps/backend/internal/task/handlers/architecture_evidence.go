package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/kandev/kandev/internal/auth/authn"
	"github.com/kandev/kandev/internal/task/service"
)

func (h *TaskHandlers) httpGetArchitectureEvidence(c *gin.Context) {
	value, err := h.service.GetArchitectureEvidence(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "architecture evidence unavailable"})
		return
	}
	if value == nil {
		c.JSON(http.StatusOK, gin.H{"state": "BASELINE_PENDING", "required": false})
		return
	}
	c.JSON(http.StatusOK, value)
}

func (h *TaskHandlers) httpRefreshArchitectureEvidence(c *gin.Context) {
	identity, _ := authn.FromGin(c)
	value, err := h.service.RefreshArchitectureEvidence(c.Request.Context(), c.Param("id"), identity.UserID)
	if err != nil && !errors.Is(err, service.ErrArchitectureEvidenceNotReady) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	status := http.StatusOK
	if err != nil {
		status = http.StatusConflict
	}
	c.JSON(status, value)
}

func (h *TaskHandlers) httpOverrideArchitectureEvidence(c *gin.Context) {
	var body struct {
		Required bool   `json:"required"`
		Reason   string `json:"reason"`
	}
	if c.ShouldBindJSON(&body) != nil || strings.TrimSpace(body.Reason) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "required and reason are required"})
		return
	}
	identity, ok := authn.FromGin(c)
	if !ok || strings.TrimSpace(identity.UserID) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "actor identity required"})
		return
	}
	value, err := h.service.OverrideArchitectureEvidence(c.Request.Context(), c.Param("id"), body.Required, body.Reason, identity.UserID)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, value)
}

func (h *TaskHandlers) httpGetArchitectureEvidenceDelta(c *gin.Context) {
	value, err := h.service.ArchitectureEvidenceDelta(c.Request.Context(), c.Param("id"), c.Param("diagramID"))
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", value)
}
