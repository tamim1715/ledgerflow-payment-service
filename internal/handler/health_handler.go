package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "UP",
	})
}

func (h *HealthHandler) Ready(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "READY",
	})
}

func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ALIVE",
	})
}

