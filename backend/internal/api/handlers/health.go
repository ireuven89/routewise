package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	db *sql.DB
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{
		db: db,
	}
}

type HealthResponse struct {
	Status    string         `json:"status"`
	Timestamp string         `json:"timestamp"`
	Database  DatabaseHealth `json:"database"`
}

type DatabaseHealth struct {
	Status      string           `json:"status"`      // "up" | "down"
	ResponseMs  int64            `json:"response_ms"` // ping latency
	Connections *ConnectionStats `json:"connections,omitempty"`
	Error       string           `json:"error,omitempty"` // if down
}

type ConnectionStats struct {
	Open  int `json:"open"`
	InUse int `json:"in_use"`
	Idle  int `json:"idle"`
	Max   int `json:"max"`
}

func (h *HealthHandler) Check(c *gin.Context) {
	health := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Check database with timing
	start := time.Now()
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		health.Status = "unhealthy"
		health.Database = DatabaseHealth{
			Status: "down",
			Error:  err.Error(),
		}
		c.JSON(http.StatusServiceUnavailable, health)
		return
	}

	responseMs := time.Since(start).Milliseconds()
	stats := h.db.Stats()

	health.Database = DatabaseHealth{
		Status:     "up",
		ResponseMs: responseMs,
		Connections: &ConnectionStats{
			Open:  stats.OpenConnections,
			InUse: stats.InUse,
			Idle:  stats.Idle,
			Max:   stats.MaxOpenConnections,
		},
	}

	c.JSON(http.StatusOK, health)
}

func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 1*time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"ready": false,
			"reason": "database unreachable",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ready": true})
}

func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"alive": true})
}
