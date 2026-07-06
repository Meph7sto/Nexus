package middleware

import (
	"github.com/Wei-Shaw/nexus/internal/service"

	"github.com/gin-gonic/gin"
)

func AdminComplianceGuard(settingService *service.SettingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Admin compliance guard disabled — always allow.
		c.Next()
	}
}
