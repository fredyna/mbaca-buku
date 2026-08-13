package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/fredy/mbaca-buku/internal/model"
	"github.com/fredy/mbaca-buku/pkg/utils"
)

// ActivityRecorder records a sign of life for an authenticated request.
// Satisfied by *service.ActivityService. A nil recorder disables recording,
// which keeps this middleware constructible without a database or Redis.
type ActivityRecorder interface {
	RecordAsync(userID, event string, meta model.RequestMeta)
}

func AuthMiddleware(jwtSecret string, recorder ActivityRecorder) gin.HandlerFunc {
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

		userID, role, err := utils.ParseToken(parts[1], jwtSecret)
		if err != nil {
			utils.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Set("role", role)

		// Only past this point is there a user to attribute activity to. The
		// recorder throttles internally, so this fires on every request but
		// writes at most one row per user per window. Metadata is read here,
		// while the context is still valid, not inside the recorder's goroutine.
		if recorder != nil {
			recorder.RecordAsync(userID, model.EventActive, utils.RequestMetaOf(c))
		}

		c.Next()
	}
}
