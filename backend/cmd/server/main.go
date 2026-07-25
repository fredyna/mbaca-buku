package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fredy/mbaca-buku/internal/config"
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

	_, err = storage.NewMinIOClient(cfg)
	if err != nil {
		log.Fatal(err)
	}

	r := gin.Default()

	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": "ok"})
	})

	log.Printf("Server starting on :%s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatal(err)
	}
}
