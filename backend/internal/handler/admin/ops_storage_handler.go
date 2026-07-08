package admin

import (
	"net/http"

	"github.com/Wei-Shaw/nexus/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

// GetStorageUsage returns best-effort storage usage for the ops dashboard.
// GET /api/v1/admin/ops/storage
func (h *OpsHandler) GetStorageUsage(c *gin.Context) {
	if h.opsService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request.Context()); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	data, err := h.opsService.GetStorageUsage(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, data)
}
