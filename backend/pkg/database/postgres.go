package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/lib/pq"

	"github.com/fredy/mbaca-buku/internal/config"
)

func Connect(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DBDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	log.Println("Connected to PostgreSQL")
	return db, nil
}

func RunMigrations(db *sql.DB, migrationsDir string) error {
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return fmt.Errorf("failed to find migrations: %w", err)
	}

	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", f, err)
		}

		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("failed to run migration %s: %w", f, err)
		}

		log.Printf("Migration applied: %s", filepath.Base(f))
	}

	return nil
}
