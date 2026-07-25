package router

import (
	"github.com/gin-gonic/gin"

	"github.com/fredy/mbaca-buku/internal/handler"
	"github.com/fredy/mbaca-buku/internal/middleware"
)

type RouterConfig struct {
	AuthHandler  *handler.AuthHandler
	EbookHandler *handler.EbookHandler
	JWTSecret    string
}

func Setup(r *gin.Engine, cfg *RouterConfig) {
	r.Use(middleware.CORSMiddleware())

	api := r.Group("/api")

	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true, "data": "ok"})
	})

	auth := api.Group("/auth")
	{
		auth.POST("/register", cfg.AuthHandler.Register)
		auth.POST("/login", cfg.AuthHandler.Login)
		auth.GET("/me", middleware.AuthMiddleware(cfg.JWTSecret), cfg.AuthHandler.Me)
	}

	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	{
		ebooks := protected.Group("/ebooks")
		{
			ebooks.GET("", cfg.EbookHandler.List)
			ebooks.GET("/:id", cfg.EbookHandler.GetByID)
			ebooks.POST("", cfg.EbookHandler.Upload)
			ebooks.PUT("/:id", cfg.EbookHandler.Update)
			ebooks.DELETE("/:id", cfg.EbookHandler.Delete)
			ebooks.GET("/:id/file", cfg.EbookHandler.GetFileURL)
		}
	}
}
