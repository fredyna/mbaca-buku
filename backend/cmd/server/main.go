package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	progressRepo := repository.NewProgressRepository(db)
	historyRepo := repository.NewHistoryRepository(db)
	readingService := service.NewReadingService(progressRepo, historyRepo, ebookRepo, rdb)
	historyService := service.NewHistoryService(historyRepo)
	readingHandler := handler.NewReadingHandler(readingService)
	historyHandler := handler.NewHistoryHandler(historyService)

	bookmarkRepo := repository.NewBookmarkRepository(db)
	bookmarkService := service.NewBookmarkService(bookmarkRepo)
	bookmarkHandler := handler.NewBookmarkHandler(bookmarkService)

	adminUserService := service.NewAdminUserService(userRepo)
	adminUserHandler := handler.NewAdminUserHandler(adminUserService)

	flusherCtx, cancelFlusher := context.WithCancel(context.Background())
	defer cancelFlusher()
	readingService.StartFlusher(flusherCtx)

	r := gin.Default()

	router.Setup(r, &router.RouterConfig{
		AuthHandler:      authHandler,
		EbookHandler:     ebookHandler,
		ReadingHandler:   readingHandler,
		HistoryHandler:   historyHandler,
		BookmarkHandler:  bookmarkHandler,
		AdminUserHandler: adminUserHandler,
		JWTSecret:        cfg.JWTSecret,
	})

	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	go func() {
		log.Printf("Server starting on :%s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown signal received, shutting down gracefully...")

	// Stop accepting new reading-progress cache writes from the periodic
	// flusher before doing the final flush below.
	cancelFlusher()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	readingService.FlushAll(shutdownCtx)

	log.Println("Server exited gracefully")
}
