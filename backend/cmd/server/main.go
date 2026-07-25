package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"

	"github.com/fredy/mbaca-buku/internal/config"
	"github.com/fredy/mbaca-buku/internal/handler"
	"github.com/fredy/mbaca-buku/internal/repository"
	"github.com/fredy/mbaca-buku/internal/router"
	"github.com/fredy/mbaca-buku/internal/service"
	"github.com/fredy/mbaca-buku/internal/storage"
	"github.com/fredy/mbaca-buku/pkg/cache"
	"github.com/fredy/mbaca-buku/pkg/database"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := database.RunMigrations(db, "./migrations"); err != nil {
		log.Fatal(err)
	}

	rdb, err := cache.NewRedisClient(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer rdb.Close()

	minioStorage, err := storage.NewMinIOClient(cfg)
	if err != nil {
		log.Fatal(err)
	}

	userRepo := repository.NewUserRepository(db)
	authService := service.NewAuthService(userRepo, cfg.JWTSecret)

	if err := authService.SeedDefaultUser(context.Background()); err != nil {
		log.Printf("Warning: failed to seed default user: %v", err)
	} else {
		log.Println("Default admin user ready")
	}

	authHandler := handler.NewAuthHandler(authService)

	ebookRepo := repository.NewEbookRepository(db)
	ebookService := service.NewEbookService(ebookRepo, minioStorage)
	ebookHandler := handler.NewEbookHandler(ebookService)

	r := gin.Default()

	router.Setup(r, &router.RouterConfig{
		AuthHandler:  authHandler,
		EbookHandler: ebookHandler,
		JWTSecret:    cfg.JWTSecret,
	})

	log.Printf("Server starting on :%s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatal(err)
	}
}
