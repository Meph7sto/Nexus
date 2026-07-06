package middleware

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type AdminPermissionMiddleware func(resource service.AdminPermissionResource, action service.AdminPermissionAction) gin.HandlerFunc

func NewAdminPermissionMiddleware(repo service.AdminPermissionRepository) AdminPermissionMiddleware {
	return func(resource service.AdminPermissionResource, action service.AdminPermissionAction) gin.HandlerFunc {
		return func(c *gin.Context) {
			role, _ := c.Get(string(ContextKeyUserRole))
			switch role {
			case service.RoleSuperAdmin:
				c.Next()
				return
			case service.RoleAdmin:
				subject, ok := GetAuthSubjectFromContext(c)
				if !ok {
					AbortWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization required")
					return
				}
				if repo == nil {
					AbortWithError(c, http.StatusForbidden, "ADMIN_PERMISSION_DENIED", "Admin permission denied")
					return
				}
				allowed, err := repo.HasPermission(c.Request.Context(), subject.UserID, resource, action)
				if err != nil {
					AbortWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
					return
				}
				if !allowed {
					AbortWithError(c, http.StatusForbidden, "ADMIN_PERMISSION_DENIED", "Admin permission denied")
					return
				}
				c.Next()
				return
			default:
				AbortWithError(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
				return
			}
		}
	}
}
