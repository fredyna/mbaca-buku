package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORSMiddleware restricts browser requests to allowedOrigins.
//
// Browsers attach an Origin header to every POST, including same-origin ones,
// and gin-contrib/cors rejects an unlisted origin with a bare 403 before the
// handler ever runs. A hardcoded localhost list therefore broke logins on every
// deployment not served from localhost — so the list comes from configuration.
func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	})
}
