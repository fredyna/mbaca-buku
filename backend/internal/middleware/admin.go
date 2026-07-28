package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fredy/mbaca-buku/pkg/utils"
)

// AdminMiddleware restricts access to users whose JWT role claim is "admin".
// It must run after AuthMiddleware, which populates the "role" context value.
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("role")
		if role != "admin" {
			utils.ErrorResponse(c, http.StatusForbidden, "FORBIDDEN", "admin access required")
			c.Abort()
			return
		}

		c.Next()
	}
}
