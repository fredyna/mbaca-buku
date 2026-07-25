package router

import (
	"github.com/gin-gonic/gin"

	"github.com/fredy/mbaca-buku/internal/handler"
	"github.com/fredy/mbaca-buku/internal/middleware"
)

func Setup(r *gin.Engine, cfg *RouterConfig) {
	r.Use(middleware.CORSMiddleware())

	api := r.Group("/api")

	auth := api.Group("/auth")
	{
		auth.POST("/register", cfg.AuthHandler.Register)
		auth.POST("/login", cfg.AuthHandler.Login)
		auth.GET("/me", middleware.AuthMiddleware(cfg.JWTSecret), cfg.AuthHandler.Me)
	}
}

type RouterConfig struct {
	AuthHandler *handler.AuthHandler
	JWTSecret   string
}
