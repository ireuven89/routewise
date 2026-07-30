package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ireuven89/routewise/internal/service"
)

type DashboardHandler struct {
	jobService service.JobService
}

func NewDashboardHandler(jobSvc service.JobService) *DashboardHandler {
	return &DashboardHandler{jobService: jobSvc}
}

func (h *DashboardHandler) GetStats(c *gin.Context) {
	organizationID := c.GetUint("organization_id")

	stats, err := h.jobService.GetDashboardStats(organizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}
