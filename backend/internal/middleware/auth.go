package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/fredy/mbaca-buku/pkg/utils"
)

func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			utils.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing authorization header")
			c.Abort()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid authorization format")
			c.Abort()
			return
		}

		userID, err := utils.ParseToken(parts[1], jwtSecret)
		if err != nil {
			utils.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}
