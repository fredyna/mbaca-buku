# Mbaca Buku Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a full-stack ebook reading platform with Go/Gin backend, React/Vite frontend, PostgreSQL, Redis, MinIO, all orchestrated via Docker Compose.

**Architecture:** Monorepo with `backend/` and `frontend/` directories. Clean architecture backend (handler → service → repository). React SPA served via Nginx reverse proxy that also forwards `/api/*` to the Go API on port 8080. Redis caches active reading progress; MinIO stores PDF files.

**Tech Stack:** Go 1.26 + Gin, React 18 + Vite + TypeScript, Tailwind CSS 3, react-pdf, PostgreSQL 16, Redis 7, MinIO, Nginx, Docker Compose

## Global Constraints

- Go module path: `github.com/fredy/mbaca-buku`
- All UUIDs generated server-side via `github.com/google/uuid`
- Passwords hashed with bcrypt (cost 10)
- JWT signed with HS256, 24h expiry
- API responses always use `{"success": bool, "data": ..., "error": ...}` envelope
- Default seed user: name=`admin`, email=`admin@mbacabuku.com`, password=`12345`
- PostgreSQL database name: `mbaca_buku`
- Redis key format: `user:{userId}:book:{bookId}:last_page`
- MinIO bucket name: `mbaca-buku`
- All timestamps in UTC

---

### Task 1: Project Scaffolding & Docker Infrastructure

**Files:**
- Create: `docker-compose.yml`
- Create: `.env`
- Create: `.env.example`
- Create: `.gitignore`
- Create: `nginx/nginx.conf`
- Create: `backend/Dockerfile`
- Create: `backend/go.mod`
- Create: `backend/cmd/server/main.go` (minimal hello-world)
- Create: `frontend/Dockerfile`
- Create: `frontend/package.json` (via `npm create vite`)
- Create: `frontend/vite.config.ts`

**Interfaces:**
- Consumes: nothing (first task)
- Produces: runnable `docker-compose up --build` that starts all 6 services (postgres, redis, minio, backend, frontend, nginx) and responds on port 80

- [ ] **Step 1: Initialize git repo**

```bash
cd /Users/fredy/Documents/GitHub/mbaca-buku
git init
```

- [ ] **Step 2: Create `.gitignore`**

```gitignore
# Go
backend/tmp/
backend/vendor/

# Node
frontend/node_modules/
frontend/dist/

# Environment
.env

# IDE
.idea/
.vscode/
*.swp

# OS
.DS_Store

# Docker
docker-compose.override.yml
```

- [ ] **Step 3: Create `.env.example` and `.env`**

```env
# PostgreSQL
POSTGRES_USER=mbaca
POSTGRES_PASSWORD=mbaca_secret
POSTGRES_DB=mbaca_buku

# Redis
REDIS_URL=redis://redis:6379/0

# MinIO
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=minioadmin123
MINIO_ENDPOINT=minio:9000
MINIO_BUCKET=mbaca-buku
MINIO_USE_SSL=false

# Backend
JWT_SECRET=mbaca-buku-jwt-secret-change-in-production
SERVER_PORT=8080
DB_HOST=postgres
DB_PORT=5432
DB_USER=mbaca
DB_PASSWORD=mbaca_secret
DB_NAME=mbaca_buku
DB_SSLMODE=disable

# Frontend
VITE_API_URL=/api
```

Copy `.env.example` to `.env` (`.env` is gitignored, `.env.example` is committed).

- [ ] **Step 4: Initialize Go module**

```bash
cd /Users/fredy/Documents/GitHub/mbaca-buku
mkdir -p backend/cmd/server
cd backend
go mod init github.com/fredy/mbaca-buku
```

- [ ] **Step 5: Create minimal `backend/cmd/server/main.go`**

```go
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": "ok"})
	})

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
```

```bash
cd /Users/fredy/Documents/GitHub/mbaca-buku/backend
go get github.com/gin-gonic/gin
```

- [ ] **Step 6: Create `backend/Dockerfile`**

```dockerfile
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /server .
COPY migrations/ ./migrations/

EXPOSE 8080

CMD ["./server"]
```

- [ ] **Step 7: Scaffold React+Vite frontend**

```bash
cd /Users/fredy/Documents/GitHub/mbaca-buku
npm create vite@latest frontend -- --template react-ts
cd frontend
npm install
npm install -D tailwindcss @tailwindcss/vite
npm install axios react-router-dom react-pdf pdfjs-dist
```

- [ ] **Step 8: Configure Tailwind in `frontend/src/index.css`**

```css
@import "tailwindcss";
```

- [ ] **Step 9: Configure `frontend/vite.config.ts`**

```typescript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    host: '0.0.0.0',
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://backend:8080',
        changeOrigin: true,
      },
    },
  },
})
```

- [ ] **Step 10: Create `frontend/Dockerfile`**

```dockerfile
FROM node:20-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
```

- [ ] **Step 11: Create `nginx/nginx.conf`**

```nginx
upstream backend {
    server backend:8080;
}

server {
    listen 80;
    server_name localhost;

    root /usr/share/nginx/html;
    index index.html;

    location /api/ {
        proxy_pass http://backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        client_max_body_size 100M;
    }

    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

- [ ] **Step 12: Create `docker-compose.yml`**

```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: ${POSTGRES_DB}
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER} -d ${POSTGRES_DB}"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 5

  minio:
    image: minio/minio
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - minio_data:/data
    command: server /data --console-address ":9001"
    healthcheck:
      test: ["CMD", "mc", "ready", "local"]
      interval: 5s
      timeout: 5s
      retries: 5

  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      SERVER_PORT: ${SERVER_PORT}
      DB_HOST: ${DB_HOST}
      DB_PORT: ${DB_PORT}
      DB_USER: ${DB_USER}
      DB_PASSWORD: ${DB_PASSWORD}
      DB_NAME: ${DB_NAME}
      DB_SSLMODE: ${DB_SSLMODE}
      REDIS_URL: ${REDIS_URL}
      MINIO_ENDPOINT: ${MINIO_ENDPOINT}
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
      MINIO_BUCKET: ${MINIO_BUCKET}
      MINIO_USE_SSL: ${MINIO_USE_SSL}
      JWT_SECRET: ${JWT_SECRET}
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      minio:
        condition: service_started

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    volumes:
      - frontend_build:/usr/share/nginx/html

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/conf.d/default.conf
      - frontend_build:/usr/share/nginx/html
    depends_on:
      - backend
      - frontend

volumes:
  pgdata:
  minio_data:
  frontend_build:
```

- [ ] **Step 13: Create placeholder migration directory**

```bash
mkdir -p /Users/fredy/Documents/GitHub/mbaca-buku/backend/migrations
touch /Users/fredy/Documents/GitHub/mbaca-buku/backend/migrations/.gitkeep
```

- [ ] **Step 14: Verify Docker Compose starts all services**

```bash
cd /Users/fredy/Documents/GitHub/mbaca-buku
docker-compose up --build -d
```

Expected: all 6 services start. `curl http://localhost/api/health` returns `{"success":true,"data":"ok"}`.

```bash
docker-compose down
```

- [ ] **Step 15: Commit**

```bash
git add .
git commit -m "feat: project scaffolding with Docker Compose, Go backend, React frontend, Nginx proxy"
```

---

### Task 2: Database, Redis, MinIO Connections & Migration

**Files:**
- Create: `backend/internal/config/config.go`
- Create: `backend/pkg/database/postgres.go`
- Create: `backend/pkg/cache/redis.go`
- Create: `backend/internal/storage/minio.go`
- Create: `backend/migrations/001_init.sql`
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/go.mod` (new dependencies)

**Interfaces:**
- Consumes: Docker services from Task 1 (postgres, redis, minio containers)
- Produces:
  - `config.Load() *Config` — loads all env vars into a struct
  - `database.Connect(cfg) (*sql.DB, error)` — returns a connection pool
  - `cache.NewRedisClient(cfg) (*redis.Client, error)` — returns Redis client
  - `storage.NewMinIOClient(cfg) (*MinIOStorage, error)` — returns MinIO storage wrapper
  - `storage.MinIOStorage.UploadFile(ctx, objectName, reader, size, contentType) (string, error)` — returns the object key
  - `storage.MinIOStorage.GetPresignedURL(ctx, objectName, expiry) (string, error)` — returns presigned URL
  - `storage.MinIOStorage.DeleteFile(ctx, objectName) error`
  - All 5 database tables created via migration

- [ ] **Step 1: Create `backend/internal/config/config.go`**

```go
package config

import "os"

type Config struct {
	ServerPort    string
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	DBSSLMode     string
	RedisURL      string
	MinIOEndpoint string
	MinIOUser     string
	MinIOPassword string
	MinIOBucket   string
	MinIOUseSSL   bool
	JWTSecret     string
}

func Load() *Config {
	return &Config{
		ServerPort:    getEnv("SERVER_PORT", "8080"),
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "mbaca"),
		DBPassword:    getEnv("DB_PASSWORD", "mbaca_secret"),
		DBName:        getEnv("DB_NAME", "mbaca_buku"),
		DBSSLMode:     getEnv("DB_SSLMODE", "disable"),
		RedisURL:      getEnv("REDIS_URL", "redis://localhost:6379/0"),
		MinIOEndpoint: getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOUser:     getEnv("MINIO_ROOT_USER", "minioadmin"),
		MinIOPassword: getEnv("MINIO_ROOT_PASSWORD", "minioadmin123"),
		MinIOBucket:   getEnv("MINIO_BUCKET", "mbaca-buku"),
		MinIOUseSSL:   getEnv("MINIO_USE_SSL", "false") == "true",
		JWTSecret:     getEnv("JWT_SECRET", "default-secret"),
	}
}

func (c *Config) DBDSN() string {
	return "host=" + c.DBHost +
		" port=" + c.DBPort +
		" user=" + c.DBUser +
		" password=" + c.DBPassword +
		" dbname=" + c.DBName +
		" sslmode=" + c.DBSSLMode
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

- [ ] **Step 2: Create `backend/migrations/001_init.sql`**

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ebooks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(255) NOT NULL,
    author VARCHAR(255) DEFAULT '',
    cover_url TEXT DEFAULT '',
    file_url TEXT NOT NULL,
    file_size BIGINT DEFAULT 0,
    total_pages INT NOT NULL DEFAULT 0,
    uploaded_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS reading_progress (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ebook_id UUID NOT NULL REFERENCES ebooks(id) ON DELETE CASCADE,
    last_page INT NOT NULL DEFAULT 1,
    status VARCHAR(20) NOT NULL DEFAULT 'reading',
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, ebook_id)
);

CREATE TABLE IF NOT EXISTS bookmarks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ebook_id UUID NOT NULL REFERENCES ebooks(id) ON DELETE CASCADE,
    page_number INT NOT NULL,
    note TEXT DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, ebook_id, page_number)
);

CREATE TABLE IF NOT EXISTS history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ebook_id UUID NOT NULL REFERENCES ebooks(id) ON DELETE CASCADE,
    opened_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_history_user_opened ON history(user_id, opened_at DESC);
CREATE INDEX IF NOT EXISTS idx_reading_progress_user ON reading_progress(user_id);
CREATE INDEX IF NOT EXISTS idx_bookmarks_user_ebook ON bookmarks(user_id, ebook_id);
```

- [ ] **Step 3: Create `backend/pkg/database/postgres.go`**

```go
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
```

- [ ] **Step 4: Create `backend/pkg/cache/redis.go`**

```go
package cache

import (
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"

	"github.com/fredy/mbaca-buku/internal/config"
)

func NewRedisClient(cfg *config.Config) (*redis.Client, error) {
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	log.Println("Connected to Redis")
	return client, nil
}
```

- [ ] **Step 5: Create `backend/internal/storage/minio.go`**

```go
package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/fredy/mbaca-buku/internal/config"
)

type MinIOStorage struct {
	client *minio.Client
	bucket string
}

func NewMinIOClient(cfg *config.Config) (*MinIOStorage, error) {
	client, err := minio.New(cfg.MinIOEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIOUser, cfg.MinIOPassword, ""),
		Secure: cfg.MinIOUseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.MinIOBucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket: %w", err)
	}

	if !exists {
		if err := client.MakeBucket(ctx, cfg.MinIOBucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
		log.Printf("Created MinIO bucket: %s", cfg.MinIOBucket)
	}

	log.Println("Connected to MinIO")
	return &MinIOStorage{client: client, bucket: cfg.MinIOBucket}, nil
}

func (s *MinIOStorage) UploadFile(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) (string, error) {
	_, err := s.client.PutObject(ctx, s.bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}
	return objectName, nil
}

func (s *MinIOStorage) GetPresignedURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	url, err := s.client.PresignedGetObject(ctx, s.bucket, objectName, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get presigned URL: %w", err)
	}
	return url.String(), nil
}

func (s *MinIOStorage) DeleteFile(ctx context.Context, objectName string) error {
	return s.client.RemoveObject(ctx, s.bucket, objectName, minio.RemoveObjectOptions{})
}
```

- [ ] **Step 6: Install Go dependencies**

```bash
cd /Users/fredy/Documents/GitHub/mbaca-buku/backend
go get github.com/lib/pq
go get github.com/redis/go-redis/v9
go get github.com/minio/minio-go/v7
```

- [ ] **Step 7: Update `backend/cmd/server/main.go` to connect all services**

```go
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
```

- [ ] **Step 8: Test with Docker Compose**

```bash
cd /Users/fredy/Documents/GitHub/mbaca-buku
docker-compose up --build -d
docker-compose logs backend
```

Expected: backend logs show "Connected to PostgreSQL", "Connected to Redis", "Connected to MinIO", "Migration applied: 001_init.sql".

```bash
docker-compose down
```

- [ ] **Step 9: Commit**

```bash
git add .
git commit -m "feat: database, Redis, MinIO connections and initial migration"
```

---

### Task 3: Utility Packages (JWT, Password Hash, API Response)

**Files:**
- Create: `backend/pkg/utils/jwt.go`
- Create: `backend/pkg/utils/hash.go`
- Create: `backend/pkg/utils/response.go`

**Interfaces:**
- Consumes: `config.Config.JWTSecret`
- Produces:
  - `utils.GenerateToken(userID string, secret string) (string, error)` — returns signed JWT string
  - `utils.ParseToken(tokenString string, secret string) (string, error)` — returns userID from claims
  - `utils.HashPassword(password string) (string, error)` — returns bcrypt hash
  - `utils.CheckPassword(hash, password string) bool`
  - `utils.SuccessResponse(c *gin.Context, code int, data interface{})` — writes `{"success":true,"data":...}`
  - `utils.ErrorResponse(c *gin.Context, code int, errCode string, message string)` — writes `{"success":false,"error":{...}}`
  - `utils.PaginatedResponse(c *gin.Context, data interface{}, page, perPage, total int)` — writes `{"success":true,"data":...,"meta":{...}}`

- [ ] **Step 1: Create `backend/pkg/utils/hash.go`**

```go
package utils

import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	return string(bytes), err
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
```

- [ ] **Step 2: Create `backend/pkg/utils/jwt.go`**

```go
package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateToken(userID string, secret string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseToken(tokenString string, secret string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid user_id in token")
	}

	return userID, nil
}
```

- [ ] **Step 3: Create `backend/pkg/utils/response.go`**

```go
package utils

import "github.com/gin-gonic/gin"

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *APIMeta    `json:"meta,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type APIMeta struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

func SuccessResponse(c *gin.Context, code int, data interface{}) {
	c.JSON(code, APIResponse{Success: true, Data: data})
}

func ErrorResponse(c *gin.Context, code int, errCode string, message string) {
	c.JSON(code, APIResponse{Success: false, Error: &APIError{Code: errCode, Message: message}})
}

func PaginatedResponse(c *gin.Context, data interface{}, page, perPage, total int) {
	c.JSON(200, APIResponse{
		Success: true,
		Data:    data,
		Meta:    &APIMeta{Page: page, PerPage: perPage, Total: total},
	})
}
```

- [ ] **Step 4: Install Go dependencies**

```bash
cd /Users/fredy/Documents/GitHub/mbaca-buku/backend
go get golang.org/x/crypto/bcrypt
go get github.com/golang-jwt/jwt/v5
```

- [ ] **Step 5: Verify compilation**

```bash
cd /Users/fredy/Documents/GitHub/mbaca-buku/backend
go build ./...
```

Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add .
git commit -m "feat: utility packages for JWT, password hashing, and API responses"
```

---

### Task 4: Auth — Models, DTOs, Repository, Service, Handler, Middleware, Seed

**Files:**
- Create: `backend/internal/model/user.go`
- Create: `backend/internal/dto/auth_dto.go`
- Create: `backend/internal/repository/user_repo.go`
- Create: `backend/internal/service/auth_service.go`
- Create: `backend/internal/handler/auth_handler.go`
- Create: `backend/internal/middleware/auth.go`
- Create: `backend/internal/middleware/cors.go`
- Create: `backend/internal/router/router.go`
- Modify: `backend/cmd/server/main.go`

**Interfaces:**
- Consumes:
  - `database.Connect()` → `*sql.DB`
  - `utils.HashPassword()`, `utils.CheckPassword()`, `utils.GenerateToken()`, `utils.ParseToken()`
  - `utils.SuccessResponse()`, `utils.ErrorResponse()`
- Produces:
  - `model.User` struct with `ID, Name, Email, PasswordHash, CreatedAt, UpdatedAt`
  - `repository.UserRepository.Create(ctx, user) error`
  - `repository.UserRepository.GetByEmail(ctx, email) (*model.User, error)`
  - `repository.UserRepository.GetByID(ctx, id) (*model.User, error)`
  - `service.AuthService.Register(ctx, dto) (*model.User, string, error)` — returns user + JWT token
  - `service.AuthService.Login(ctx, dto) (*model.User, string, error)` — returns user + JWT token
  - `service.AuthService.SeedDefaultUser(ctx) error` — creates admin/12345 if not exists
  - `middleware.AuthMiddleware(secret) gin.HandlerFunc` — sets `user_id` in gin context
  - Endpoints: `POST /api/auth/register`, `POST /api/auth/login`, `GET /api/auth/me`

- [ ] **Step 1: Create `backend/internal/model/user.go`**

```go
package model

import "time"

type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
```

- [ ] **Step 2: Create `backend/internal/dto/auth_dto.go`**

```go
package dto

type RegisterRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=255"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=5"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	User  UserResponse `json:"user"`
	Token string       `json:"token"`
}

type UserResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}
```

- [ ] **Step 3: Create `backend/internal/repository/user_repo.go`**

```go
package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/fredy/mbaca-buku/internal/model"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	query := `INSERT INTO users (name, email, password_hash) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, query, user.Name, user.Email, user.PasswordHash).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	user := &model.User{}
	query := `SELECT id, name, email, password_hash, created_at, updated_at FROM users WHERE email = $1`
	err := r.db.QueryRowContext(ctx, query, email).
		Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	return user, err
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	user := &model.User{}
	query := `SELECT id, name, email, password_hash, created_at, updated_at FROM users WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).
		Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	return user, err
}
```

- [ ] **Step 4: Create `backend/internal/service/auth_service.go`**

```go
package service

import (
	"context"
	"fmt"

	"github.com/fredy/mbaca-buku/internal/dto"
	"github.com/fredy/mbaca-buku/internal/model"
	"github.com/fredy/mbaca-buku/internal/repository"
	"github.com/fredy/mbaca-buku/pkg/utils"
)

type AuthService struct {
	userRepo  *repository.UserRepository
	jwtSecret string
}

func NewAuthService(userRepo *repository.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{userRepo: userRepo, jwtSecret: jwtSecret}
}

func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
	existing, _ := s.userRepo.GetByEmail(ctx, req.Email)
	if existing != nil {
		return nil, fmt.Errorf("email already registered")
	}

	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &model.User{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: hash,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	token, err := utils.GenerateToken(user.ID, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &dto.AuthResponse{
		User:  dto.UserResponse{ID: user.ID, Name: user.Name, Email: user.Email},
		Token: token,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	if !utils.CheckPassword(user.PasswordHash, req.Password) {
		return nil, fmt.Errorf("invalid email or password")
	}

	token, err := utils.GenerateToken(user.ID, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &dto.AuthResponse{
		User:  dto.UserResponse{ID: user.ID, Name: user.Name, Email: user.Email},
		Token: token,
	}, nil
}

func (s *AuthService) SeedDefaultUser(ctx context.Context) error {
	existing, _ := s.userRepo.GetByEmail(ctx, "admin@mbacabuku.com")
	if existing != nil {
		return nil
	}

	hash, err := utils.HashPassword("12345")
	if err != nil {
		return err
	}

	user := &model.User{
		Name:         "admin",
		Email:        "admin@mbacabuku.com",
		PasswordHash: hash,
	}

	return s.userRepo.Create(ctx, user)
}
```

- [ ] **Step 5: Create `backend/internal/handler/auth_handler.go`**

```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fredy/mbaca-buku/internal/dto"
	"github.com/fredy/mbaca-buku/internal/service"
	"github.com/fredy/mbaca-buku/pkg/utils"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	resp, err := h.authService.Register(c.Request.Context(), req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusConflict, "REGISTER_ERROR", err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, resp)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, "AUTH_ERROR", err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, resp)
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetString("user_id")

	user, err := h.authService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, dto.UserResponse{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
	})
}
```

Then add `GetUserByID` to `auth_service.go`:

```go
func (s *AuthService) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	return s.userRepo.GetByID(ctx, id)
}
```

- [ ] **Step 6: Create `backend/internal/middleware/auth.go`**

```go
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
```

- [ ] **Step 7: Create `backend/internal/middleware/cors.go`**

```go
package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORSMiddleware() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost", "http://localhost:5173", "http://localhost:80"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	})
}
```

- [ ] **Step 8: Create `backend/internal/router/router.go`**

```go
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
```

- [ ] **Step 9: Update `backend/cmd/server/main.go` to wire auth**

```go
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

	_, err = storage.NewMinIOClient(cfg)
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

	r := gin.Default()

	router.Setup(r, &router.RouterConfig{
		AuthHandler: authHandler,
		JWTSecret:   cfg.JWTSecret,
	})

	log.Printf("Server starting on :%s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 10: Install CORS dependency**

```bash
cd /Users/fredy/Documents/GitHub/mbaca-buku/backend
go get github.com/gin-contrib/cors
```

- [ ] **Step 11: Test auth endpoints with Docker**

```bash
docker-compose up --build -d

# Register
curl -X POST http://localhost/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"test","email":"test@test.com","password":"test123"}'

# Login with seeded admin
curl -X POST http://localhost/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@mbacabuku.com","password":"12345"}'

# Me (use token from login response)
curl http://localhost/api/auth/me \
  -H "Authorization: Bearer <token>"

docker-compose down
```

Expected: register returns 201 with user+token, login returns 200 with user+token, me returns 200 with user profile.

- [ ] **Step 12: Commit**

```bash
git add .
git commit -m "feat: JWT auth with register, login, me endpoints and admin seed"
```

---

### Task 5: Ebook CRUD — Model, Repository, Service, Handler

**Files:**
- Create: `backend/internal/model/ebook.go`
- Create: `backend/internal/dto/ebook_dto.go`
- Create: `backend/internal/repository/ebook_repo.go`
- Create: `backend/internal/service/ebook_service.go`
- Create: `backend/internal/handler/ebook_handler.go`
- Modify: `backend/internal/router/router.go`
- Modify: `backend/cmd/server/main.go`

**Interfaces:**
- Consumes:
  - `storage.MinIOStorage.UploadFile()`, `.GetPresignedURL()`, `.DeleteFile()`
  - `middleware.AuthMiddleware()`
  - `utils.SuccessResponse()`, `utils.ErrorResponse()`, `utils.PaginatedResponse()`
- Produces:
  - `model.Ebook` struct with `ID, Title, Author, CoverURL, FileURL, FileSize, TotalPages, UploadedBy, CreatedAt, UpdatedAt`
  - `repository.EbookRepository.Create(ctx, ebook) error`
  - `repository.EbookRepository.GetByID(ctx, id) (*model.Ebook, error)`
  - `repository.EbookRepository.List(ctx, page, perPage) ([]*model.Ebook, int, error)` — returns ebooks + total count
  - `repository.EbookRepository.Update(ctx, ebook) error`
  - `repository.EbookRepository.Delete(ctx, id) error`
  - Endpoints: `GET/POST /api/ebooks`, `GET/PUT/DELETE /api/ebooks/:id`, `GET /api/ebooks/:id/file`

- [ ] **Step 1: Create `backend/internal/model/ebook.go`**

```go
package model

import "time"

type Ebook struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Author     string    `json:"author"`
	CoverURL   string    `json:"cover_url"`
	FileURL    string    `json:"file_url"`
	FileSize   int64     `json:"file_size"`
	TotalPages int       `json:"total_pages"`
	UploadedBy string    `json:"uploaded_by"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
```

- [ ] **Step 2: Create `backend/internal/dto/ebook_dto.go`**

```go
package dto

import "time"

type EbookUploadRequest struct {
	Title      string `form:"title" binding:"required,min=1,max=255"`
	Author     string `form:"author"`
	TotalPages int    `form:"total_pages" binding:"required,min=1"`
}

type EbookUpdateRequest struct {
	Title  string `json:"title" binding:"required,min=1,max=255"`
	Author string `json:"author"`
}

type EbookResponse struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Author     string    `json:"author"`
	CoverURL   string    `json:"cover_url"`
	FileSize   int64     `json:"file_size"`
	TotalPages int       `json:"total_pages"`
	CreatedAt  time.Time `json:"created_at"`
}

type EbookFileResponse struct {
	URL string `json:"url"`
}
```

- [ ] **Step 3: Create `backend/internal/repository/ebook_repo.go`**

```go
package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/fredy/mbaca-buku/internal/model"
)

type EbookRepository struct {
	db *sql.DB
}

func NewEbookRepository(db *sql.DB) *EbookRepository {
	return &EbookRepository{db: db}
}

func (r *EbookRepository) Create(ctx context.Context, ebook *model.Ebook) error {
	query := `INSERT INTO ebooks (title, author, cover_url, file_url, file_size, total_pages, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, query,
		ebook.Title, ebook.Author, ebook.CoverURL, ebook.FileURL,
		ebook.FileSize, ebook.TotalPages, ebook.UploadedBy,
	).Scan(&ebook.ID, &ebook.CreatedAt, &ebook.UpdatedAt)
}

func (r *EbookRepository) GetByID(ctx context.Context, id string) (*model.Ebook, error) {
	ebook := &model.Ebook{}
	query := `SELECT id, title, author, cover_url, file_url, file_size, total_pages, uploaded_by, created_at, updated_at
		FROM ebooks WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&ebook.ID, &ebook.Title, &ebook.Author, &ebook.CoverURL, &ebook.FileURL,
		&ebook.FileSize, &ebook.TotalPages, &ebook.UploadedBy, &ebook.CreatedAt, &ebook.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("ebook not found")
	}
	return ebook, err
}

func (r *EbookRepository) List(ctx context.Context, page, perPage int) ([]*model.Ebook, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ebooks`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	query := `SELECT id, title, author, cover_url, file_url, file_size, total_pages, uploaded_by, created_at, updated_at
		FROM ebooks ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.QueryContext(ctx, query, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var ebooks []*model.Ebook
	for rows.Next() {
		e := &model.Ebook{}
		if err := rows.Scan(&e.ID, &e.Title, &e.Author, &e.CoverURL, &e.FileURL,
			&e.FileSize, &e.TotalPages, &e.UploadedBy, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, 0, err
		}
		ebooks = append(ebooks, e)
	}

	return ebooks, total, nil
}

func (r *EbookRepository) Update(ctx context.Context, ebook *model.Ebook) error {
	query := `UPDATE ebooks SET title = $1, author = $2, updated_at = NOW() WHERE id = $3`
	result, err := r.db.ExecContext(ctx, query, ebook.Title, ebook.Author, ebook.ID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("ebook not found")
	}
	return nil
}

func (r *EbookRepository) Delete(ctx context.Context, id string) (*model.Ebook, error) {
	ebook, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	_, err = r.db.ExecContext(ctx, `DELETE FROM ebooks WHERE id = $1`, id)
	return ebook, err
}
```

- [ ] **Step 4: Create `backend/internal/service/ebook_service.go`**

```go
package service

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/fredy/mbaca-buku/internal/dto"
	"github.com/fredy/mbaca-buku/internal/model"
	"github.com/fredy/mbaca-buku/internal/repository"
	"github.com/fredy/mbaca-buku/internal/storage"
)

type EbookService struct {
	ebookRepo *repository.EbookRepository
	storage   *storage.MinIOStorage
}

func NewEbookService(ebookRepo *repository.EbookRepository, storage *storage.MinIOStorage) *EbookService {
	return &EbookService{ebookRepo: ebookRepo, storage: storage}
}

func (s *EbookService) Upload(ctx context.Context, req dto.EbookUploadRequest, file io.Reader, fileSize int64, fileName string, userID string) (*model.Ebook, error) {
	objectName := fmt.Sprintf("ebooks/%s/%s", userID, fileName)
	key, err := s.storage.UploadFile(ctx, objectName, file, fileSize, "application/pdf")
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	ebook := &model.Ebook{
		Title:      req.Title,
		Author:     req.Author,
		FileURL:    key,
		FileSize:   fileSize,
		TotalPages: req.TotalPages,
		UploadedBy: userID,
	}

	if err := s.ebookRepo.Create(ctx, ebook); err != nil {
		return nil, fmt.Errorf("failed to save ebook: %w", err)
	}

	return ebook, nil
}

func (s *EbookService) GetByID(ctx context.Context, id string) (*model.Ebook, error) {
	return s.ebookRepo.GetByID(ctx, id)
}

func (s *EbookService) List(ctx context.Context, page, perPage int) ([]*model.Ebook, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 50 {
		perPage = 20
	}
	return s.ebookRepo.List(ctx, page, perPage)
}

func (s *EbookService) Update(ctx context.Context, id string, req dto.EbookUpdateRequest) (*model.Ebook, error) {
	ebook, err := s.ebookRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	ebook.Title = req.Title
	ebook.Author = req.Author

	if err := s.ebookRepo.Update(ctx, ebook); err != nil {
		return nil, err
	}

	return s.ebookRepo.GetByID(ctx, id)
}

func (s *EbookService) Delete(ctx context.Context, id string) error {
	ebook, err := s.ebookRepo.Delete(ctx, id)
	if err != nil {
		return err
	}

	if ebook.FileURL != "" {
		_ = s.storage.DeleteFile(ctx, ebook.FileURL)
	}
	if ebook.CoverURL != "" {
		_ = s.storage.DeleteFile(ctx, ebook.CoverURL)
	}

	return nil
}

func (s *EbookService) GetFileURL(ctx context.Context, id string) (string, error) {
	ebook, err := s.ebookRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	return s.storage.GetPresignedURL(ctx, ebook.FileURL, 1*time.Hour)
}
```

- [ ] **Step 5: Create `backend/internal/handler/ebook_handler.go`**

```go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fredy/mbaca-buku/internal/dto"
	"github.com/fredy/mbaca-buku/internal/service"
	"github.com/fredy/mbaca-buku/pkg/utils"
)

type EbookHandler struct {
	ebookService *service.EbookService
}

func NewEbookHandler(ebookService *service.EbookService) *EbookHandler {
	return &EbookHandler{ebookService: ebookService}
}

func (h *EbookHandler) Upload(c *gin.Context) {
	var req dto.EbookUploadRequest
	if err := c.ShouldBind(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", "file is required")
		return
	}
	defer file.Close()

	userID := c.GetString("user_id")
	fileName := uuid.New().String() + ".pdf"

	ebook, err := h.ebookService.Upload(c.Request.Context(), req, file, header.Size, fileName, userID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "UPLOAD_ERROR", err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, ebook)
}

func (h *EbookHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	ebook, err := h.ebookService.GetByID(c.Request.Context(), id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, ebook)
}

func (h *EbookHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	ebooks, total, err := h.ebookService.List(c.Request.Context(), page, perPage)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	utils.PaginatedResponse(c, ebooks, page, perPage, total)
}

func (h *EbookHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req dto.EbookUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	ebook, err := h.ebookService.Update(c.Request.Context(), id, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, ebook)
}

func (h *EbookHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.ebookService.Delete(c.Request.Context(), id); err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, gin.H{"message": "ebook deleted"})
}

func (h *EbookHandler) GetFileURL(c *gin.Context) {
	id := c.Param("id")
	url, err := h.ebookService.GetFileURL(c.Request.Context(), id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, dto.EbookFileResponse{URL: url})
}
```

- [ ] **Step 6: Update `backend/internal/router/router.go` to add ebook routes**

```go
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
```

- [ ] **Step 7: Update `backend/cmd/server/main.go` to wire ebook service**

Add after the authHandler initialization:

```go
	ebookRepo := repository.NewEbookRepository(db)
	ebookService := service.NewEbookService(ebookRepo, minioStorage)
	ebookHandler := handler.NewEbookHandler(ebookService)
```

Change the MinIO initialization line to capture the return value:

```go
	minioStorage, err := storage.NewMinIOClient(cfg)
```

Update the router config:

```go
	router.Setup(r, &router.RouterConfig{
		AuthHandler:  authHandler,
		EbookHandler: ebookHandler,
		JWTSecret:    cfg.JWTSecret,
	})
```

- [ ] **Step 8: Install uuid dependency**

```bash
cd /Users/fredy/Documents/GitHub/mbaca-buku/backend
go get github.com/google/uuid
```

- [ ] **Step 9: Test ebook CRUD with Docker**

```bash
docker-compose up --build -d

# Login first
TOKEN=$(curl -s -X POST http://localhost/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@mbacabuku.com","password":"12345"}' | jq -r '.data.token')

# Upload (use any small PDF for testing)
curl -X POST http://localhost/api/ebooks \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@test.pdf" \
  -F "title=Test Book" \
  -F "author=Test Author" \
  -F "total_pages=10"

# List
curl http://localhost/api/ebooks -H "Authorization: Bearer $TOKEN"

docker-compose down
```

- [ ] **Step 10: Commit**

```bash
git add .
git commit -m "feat: ebook CRUD with MinIO file storage"
```

---

### Task 6: Reading Progress & History — Redis Cache + DB Sync

**Files:**
- Create: `backend/internal/model/reading_progress.go`
- Create: `backend/internal/model/history.go`
- Create: `backend/internal/dto/progress_dto.go`
- Create: `backend/internal/repository/progress_repo.go`
- Create: `backend/internal/repository/history_repo.go`
- Create: `backend/internal/service/reading_service.go`
- Create: `backend/internal/service/history_service.go`
- Create: `backend/internal/handler/reading_handler.go`
- Create: `backend/internal/handler/history_handler.go`
- Modify: `backend/internal/router/router.go`
- Modify: `backend/cmd/server/main.go`

**Interfaces:**
- Consumes:
  - `cache.NewRedisClient()` → `*redis.Client`
  - `repository.EbookRepository.GetByID()`
  - `middleware.AuthMiddleware()`
- Produces:
  - `model.ReadingProgress` struct
  - `model.History` struct
  - `service.ReadingService.OpenBook(ctx, userID, ebookID) (lastPage int, error)` — logs history, returns cached/DB last page
  - `service.ReadingService.UpdateProgress(ctx, userID, ebookID, page) error` — writes Redis, detects completion
  - `service.ReadingService.GetProgress(ctx, userID, ebookID) (int, string, error)` — returns page + status
  - `service.ReadingService.StartFlusher(ctx)` — background goroutine that syncs Redis → DB every 30s
  - `service.HistoryService.GetHistory(ctx, userID) (reading, completed []HistoryItem, error)`
  - Endpoints: `POST /api/ebooks/:id/open`, `GET/PUT /api/ebooks/:id/progress`, `GET /api/history`

- [ ] **Step 1: Create `backend/internal/model/reading_progress.go`**

```go
package model

import "time"

type ReadingProgress struct {
	ID       string    `json:"id"`
	UserID   string    `json:"user_id"`
	EbookID  string    `json:"ebook_id"`
	LastPage int       `json:"last_page"`
	Status   string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}
```

- [ ] **Step 2: Create `backend/internal/model/history.go`**

```go
package model

import "time"

type History struct {
	ID       string    `json:"id"`
	UserID   string    `json:"user_id"`
	EbookID  string    `json:"ebook_id"`
	OpenedAt time.Time `json:"opened_at"`
}
```

- [ ] **Step 3: Create `backend/internal/dto/progress_dto.go`**

```go
package dto

import "time"

type UpdateProgressRequest struct {
	Page int `json:"page" binding:"required,min=1"`
}

type ProgressResponse struct {
	EbookID  string `json:"ebook_id"`
	LastPage int    `json:"last_page"`
	Status   string `json:"status"`
}

type OpenBookResponse struct {
	EbookID  string `json:"ebook_id"`
	LastPage int    `json:"last_page"`
	FileURL  string `json:"file_url"`
}

type HistoryItem struct {
	EbookID    string    `json:"ebook_id"`
	Title      string    `json:"title"`
	Author     string    `json:"author"`
	CoverURL   string    `json:"cover_url"`
	TotalPages int       `json:"total_pages"`
	LastPage   int       `json:"last_page"`
	Status     string    `json:"status"`
	LastOpened time.Time `json:"last_opened"`
}

type HistoryResponse struct {
	Reading   []HistoryItem `json:"reading"`
	Completed []HistoryItem `json:"completed"`
}
```

- [ ] **Step 4: Create `backend/internal/repository/progress_repo.go`**

```go
package repository

import (
	"context"
	"database/sql"

	"github.com/fredy/mbaca-buku/internal/model"
)

type ProgressRepository struct {
	db *sql.DB
}

func NewProgressRepository(db *sql.DB) *ProgressRepository {
	return &ProgressRepository{db: db}
}

func (r *ProgressRepository) Upsert(ctx context.Context, userID, ebookID string, lastPage int, status string) error {
	query := `INSERT INTO reading_progress (user_id, ebook_id, last_page, status)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, ebook_id)
		DO UPDATE SET last_page = $3, status = $4, updated_at = NOW()`
	_, err := r.db.ExecContext(ctx, query, userID, ebookID, lastPage, status)
	return err
}

func (r *ProgressRepository) GetByUserAndEbook(ctx context.Context, userID, ebookID string) (*model.ReadingProgress, error) {
	p := &model.ReadingProgress{}
	query := `SELECT id, user_id, ebook_id, last_page, status, updated_at
		FROM reading_progress WHERE user_id = $1 AND ebook_id = $2`
	err := r.db.QueryRowContext(ctx, query, userID, ebookID).
		Scan(&p.ID, &p.UserID, &p.EbookID, &p.LastPage, &p.Status, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}
```

- [ ] **Step 5: Create `backend/internal/repository/history_repo.go`**

```go
package repository

import (
	"context"
	"database/sql"

	"github.com/fredy/mbaca-buku/internal/dto"
)

type HistoryRepository struct {
	db *sql.DB
}

func NewHistoryRepository(db *sql.DB) *HistoryRepository {
	return &HistoryRepository{db: db}
}

func (r *HistoryRepository) LogOpen(ctx context.Context, userID, ebookID string) error {
	query := `INSERT INTO history (user_id, ebook_id) VALUES ($1, $2)`
	_, err := r.db.ExecContext(ctx, query, userID, ebookID)
	return err
}

func (r *HistoryRepository) GetUserHistory(ctx context.Context, userID string) ([]dto.HistoryItem, error) {
	query := `SELECT DISTINCT ON (rp.ebook_id)
		rp.ebook_id, e.title, e.author, e.cover_url, e.total_pages, rp.last_page, rp.status, h.opened_at
		FROM reading_progress rp
		JOIN ebooks e ON e.id = rp.ebook_id
		LEFT JOIN history h ON h.user_id = rp.user_id AND h.ebook_id = rp.ebook_id
		WHERE rp.user_id = $1
		ORDER BY rp.ebook_id, h.opened_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []dto.HistoryItem
	for rows.Next() {
		var item dto.HistoryItem
		if err := rows.Scan(&item.EbookID, &item.Title, &item.Author, &item.CoverURL,
			&item.TotalPages, &item.LastPage, &item.Status, &item.LastOpened); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}
```

- [ ] **Step 6: Create `backend/internal/service/reading_service.go`**

```go
package service

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/fredy/mbaca-buku/internal/repository"
)

type ReadingService struct {
	progressRepo *repository.ProgressRepository
	historyRepo  *repository.HistoryRepository
	ebookRepo    *repository.EbookRepository
	rdb          *redis.Client
}

func NewReadingService(
	progressRepo *repository.ProgressRepository,
	historyRepo *repository.HistoryRepository,
	ebookRepo *repository.EbookRepository,
	rdb *redis.Client,
) *ReadingService {
	return &ReadingService{
		progressRepo: progressRepo,
		historyRepo:  historyRepo,
		ebookRepo:    ebookRepo,
		rdb:          rdb,
	}
}

func (s *ReadingService) redisKey(userID, ebookID string) string {
	return fmt.Sprintf("user:%s:book:%s:last_page", userID, ebookID)
}

func (s *ReadingService) OpenBook(ctx context.Context, userID, ebookID string) (int, error) {
	_, err := s.ebookRepo.GetByID(ctx, ebookID)
	if err != nil {
		return 0, fmt.Errorf("ebook not found")
	}

	_ = s.historyRepo.LogOpen(ctx, userID, ebookID)

	lastPage := 1
	cached, err := s.rdb.Get(ctx, s.redisKey(userID, ebookID)).Result()
	if err == nil {
		if p, err := strconv.Atoi(cached); err == nil {
			lastPage = p
		}
	} else {
		progress, err := s.progressRepo.GetByUserAndEbook(ctx, userID, ebookID)
		if err == nil && progress != nil {
			lastPage = progress.LastPage
			s.rdb.Set(ctx, s.redisKey(userID, ebookID), lastPage, 24*time.Hour)
		} else {
			_ = s.progressRepo.Upsert(ctx, userID, ebookID, 1, "reading")
		}
	}

	return lastPage, nil
}

func (s *ReadingService) GetProgress(ctx context.Context, userID, ebookID string) (int, string, error) {
	cached, err := s.rdb.Get(ctx, s.redisKey(userID, ebookID)).Result()
	if err == nil {
		if p, err := strconv.Atoi(cached); err == nil {
			progress, _ := s.progressRepo.GetByUserAndEbook(ctx, userID, ebookID)
			status := "reading"
			if progress != nil {
				status = progress.Status
			}
			return p, status, nil
		}
	}

	progress, err := s.progressRepo.GetByUserAndEbook(ctx, userID, ebookID)
	if err != nil || progress == nil {
		return 1, "reading", nil
	}

	s.rdb.Set(ctx, s.redisKey(userID, ebookID), progress.LastPage, 24*time.Hour)
	return progress.LastPage, progress.Status, nil
}

func (s *ReadingService) UpdateProgress(ctx context.Context, userID, ebookID string, page int) error {
	ebook, err := s.ebookRepo.GetByID(ctx, ebookID)
	if err != nil {
		return fmt.Errorf("ebook not found")
	}

	s.rdb.Set(ctx, s.redisKey(userID, ebookID), page, 24*time.Hour)

	status := "reading"
	if page >= ebook.TotalPages {
		status = "completed"
		return s.progressRepo.Upsert(ctx, userID, ebookID, page, status)
	}

	return nil
}

func (s *ReadingService) StartFlusher(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				s.flushToDB(ctx)
			}
		}
	}()
	log.Println("Reading progress flusher started (30s interval)")
}

func (s *ReadingService) flushToDB(ctx context.Context) {
	keys, err := s.rdb.Keys(ctx, "user:*:book:*:last_page").Result()
	if err != nil {
		log.Printf("Flusher: failed to scan keys: %v", err)
		return
	}

	for _, key := range keys {
		val, err := s.rdb.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		page, err := strconv.Atoi(val)
		if err != nil {
			continue
		}

		var userID, ebookID string
		_, err = fmt.Sscanf(key, "user:%s", &userID)
		if err != nil {
			continue
		}

		parts := splitRedisKey(key)
		if parts == nil {
			continue
		}
		userID = parts[0]
		ebookID = parts[1]

		progress, _ := s.progressRepo.GetByUserAndEbook(ctx, userID, ebookID)
		status := "reading"
		if progress != nil {
			status = progress.Status
		}

		_ = s.progressRepo.Upsert(ctx, userID, ebookID, page, status)
	}
}

func splitRedisKey(key string) []string {
	// key format: user:{userId}:book:{bookId}:last_page
	var userID, bookID string
	n, _ := fmt.Sscanf(key, "user:%s", &userID)
	if n != 1 {
		return nil
	}

	// More robust parsing
	const prefix = "user:"
	const bookSep = ":book:"
	const suffix = ":last_page"

	start := len(prefix)
	bookIdx := indexOf(key, bookSep)
	if bookIdx < 0 {
		return nil
	}
	userID = key[start:bookIdx]

	bookStart := bookIdx + len(bookSep)
	suffixIdx := indexOf(key, suffix)
	if suffixIdx < 0 {
		return nil
	}
	bookID = key[bookStart:suffixIdx]

	return []string{userID, bookID}
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
```

- [ ] **Step 7: Create `backend/internal/service/history_service.go`**

```go
package service

import (
	"context"

	"github.com/fredy/mbaca-buku/internal/dto"
	"github.com/fredy/mbaca-buku/internal/repository"
)

type HistoryService struct {
	historyRepo *repository.HistoryRepository
}

func NewHistoryService(historyRepo *repository.HistoryRepository) *HistoryService {
	return &HistoryService{historyRepo: historyRepo}
}

func (s *HistoryService) GetHistory(ctx context.Context, userID string) (*dto.HistoryResponse, error) {
	items, err := s.historyRepo.GetUserHistory(ctx, userID)
	if err != nil {
		return nil, err
	}

	resp := &dto.HistoryResponse{
		Reading:   []dto.HistoryItem{},
		Completed: []dto.HistoryItem{},
	}

	for _, item := range items {
		if item.Status == "completed" {
			resp.Completed = append(resp.Completed, item)
		} else {
			resp.Reading = append(resp.Reading, item)
		}
	}

	return resp, nil
}
```

- [ ] **Step 8: Create `backend/internal/handler/reading_handler.go`**

```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fredy/mbaca-buku/internal/dto"
	"github.com/fredy/mbaca-buku/internal/service"
	"github.com/fredy/mbaca-buku/pkg/utils"
)

type ReadingHandler struct {
	readingService *service.ReadingService
}

func NewReadingHandler(readingService *service.ReadingService) *ReadingHandler {
	return &ReadingHandler{readingService: readingService}
}

func (h *ReadingHandler) OpenBook(c *gin.Context) {
	userID := c.GetString("user_id")
	ebookID := c.Param("id")

	lastPage, err := h.readingService.OpenBook(c.Request.Context(), userID, ebookID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, dto.OpenBookResponse{
		EbookID:  ebookID,
		LastPage: lastPage,
	})
}

func (h *ReadingHandler) GetProgress(c *gin.Context) {
	userID := c.GetString("user_id")
	ebookID := c.Param("id")

	page, status, err := h.readingService.GetProgress(c.Request.Context(), userID, ebookID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, dto.ProgressResponse{
		EbookID:  ebookID,
		LastPage: page,
		Status:   status,
	})
}

func (h *ReadingHandler) UpdateProgress(c *gin.Context) {
	userID := c.GetString("user_id")
	ebookID := c.Param("id")

	var req dto.UpdateProgressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	if err := h.readingService.UpdateProgress(c.Request.Context(), userID, ebookID, req.Page); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, gin.H{"message": "progress updated"})
}
```

- [ ] **Step 9: Create `backend/internal/handler/history_handler.go`**

```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fredy/mbaca-buku/internal/service"
	"github.com/fredy/mbaca-buku/pkg/utils"
)

type HistoryHandler struct {
	historyService *service.HistoryService
}

func NewHistoryHandler(historyService *service.HistoryService) *HistoryHandler {
	return &HistoryHandler{historyService: historyService}
}

func (h *HistoryHandler) GetHistory(c *gin.Context) {
	userID := c.GetString("user_id")

	history, err := h.historyService.GetHistory(c.Request.Context(), userID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, history)
}
```

- [ ] **Step 10: Update `backend/internal/router/router.go` to add reading + history routes**

```go
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/fredy/mbaca-buku/internal/handler"
	"github.com/fredy/mbaca-buku/internal/middleware"
)

type RouterConfig struct {
	AuthHandler    *handler.AuthHandler
	EbookHandler   *handler.EbookHandler
	ReadingHandler *handler.ReadingHandler
	HistoryHandler *handler.HistoryHandler
	JWTSecret      string
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

			ebooks.POST("/:id/open", cfg.ReadingHandler.OpenBook)
			ebooks.GET("/:id/progress", cfg.ReadingHandler.GetProgress)
			ebooks.PUT("/:id/progress", cfg.ReadingHandler.UpdateProgress)

			ebooks.GET("/:id/bookmarks", cfg.BookmarkHandler.List)
			ebooks.POST("/:id/bookmarks", cfg.BookmarkHandler.Create)
		}

		protected.GET("/history", cfg.HistoryHandler.GetHistory)
		protected.DELETE("/bookmarks/:id", cfg.BookmarkHandler.Delete)
	}
}
```

Note: this includes bookmark routes from Task 7 — we'll add the BookmarkHandler field then. For now, temporarily remove the bookmark lines until Task 7 is done, or add all at once in Task 7.

**Temporary version without bookmarks (use this now):**

```go
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/fredy/mbaca-buku/internal/handler"
	"github.com/fredy/mbaca-buku/internal/middleware"
)

type RouterConfig struct {
	AuthHandler    *handler.AuthHandler
	EbookHandler   *handler.EbookHandler
	ReadingHandler *handler.ReadingHandler
	HistoryHandler *handler.HistoryHandler
	JWTSecret      string
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

			ebooks.POST("/:id/open", cfg.ReadingHandler.OpenBook)
			ebooks.GET("/:id/progress", cfg.ReadingHandler.GetProgress)
			ebooks.PUT("/:id/progress", cfg.ReadingHandler.UpdateProgress)
		}

		protected.GET("/history", cfg.HistoryHandler.GetHistory)
	}
}
```

- [ ] **Step 11: Update `backend/cmd/server/main.go` to wire reading + history**

Add after ebook wiring:

```go
	progressRepo := repository.NewProgressRepository(db)
	historyRepo := repository.NewHistoryRepository(db)
	readingService := service.NewReadingService(progressRepo, historyRepo, ebookRepo, rdb)
	historyService := service.NewHistoryService(historyRepo)
	readingHandler := handler.NewReadingHandler(readingService)
	historyHandler := handler.NewHistoryHandler(historyService)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	readingService.StartFlusher(ctx)
```

Update router config:

```go
	router.Setup(r, &router.RouterConfig{
		AuthHandler:    authHandler,
		EbookHandler:   ebookHandler,
		ReadingHandler: readingHandler,
		HistoryHandler: historyHandler,
		JWTSecret:      cfg.JWTSecret,
	})
```

- [ ] **Step 12: Test reading progress flow with Docker**

```bash
docker-compose up --build -d

# Login
TOKEN=$(curl -s -X POST http://localhost/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@mbacabuku.com","password":"12345"}' | jq -r '.data.token')

# Assume an ebook with ID exists from Task 5 testing
# Open book
curl -X POST http://localhost/api/ebooks/{ebook_id}/open \
  -H "Authorization: Bearer $TOKEN"

# Update progress
curl -X PUT http://localhost/api/ebooks/{ebook_id}/progress \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"page": 5}'

# Get progress
curl http://localhost/api/ebooks/{ebook_id}/progress \
  -H "Authorization: Bearer $TOKEN"

# Get history
curl http://localhost/api/history \
  -H "Authorization: Bearer $TOKEN"

docker-compose down
```

- [ ] **Step 13: Commit**

```bash
git add .
git commit -m "feat: reading progress with Redis cache, history tracking, completion detection"
```

---

### Task 7: Bookmarks — Model, Repository, Service, Handler

**Files:**
- Create: `backend/internal/model/bookmark.go`
- Create: `backend/internal/dto/bookmark_dto.go`
- Create: `backend/internal/repository/bookmark_repo.go`
- Create: `backend/internal/service/bookmark_service.go`
- Create: `backend/internal/handler/bookmark_handler.go`
- Modify: `backend/internal/router/router.go`
- Modify: `backend/cmd/server/main.go`

**Interfaces:**
- Consumes: `middleware.AuthMiddleware()`, `utils.SuccessResponse()`, `utils.ErrorResponse()`
- Produces:
  - `model.Bookmark` struct
  - `service.BookmarkService.Create(ctx, userID, ebookID, page, note) error`
  - `service.BookmarkService.List(ctx, userID, ebookID) ([]model.Bookmark, error)`
  - `service.BookmarkService.Delete(ctx, id, userID) error`
  - Endpoints: `GET/POST /api/ebooks/:id/bookmarks`, `DELETE /api/bookmarks/:id`

- [ ] **Step 1: Create `backend/internal/model/bookmark.go`**

```go
package model

import "time"

type Bookmark struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	EbookID    string    `json:"ebook_id"`
	PageNumber int       `json:"page_number"`
	Note       string    `json:"note"`
	CreatedAt  time.Time `json:"created_at"`
}
```

- [ ] **Step 2: Create `backend/internal/dto/bookmark_dto.go`**

```go
package dto

type CreateBookmarkRequest struct {
	PageNumber int    `json:"page_number" binding:"required,min=1"`
	Note       string `json:"note"`
}
```

- [ ] **Step 3: Create `backend/internal/repository/bookmark_repo.go`**

```go
package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/fredy/mbaca-buku/internal/model"
)

type BookmarkRepository struct {
	db *sql.DB
}

func NewBookmarkRepository(db *sql.DB) *BookmarkRepository {
	return &BookmarkRepository{db: db}
}

func (r *BookmarkRepository) Create(ctx context.Context, bookmark *model.Bookmark) error {
	query := `INSERT INTO bookmarks (user_id, ebook_id, page_number, note)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`
	return r.db.QueryRowContext(ctx, query,
		bookmark.UserID, bookmark.EbookID, bookmark.PageNumber, bookmark.Note,
	).Scan(&bookmark.ID, &bookmark.CreatedAt)
}

func (r *BookmarkRepository) ListByUserAndEbook(ctx context.Context, userID, ebookID string) ([]model.Bookmark, error) {
	query := `SELECT id, user_id, ebook_id, page_number, note, created_at
		FROM bookmarks WHERE user_id = $1 AND ebook_id = $2
		ORDER BY page_number ASC`

	rows, err := r.db.QueryContext(ctx, query, userID, ebookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookmarks []model.Bookmark
	for rows.Next() {
		var b model.Bookmark
		if err := rows.Scan(&b.ID, &b.UserID, &b.EbookID, &b.PageNumber, &b.Note, &b.CreatedAt); err != nil {
			return nil, err
		}
		bookmarks = append(bookmarks, b)
	}

	if bookmarks == nil {
		bookmarks = []model.Bookmark{}
	}
	return bookmarks, nil
}

func (r *BookmarkRepository) Delete(ctx context.Context, id, userID string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM bookmarks WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("bookmark not found")
	}
	return nil
}
```

- [ ] **Step 4: Create `backend/internal/service/bookmark_service.go`**

```go
package service

import (
	"context"

	"github.com/fredy/mbaca-buku/internal/dto"
	"github.com/fredy/mbaca-buku/internal/model"
	"github.com/fredy/mbaca-buku/internal/repository"
)

type BookmarkService struct {
	bookmarkRepo *repository.BookmarkRepository
}

func NewBookmarkService(bookmarkRepo *repository.BookmarkRepository) *BookmarkService {
	return &BookmarkService{bookmarkRepo: bookmarkRepo}
}

func (s *BookmarkService) Create(ctx context.Context, userID, ebookID string, req dto.CreateBookmarkRequest) (*model.Bookmark, error) {
	bookmark := &model.Bookmark{
		UserID:     userID,
		EbookID:    ebookID,
		PageNumber: req.PageNumber,
		Note:       req.Note,
	}

	if err := s.bookmarkRepo.Create(ctx, bookmark); err != nil {
		return nil, err
	}
	return bookmark, nil
}

func (s *BookmarkService) List(ctx context.Context, userID, ebookID string) ([]model.Bookmark, error) {
	return s.bookmarkRepo.ListByUserAndEbook(ctx, userID, ebookID)
}

func (s *BookmarkService) Delete(ctx context.Context, id, userID string) error {
	return s.bookmarkRepo.Delete(ctx, id, userID)
}
```

- [ ] **Step 5: Create `backend/internal/handler/bookmark_handler.go`**

```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fredy/mbaca-buku/internal/dto"
	"github.com/fredy/mbaca-buku/internal/service"
	"github.com/fredy/mbaca-buku/pkg/utils"
)

type BookmarkHandler struct {
	bookmarkService *service.BookmarkService
}

func NewBookmarkHandler(bookmarkService *service.BookmarkService) *BookmarkHandler {
	return &BookmarkHandler{bookmarkService: bookmarkService}
}

func (h *BookmarkHandler) Create(c *gin.Context) {
	userID := c.GetString("user_id")
	ebookID := c.Param("id")

	var req dto.CreateBookmarkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	bookmark, err := h.bookmarkService.Create(c.Request.Context(), userID, ebookID, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusConflict, "BOOKMARK_ERROR", err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, bookmark)
}

func (h *BookmarkHandler) List(c *gin.Context) {
	userID := c.GetString("user_id")
	ebookID := c.Param("id")

	bookmarks, err := h.bookmarkService.List(c.Request.Context(), userID, ebookID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, bookmarks)
}

func (h *BookmarkHandler) Delete(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")

	if err := h.bookmarkService.Delete(c.Request.Context(), id, userID); err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, gin.H{"message": "bookmark deleted"})
}
```

- [ ] **Step 6: Update `backend/internal/router/router.go` — add BookmarkHandler field and routes**

Add `BookmarkHandler *handler.BookmarkHandler` to `RouterConfig` struct, and add bookmark routes inside the ebooks group:

```go
		ebooks.GET("/:id/bookmarks", cfg.BookmarkHandler.List)
		ebooks.POST("/:id/bookmarks", cfg.BookmarkHandler.Create)
```

And outside ebooks group but inside protected:

```go
		protected.DELETE("/bookmarks/:id", cfg.BookmarkHandler.Delete)
```

- [ ] **Step 7: Update `backend/cmd/server/main.go` — wire bookmark service**

Add after reading/history wiring:

```go
	bookmarkRepo := repository.NewBookmarkRepository(db)
	bookmarkService := service.NewBookmarkService(bookmarkRepo)
	bookmarkHandler := handler.NewBookmarkHandler(bookmarkService)
```

Add `BookmarkHandler: bookmarkHandler` to the router config.

- [ ] **Step 8: Verify compilation and test**

```bash
cd /Users/fredy/Documents/GitHub/mbaca-buku/backend
go build ./...
docker-compose up --build -d

# Test bookmark creation
curl -X POST http://localhost/api/ebooks/{ebook_id}/bookmarks \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"page_number": 5, "note": "Important section"}'

docker-compose down
```

- [ ] **Step 9: Commit**

```bash
git add .
git commit -m "feat: manual bookmarks CRUD"
```

---

### Task 8: Frontend — Auth (Login/Register), API Client, Routing

**Files:**
- Create: `frontend/src/api/client.ts`
- Create: `frontend/src/api/auth.ts`
- Create: `frontend/src/context/AuthContext.tsx`
- Create: `frontend/src/pages/LoginPage.tsx`
- Create: `frontend/src/pages/RegisterPage.tsx`
- Create: `frontend/src/components/layout/Layout.tsx`
- Create: `frontend/src/components/layout/Navbar.tsx`
- Create: `frontend/src/router/index.tsx`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/main.tsx`
- Modify: `frontend/src/index.css`

**Interfaces:**
- Consumes: Backend auth endpoints (`POST /api/auth/login`, `POST /api/auth/register`, `GET /api/auth/me`)
- Produces:
  - `api.client` — axios instance with JWT interceptor
  - `api.auth.login(email, password)` → `{user, token}`
  - `api.auth.register(name, email, password)` → `{user, token}`
  - `AuthContext` providing `{user, token, login, register, logout, isLoading}`
  - Protected route wrapper that redirects to `/login` when unauthenticated
  - LoginPage and RegisterPage components

- [ ] **Step 1: Create `frontend/src/api/client.ts`**

```typescript
import axios from 'axios';

const client = axios.create({
  baseURL: '/api',
});

client.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

client.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

export default client;
```

- [ ] **Step 2: Create `frontend/src/api/auth.ts`**

```typescript
import client from './client';

export interface User {
  id: string;
  name: string;
  email: string;
}

export interface AuthResponse {
  user: User;
  token: string;
}

export const authApi = {
  login: async (email: string, password: string) => {
    const res = await client.post<{ success: boolean; data: AuthResponse }>('/auth/login', { email, password });
    return res.data.data;
  },

  register: async (name: string, email: string, password: string) => {
    const res = await client.post<{ success: boolean; data: AuthResponse }>('/auth/register', { name, email, password });
    return res.data.data;
  },

  me: async () => {
    const res = await client.get<{ success: boolean; data: User }>('/auth/me');
    return res.data.data;
  },
};
```

- [ ] **Step 3: Create `frontend/src/context/AuthContext.tsx`**

```tsx
import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { authApi, User } from '../api/auth';

interface AuthContextType {
  user: User | null;
  token: string | null;
  login: (email: string, password: string) => Promise<void>;
  register: (name: string, email: string, password: string) => Promise<void>;
  logout: () => void;
  isLoading: boolean;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(localStorage.getItem('token'));
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    if (token) {
      authApi.me()
        .then(setUser)
        .catch(() => {
          localStorage.removeItem('token');
          setToken(null);
        })
        .finally(() => setIsLoading(false));
    } else {
      setIsLoading(false);
    }
  }, [token]);

  const login = async (email: string, password: string) => {
    const data = await authApi.login(email, password);
    localStorage.setItem('token', data.token);
    setToken(data.token);
    setUser(data.user);
  };

  const register = async (name: string, email: string, password: string) => {
    const data = await authApi.register(name, email, password);
    localStorage.setItem('token', data.token);
    setToken(data.token);
    setUser(data.user);
  };

  const logout = () => {
    localStorage.removeItem('token');
    setToken(null);
    setUser(null);
  };

  return (
    <AuthContext.Provider value={{ user, token, login, register, logout, isLoading }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
```

- [ ] **Step 4: Create `frontend/src/components/layout/Navbar.tsx`**

```tsx
import { Link } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';

export default function Navbar() {
  const { user, logout } = useAuth();

  return (
    <nav className="bg-white border-b border-gray-200">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between h-16 items-center">
          <div className="flex items-center gap-8">
            <Link to="/" className="text-xl font-bold text-gray-900">
              Mbaca Buku
            </Link>
            <div className="hidden sm:flex gap-6">
              <Link to="/" className="text-gray-600 hover:text-gray-900">Dashboard</Link>
              <Link to="/ebooks" className="text-gray-600 hover:text-gray-900">Ebooks</Link>
              <Link to="/history" className="text-gray-600 hover:text-gray-900">History</Link>
            </div>
          </div>
          <div className="flex items-center gap-4">
            <span className="text-sm text-gray-600">{user?.name}</span>
            <button
              onClick={logout}
              className="text-sm text-red-600 hover:text-red-800"
            >
              Logout
            </button>
          </div>
        </div>
      </div>
    </nav>
  );
}
```

- [ ] **Step 5: Create `frontend/src/components/layout/Layout.tsx`**

```tsx
import { Outlet } from 'react-router-dom';
import Navbar from './Navbar';

export default function Layout() {
  return (
    <div className="min-h-screen bg-gray-50">
      <Navbar />
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <Outlet />
      </main>
    </div>
  );
}
```

- [ ] **Step 6: Create `frontend/src/pages/LoginPage.tsx`**

```tsx
import { useState, FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

export default function LoginPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await login(email, password);
      navigate('/');
    } catch {
      setError('Invalid email or password');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="w-full max-w-md bg-white rounded-lg shadow p-8">
        <h1 className="text-2xl font-bold text-center mb-6">Mbaca Buku</h1>
        <p className="text-center text-gray-500 mb-8">Sign in to your account</p>

        {error && (
          <div className="bg-red-50 text-red-600 p-3 rounded mb-4 text-sm">{error}</div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Email</label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Password</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              required
            />
          </div>
          <button
            type="submit"
            disabled={loading}
            className="w-full py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
          >
            {loading ? 'Signing in...' : 'Sign In'}
          </button>
        </form>

        <p className="mt-6 text-center text-sm text-gray-600">
          Don't have an account?{' '}
          <Link to="/register" className="text-blue-600 hover:underline">Register</Link>
        </p>
      </div>
    </div>
  );
}
```

- [ ] **Step 7: Create `frontend/src/pages/RegisterPage.tsx`**

```tsx
import { useState, FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

export default function RegisterPage() {
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const { register } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await register(name, email, password);
      navigate('/');
    } catch {
      setError('Registration failed. Email might already be in use.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="w-full max-w-md bg-white rounded-lg shadow p-8">
        <h1 className="text-2xl font-bold text-center mb-6">Mbaca Buku</h1>
        <p className="text-center text-gray-500 mb-8">Create your account</p>

        {error && (
          <div className="bg-red-50 text-red-600 p-3 rounded mb-4 text-sm">{error}</div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Email</label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Password</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              minLength={5}
              required
            />
          </div>
          <button
            type="submit"
            disabled={loading}
            className="w-full py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
          >
            {loading ? 'Creating account...' : 'Register'}
          </button>
        </form>

        <p className="mt-6 text-center text-sm text-gray-600">
          Already have an account?{' '}
          <Link to="/login" className="text-blue-600 hover:underline">Sign in</Link>
        </p>
      </div>
    </div>
  );
}
```

- [ ] **Step 8: Create `frontend/src/router/index.tsx`**

```tsx
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import Layout from '../components/layout/Layout';
import LoginPage from '../pages/LoginPage';
import RegisterPage from '../pages/RegisterPage';

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { user, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-gray-500">Loading...</div>
      </div>
    );
  }

  if (!user) return <Navigate to="/login" />;
  return <>{children}</>;
}

function PlaceholderPage({ title }: { title: string }) {
  return <h1 className="text-2xl font-bold">{title}</h1>;
}

export default function AppRouter() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route
          element={
            <ProtectedRoute>
              <Layout />
            </ProtectedRoute>
          }
        >
          <Route path="/" element={<PlaceholderPage title="Dashboard" />} />
          <Route path="/ebooks" element={<PlaceholderPage title="Ebooks" />} />
          <Route path="/read/:id" element={<PlaceholderPage title="Reader" />} />
          <Route path="/history" element={<PlaceholderPage title="History" />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
```

- [ ] **Step 9: Update `frontend/src/App.tsx`**

```tsx
import { AuthProvider } from './context/AuthContext';
import AppRouter from './router';

export default function App() {
  return (
    <AuthProvider>
      <AppRouter />
    </AuthProvider>
  );
}
```

- [ ] **Step 10: Update `frontend/src/main.tsx`**

```tsx
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import App from './App';
import './index.css';

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>
);
```

- [ ] **Step 11: Verify frontend builds and login works**

```bash
cd /Users/fredy/Documents/GitHub/mbaca-buku
docker-compose up --build -d
```

Open `http://localhost` in browser. Expected: redirects to `/login`. Login with `admin@mbacabuku.com` / `12345`. Expected: redirects to `/` showing "Dashboard" placeholder.

- [ ] **Step 12: Commit**

```bash
git add .
git commit -m "feat: frontend auth with login, register, protected routes, and layout"
```

---

### Task 9: Frontend — Dashboard & Ebook List Pages

**Files:**
- Create: `frontend/src/api/ebooks.ts`
- Create: `frontend/src/api/history.ts`
- Create: `frontend/src/components/ebook/EbookCard.tsx`
- Create: `frontend/src/components/ebook/EbookUpload.tsx`
- Create: `frontend/src/components/common/Modal.tsx`
- Create: `frontend/src/components/common/EmptyState.tsx`
- Create: `frontend/src/pages/DashboardPage.tsx`
- Create: `frontend/src/pages/EbookListPage.tsx`
- Modify: `frontend/src/router/index.tsx`

**Interfaces:**
- Consumes:
  - `api.client` axios instance
  - Backend endpoints: `GET /api/ebooks`, `POST /api/ebooks`, `DELETE /api/ebooks/:id`, `GET /api/history`
- Produces:
  - `ebooksApi.list(page, perPage)` → `{data: Ebook[], meta}`
  - `ebooksApi.upload(formData)` → `Ebook`
  - `ebooksApi.delete(id)` → void
  - `historyApi.getHistory()` → `{reading: HistoryItem[], completed: HistoryItem[]}`
  - DashboardPage showing "Continue Reading" + "Recently Added"
  - EbookListPage with grid of cards + upload button
  - EbookUpload modal component

- [ ] **Step 1: Create `frontend/src/api/ebooks.ts`**

```typescript
import client from './client';

export interface Ebook {
  id: string;
  title: string;
  author: string;
  cover_url: string;
  file_size: number;
  total_pages: number;
  created_at: string;
}

export interface EbookListResponse {
  success: boolean;
  data: Ebook[];
  meta: { page: number; per_page: number; total: number };
}

export const ebooksApi = {
  list: async (page = 1, perPage = 20) => {
    const res = await client.get<EbookListResponse>(`/ebooks?page=${page}&per_page=${perPage}`);
    return { ebooks: res.data.data || [], meta: res.data.meta };
  },

  getById: async (id: string) => {
    const res = await client.get<{ success: boolean; data: Ebook }>(`/ebooks/${id}`);
    return res.data.data;
  },

  upload: async (formData: FormData) => {
    const res = await client.post<{ success: boolean; data: Ebook }>('/ebooks', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    return res.data.data;
  },

  update: async (id: string, data: { title: string; author: string }) => {
    const res = await client.put<{ success: boolean; data: Ebook }>(`/ebooks/${id}`, data);
    return res.data.data;
  },

  delete: async (id: string) => {
    await client.delete(`/ebooks/${id}`);
  },

  getFileUrl: async (id: string) => {
    const res = await client.get<{ success: boolean; data: { url: string } }>(`/ebooks/${id}/file`);
    return res.data.data.url;
  },
};
```

- [ ] **Step 2: Create `frontend/src/api/history.ts`**

```typescript
import client from './client';

export interface HistoryItem {
  ebook_id: string;
  title: string;
  author: string;
  cover_url: string;
  total_pages: number;
  last_page: number;
  status: string;
  last_opened: string;
}

export interface HistoryResponse {
  reading: HistoryItem[];
  completed: HistoryItem[];
}

export const historyApi = {
  getHistory: async () => {
    const res = await client.get<{ success: boolean; data: HistoryResponse }>('/history');
    return res.data.data;
  },

  openBook: async (ebookId: string) => {
    const res = await client.post<{ success: boolean; data: { ebook_id: string; last_page: number } }>(
      `/ebooks/${ebookId}/open`
    );
    return res.data.data;
  },
};
```

- [ ] **Step 3: Create `frontend/src/components/common/Modal.tsx`**

```tsx
import { ReactNode } from 'react';

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
}

export default function Modal({ isOpen, onClose, title, children }: ModalProps) {
  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <div className="relative bg-white rounded-lg shadow-xl w-full max-w-md mx-4 p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold">{title}</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600 text-xl">
            &times;
          </button>
        </div>
        {children}
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Create `frontend/src/components/common/EmptyState.tsx`**

```tsx
interface EmptyStateProps {
  title: string;
  description: string;
}

export default function EmptyState({ title, description }: EmptyStateProps) {
  return (
    <div className="text-center py-12">
      <h3 className="text-lg font-medium text-gray-900">{title}</h3>
      <p className="mt-1 text-sm text-gray-500">{description}</p>
    </div>
  );
}
```

- [ ] **Step 5: Create `frontend/src/components/ebook/EbookCard.tsx`**

```tsx
import { Ebook } from '../../api/ebooks';

interface EbookCardProps {
  ebook: Ebook;
  onRead: (id: string) => void;
  onDelete?: (id: string) => void;
  progress?: { last_page: number; total_pages: number };
}

export default function EbookCard({ ebook, onRead, onDelete, progress }: EbookCardProps) {
  const progressPercent = progress
    ? Math.round((progress.last_page / progress.total_pages) * 100)
    : 0;

  return (
    <div className="bg-white rounded-lg shadow hover:shadow-md transition-shadow overflow-hidden">
      <div className="h-48 bg-gradient-to-br from-blue-500 to-blue-700 flex items-center justify-center">
        <span className="text-white text-4xl font-bold opacity-30">PDF</span>
      </div>
      <div className="p-4">
        <h3 className="font-semibold text-gray-900 truncate">{ebook.title}</h3>
        <p className="text-sm text-gray-500 mt-1">{ebook.author || 'Unknown author'}</p>
        <p className="text-xs text-gray-400 mt-1">{ebook.total_pages} pages</p>

        {progress && (
          <div className="mt-3">
            <div className="flex justify-between text-xs text-gray-500 mb-1">
              <span>Progress</span>
              <span>{progressPercent}%</span>
            </div>
            <div className="w-full bg-gray-200 rounded-full h-1.5">
              <div
                className="bg-blue-600 h-1.5 rounded-full"
                style={{ width: `${progressPercent}%` }}
              />
            </div>
          </div>
        )}

        <div className="mt-4 flex gap-2">
          <button
            onClick={() => onRead(ebook.id)}
            className="flex-1 py-1.5 bg-blue-600 text-white text-sm rounded hover:bg-blue-700"
          >
            Read
          </button>
          {onDelete && (
            <button
              onClick={() => onDelete(ebook.id)}
              className="py-1.5 px-3 text-red-600 text-sm border border-red-200 rounded hover:bg-red-50"
            >
              Delete
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 6: Create `frontend/src/components/ebook/EbookUpload.tsx`**

```tsx
import { useState, FormEvent } from 'react';
import Modal from '../common/Modal';
import { ebooksApi } from '../../api/ebooks';

interface EbookUploadProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export default function EbookUpload({ isOpen, onClose, onSuccess }: EbookUploadProps) {
  const [title, setTitle] = useState('');
  const [author, setAuthor] = useState('');
  const [totalPages, setTotalPages] = useState('');
  const [file, setFile] = useState<File | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!file) return;

    setLoading(true);
    setError('');

    const formData = new FormData();
    formData.append('file', file);
    formData.append('title', title);
    formData.append('author', author);
    formData.append('total_pages', totalPages);

    try {
      await ebooksApi.upload(formData);
      setTitle('');
      setAuthor('');
      setTotalPages('');
      setFile(null);
      onSuccess();
      onClose();
    } catch {
      setError('Failed to upload ebook');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Upload Ebook">
      {error && <div className="bg-red-50 text-red-600 p-3 rounded mb-4 text-sm">{error}</div>}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Title</label>
          <input
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Author</label>
          <input
            type="text"
            value={author}
            onChange={(e) => setAuthor(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Total Pages</label>
          <input
            type="number"
            value={totalPages}
            onChange={(e) => setTotalPages(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            min="1"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">PDF File</label>
          <input
            type="file"
            accept=".pdf"
            onChange={(e) => setFile(e.target.files?.[0] || null)}
            className="w-full text-sm text-gray-500 file:mr-4 file:py-2 file:px-4 file:rounded file:border-0 file:text-sm file:bg-blue-50 file:text-blue-700 hover:file:bg-blue-100"
            required
          />
        </div>
        <button
          type="submit"
          disabled={loading}
          className="w-full py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
        >
          {loading ? 'Uploading...' : 'Upload'}
        </button>
      </form>
    </Modal>
  );
}
```

- [ ] **Step 7: Create `frontend/src/pages/DashboardPage.tsx`**

```tsx
import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ebooksApi, Ebook } from '../api/ebooks';
import { historyApi, HistoryItem } from '../api/history';
import EbookCard from '../components/ebook/EbookCard';
import EmptyState from '../components/common/EmptyState';

export default function DashboardPage() {
  const [recentEbooks, setRecentEbooks] = useState<Ebook[]>([]);
  const [reading, setReading] = useState<HistoryItem[]>([]);
  const navigate = useNavigate();

  useEffect(() => {
    ebooksApi.list(1, 4).then(({ ebooks }) => setRecentEbooks(ebooks));
    historyApi.getHistory().then((data) => setReading(data.reading || []));
  }, []);

  const handleRead = async (id: string) => {
    await historyApi.openBook(id);
    navigate(`/read/${id}`);
  };

  return (
    <div className="space-y-10">
      <section>
        <h2 className="text-xl font-bold text-gray-900 mb-4">Continue Reading</h2>
        {reading.length === 0 ? (
          <EmptyState title="No books in progress" description="Start reading a book to see it here" />
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
            {reading.map((item) => (
              <EbookCard
                key={item.ebook_id}
                ebook={{
                  id: item.ebook_id,
                  title: item.title,
                  author: item.author,
                  cover_url: item.cover_url,
                  total_pages: item.total_pages,
                  file_size: 0,
                  created_at: item.last_opened,
                }}
                onRead={handleRead}
                progress={{ last_page: item.last_page, total_pages: item.total_pages }}
              />
            ))}
          </div>
        )}
      </section>

      <section>
        <h2 className="text-xl font-bold text-gray-900 mb-4">Recently Added</h2>
        {recentEbooks.length === 0 ? (
          <EmptyState title="No ebooks yet" description="Upload your first ebook to get started" />
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
            {recentEbooks.map((ebook) => (
              <EbookCard key={ebook.id} ebook={ebook} onRead={handleRead} />
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
```

- [ ] **Step 8: Create `frontend/src/pages/EbookListPage.tsx`**

```tsx
import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ebooksApi, Ebook } from '../api/ebooks';
import { historyApi } from '../api/history';
import EbookCard from '../components/ebook/EbookCard';
import EbookUpload from '../components/ebook/EbookUpload';
import EmptyState from '../components/common/EmptyState';

export default function EbookListPage() {
  const [ebooks, setEbooks] = useState<Ebook[]>([]);
  const [showUpload, setShowUpload] = useState(false);
  const navigate = useNavigate();

  const loadEbooks = () => {
    ebooksApi.list(1, 50).then(({ ebooks }) => setEbooks(ebooks));
  };

  useEffect(() => {
    loadEbooks();
  }, []);

  const handleRead = async (id: string) => {
    await historyApi.openBook(id);
    navigate(`/read/${id}`);
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this ebook?')) return;
    await ebooksApi.delete(id);
    loadEbooks();
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Ebooks</h1>
        <button
          onClick={() => setShowUpload(true)}
          className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
        >
          Upload Ebook
        </button>
      </div>

      {ebooks.length === 0 ? (
        <EmptyState title="No ebooks yet" description="Upload your first PDF ebook to get started" />
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
          {ebooks.map((ebook) => (
            <EbookCard key={ebook.id} ebook={ebook} onRead={handleRead} onDelete={handleDelete} />
          ))}
        </div>
      )}

      <EbookUpload isOpen={showUpload} onClose={() => setShowUpload(false)} onSuccess={loadEbooks} />
    </div>
  );
}
```

- [ ] **Step 9: Update `frontend/src/router/index.tsx` — replace placeholders with real pages**

Replace the `PlaceholderPage` imports and route elements for `/` and `/ebooks`:

```tsx
import DashboardPage from '../pages/DashboardPage';
import EbookListPage from '../pages/EbookListPage';
```

```tsx
<Route path="/" element={<DashboardPage />} />
<Route path="/ebooks" element={<EbookListPage />} />
```

Keep `/read/:id` and `/history` as placeholders for now.

- [ ] **Step 10: Verify with Docker**

```bash
docker-compose up --build -d
```

Open `http://localhost`. Login → Dashboard shows "Continue Reading" and "Recently Added". Navigate to `/ebooks` → shows grid with upload button. Upload a PDF → card appears.

- [ ] **Step 11: Commit**

```bash
git add .
git commit -m "feat: dashboard and ebook list pages with upload modal"
```

---

### Task 10: Frontend — PDF Reader Page

**Files:**
- Create: `frontend/src/api/progress.ts`
- Create: `frontend/src/api/bookmarks.ts`
- Create: `frontend/src/components/reader/PdfReader.tsx`
- Create: `frontend/src/components/reader/PageControls.tsx`
- Create: `frontend/src/components/reader/BookmarkButton.tsx`
- Create: `frontend/src/components/reader/ProgressBar.tsx`
- Create: `frontend/src/hooks/useDebounce.ts`
- Create: `frontend/src/hooks/useBeforeUnload.ts`
- Create: `frontend/src/pages/ReaderPage.tsx`
- Modify: `frontend/src/router/index.tsx`

**Interfaces:**
- Consumes:
  - `ebooksApi.getById()`, `ebooksApi.getFileUrl()`
  - `historyApi.openBook()` — returns `{last_page}`
  - Backend endpoints: `PUT /api/ebooks/:id/progress`, `GET/POST/DELETE bookmarks`
- Produces:
  - `progressApi.update(ebookId, page)` — debounced progress save
  - `bookmarksApi.list(ebookId)`, `.create(ebookId, page, note)`, `.delete(id)`
  - Full PDF reader with 1-page/2-page toggle, page navigation, bookmarks panel, progress bar
  - `useDebounce(value, delay)` hook
  - `useBeforeUnload(callback)` hook for saving on tab close

- [ ] **Step 1: Create `frontend/src/api/progress.ts`**

```typescript
import client from './client';

export const progressApi = {
  get: async (ebookId: string) => {
    const res = await client.get<{ success: boolean; data: { last_page: number; status: string } }>(
      `/ebooks/${ebookId}/progress`
    );
    return res.data.data;
  },

  update: async (ebookId: string, page: number) => {
    await client.put(`/ebooks/${ebookId}/progress`, { page });
  },
};
```

- [ ] **Step 2: Create `frontend/src/api/bookmarks.ts`**

```typescript
import client from './client';

export interface Bookmark {
  id: string;
  ebook_id: string;
  page_number: number;
  note: string;
  created_at: string;
}

export const bookmarksApi = {
  list: async (ebookId: string) => {
    const res = await client.get<{ success: boolean; data: Bookmark[] }>(`/ebooks/${ebookId}/bookmarks`);
    return res.data.data;
  },

  create: async (ebookId: string, pageNumber: number, note = '') => {
    const res = await client.post<{ success: boolean; data: Bookmark }>(`/ebooks/${ebookId}/bookmarks`, {
      page_number: pageNumber,
      note,
    });
    return res.data.data;
  },

  delete: async (id: string) => {
    await client.delete(`/bookmarks/${id}`);
  },
};
```

- [ ] **Step 3: Create `frontend/src/hooks/useDebounce.ts`**

```typescript
import { useEffect, useRef } from 'react';

export function useDebouncedCallback<T extends (...args: unknown[]) => void>(
  callback: T,
  delay: number
): T {
  const timeoutRef = useRef<ReturnType<typeof setTimeout>>(null);

  useEffect(() => {
    return () => {
      if (timeoutRef.current) clearTimeout(timeoutRef.current);
    };
  }, []);

  return ((...args: unknown[]) => {
    if (timeoutRef.current) clearTimeout(timeoutRef.current);
    timeoutRef.current = setTimeout(() => callback(...args), delay);
  }) as T;
}
```

- [ ] **Step 4: Create `frontend/src/hooks/useBeforeUnload.ts`**

```typescript
import { useEffect, useRef } from 'react';

export function useBeforeUnload(callback: () => void) {
  const callbackRef = useRef(callback);
  callbackRef.current = callback;

  useEffect(() => {
    const handler = () => callbackRef.current();
    window.addEventListener('beforeunload', handler);
    return () => window.removeEventListener('beforeunload', handler);
  }, []);
}
```

- [ ] **Step 5: Create `frontend/src/components/reader/ProgressBar.tsx`**

```tsx
interface ProgressBarProps {
  currentPage: number;
  totalPages: number;
}

export default function ProgressBar({ currentPage, totalPages }: ProgressBarProps) {
  const percent = Math.round((currentPage / totalPages) * 100);

  return (
    <div className="flex items-center gap-3">
      <div className="flex-1 bg-gray-200 rounded-full h-2">
        <div
          className="bg-blue-600 h-2 rounded-full transition-all"
          style={{ width: `${percent}%` }}
        />
      </div>
      <span className="text-sm text-gray-500 whitespace-nowrap">{percent}%</span>
    </div>
  );
}
```

- [ ] **Step 6: Create `frontend/src/components/reader/PageControls.tsx`**

```tsx
interface PageControlsProps {
  currentPage: number;
  totalPages: number;
  onPageChange: (page: number) => void;
  dualPage: boolean;
  onToggleDualPage: () => void;
}

export default function PageControls({
  currentPage,
  totalPages,
  onPageChange,
  dualPage,
  onToggleDualPage,
}: PageControlsProps) {
  const step = dualPage ? 2 : 1;

  return (
    <div className="flex items-center justify-between bg-white border-t border-gray-200 px-4 py-3">
      <button
        onClick={() => onPageChange(Math.max(1, currentPage - step))}
        disabled={currentPage <= 1}
        className="px-4 py-2 text-sm bg-gray-100 rounded hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        Previous
      </button>

      <div className="flex items-center gap-4">
        <div className="flex items-center gap-2">
          <span className="text-sm text-gray-600">Page</span>
          <input
            type="number"
            value={currentPage}
            onChange={(e) => {
              const p = parseInt(e.target.value);
              if (p >= 1 && p <= totalPages) onPageChange(p);
            }}
            className="w-16 px-2 py-1 text-center border border-gray-300 rounded text-sm"
            min={1}
            max={totalPages}
          />
          <span className="text-sm text-gray-600">of {totalPages}</span>
        </div>

        <button
          onClick={onToggleDualPage}
          className="px-3 py-1.5 text-xs border border-gray-300 rounded hover:bg-gray-50"
        >
          {dualPage ? '1 Page' : '2 Pages'}
        </button>
      </div>

      <button
        onClick={() => onPageChange(Math.min(totalPages, currentPage + step))}
        disabled={currentPage >= totalPages}
        className="px-4 py-2 text-sm bg-gray-100 rounded hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        Next
      </button>
    </div>
  );
}
```

- [ ] **Step 7: Create `frontend/src/components/reader/BookmarkButton.tsx`**

```tsx
import { useState, useEffect } from 'react';
import { bookmarksApi, Bookmark } from '../../api/bookmarks';

interface BookmarkButtonProps {
  ebookId: string;
  currentPage: number;
}

export default function BookmarkButton({ ebookId, currentPage }: BookmarkButtonProps) {
  const [bookmarks, setBookmarks] = useState<Bookmark[]>([]);
  const [showList, setShowList] = useState(false);

  const loadBookmarks = () => {
    bookmarksApi.list(ebookId).then(setBookmarks);
  };

  useEffect(() => {
    loadBookmarks();
  }, [ebookId]);

  const isCurrentPageBookmarked = bookmarks.some((b) => b.page_number === currentPage);

  const toggleBookmark = async () => {
    if (isCurrentPageBookmarked) {
      const bm = bookmarks.find((b) => b.page_number === currentPage);
      if (bm) {
        await bookmarksApi.delete(bm.id);
        loadBookmarks();
      }
    } else {
      await bookmarksApi.create(ebookId, currentPage);
      loadBookmarks();
    }
  };

  return (
    <div className="relative">
      <div className="flex items-center gap-2">
        <button
          onClick={toggleBookmark}
          className={`px-3 py-1.5 text-sm rounded border ${
            isCurrentPageBookmarked
              ? 'bg-yellow-50 border-yellow-300 text-yellow-700'
              : 'border-gray-300 text-gray-600 hover:bg-gray-50'
          }`}
        >
          {isCurrentPageBookmarked ? 'Bookmarked' : 'Bookmark'}
        </button>
        <button
          onClick={() => setShowList(!showList)}
          className="px-2 py-1.5 text-sm border border-gray-300 rounded hover:bg-gray-50"
        >
          {bookmarks.length}
        </button>
      </div>

      {showList && bookmarks.length > 0 && (
        <div className="absolute right-0 mt-2 w-48 bg-white border border-gray-200 rounded-lg shadow-lg z-10">
          <div className="p-2 max-h-60 overflow-y-auto">
            {bookmarks.map((b) => (
              <button
                key={b.id}
                onClick={() => {
                  window.dispatchEvent(new CustomEvent('goto-page', { detail: b.page_number }));
                  setShowList(false);
                }}
                className="w-full text-left px-3 py-2 text-sm hover:bg-gray-50 rounded"
              >
                Page {b.page_number}
                {b.note && <span className="block text-xs text-gray-400">{b.note}</span>}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 8: Create `frontend/src/components/reader/PdfReader.tsx`**

```tsx
import { useState, useEffect } from 'react';
import { Document, Page, pdfjs } from 'react-pdf';
import 'react-pdf/dist/esm/Page/AnnotationLayer.css';
import 'react-pdf/dist/esm/Page/TextLayer.css';

pdfjs.GlobalWorkerOptions.workerSrc = new URL(
  'pdfjs-dist/build/pdf.worker.min.mjs',
  import.meta.url,
).toString();

interface PdfReaderProps {
  fileUrl: string;
  currentPage: number;
  dualPage: boolean;
  onPageChange: (page: number) => void;
  onDocumentLoad: (numPages: number) => void;
}

export default function PdfReader({
  fileUrl,
  currentPage,
  dualPage,
  onPageChange,
  onDocumentLoad,
}: PdfReaderProps) {
  const [numPages, setNumPages] = useState(0);
  const [containerWidth, setContainerWidth] = useState(800);

  useEffect(() => {
    const updateWidth = () => {
      const el = document.getElementById('pdf-container');
      if (el) setContainerWidth(el.clientWidth);
    };
    updateWidth();
    window.addEventListener('resize', updateWidth);
    return () => window.removeEventListener('resize', updateWidth);
  }, []);

  useEffect(() => {
    const handler = (e: Event) => {
      const page = (e as CustomEvent).detail;
      if (page >= 1 && page <= numPages) onPageChange(page);
    };
    window.addEventListener('goto-page', handler);
    return () => window.removeEventListener('goto-page', handler);
  }, [numPages, onPageChange]);

  const onDocLoad = ({ numPages: n }: { numPages: number }) => {
    setNumPages(n);
    onDocumentLoad(n);
  };

  const pageWidth = dualPage ? (containerWidth - 16) / 2 : containerWidth;

  return (
    <div id="pdf-container" className="flex-1 overflow-auto bg-gray-100 flex justify-center p-4">
      <Document file={fileUrl} onLoadSuccess={onDocLoad} loading={<div className="text-gray-500">Loading PDF...</div>}>
        <div className={`flex ${dualPage ? 'gap-4' : ''}`}>
          <Page
            pageNumber={currentPage}
            width={pageWidth}
            renderTextLayer={true}
            renderAnnotationLayer={true}
          />
          {dualPage && currentPage + 1 <= numPages && (
            <Page
              pageNumber={currentPage + 1}
              width={pageWidth}
              renderTextLayer={true}
              renderAnnotationLayer={true}
            />
          )}
        </div>
      </Document>
    </div>
  );
}
```

- [ ] **Step 9: Create `frontend/src/pages/ReaderPage.tsx`**

```tsx
import { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ebooksApi, Ebook } from '../api/ebooks';
import { historyApi } from '../api/history';
import { progressApi } from '../api/progress';
import PdfReader from '../components/reader/PdfReader';
import PageControls from '../components/reader/PageControls';
import ProgressBar from '../components/reader/ProgressBar';
import BookmarkButton from '../components/reader/BookmarkButton';
import { useDebouncedCallback } from '../hooks/useDebounce';
import { useBeforeUnload } from '../hooks/useBeforeUnload';

export default function ReaderPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [ebook, setEbook] = useState<Ebook | null>(null);
  const [fileUrl, setFileUrl] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const [totalPages, setTotalPages] = useState(0);
  const [dualPage, setDualPage] = useState(window.innerWidth > 1024);
  const [loading, setLoading] = useState(true);

  const saveProgress = useDebouncedCallback((page: number) => {
    if (id) progressApi.update(id, page);
  }, 3000);

  useBeforeUnload(() => {
    if (id) {
      const data = JSON.stringify({ page: currentPage });
      navigator.sendBeacon(`/api/ebooks/${id}/progress`, new Blob([data], { type: 'application/json' }));
    }
  });

  useEffect(() => {
    if (!id) return;

    const init = async () => {
      try {
        const [ebookData, openData, url] = await Promise.all([
          ebooksApi.getById(id),
          historyApi.openBook(id),
          ebooksApi.getFileUrl(id),
        ]);
        setEbook(ebookData);
        setCurrentPage(openData.last_page);
        setFileUrl(url);
      } catch {
        navigate('/');
      } finally {
        setLoading(false);
      }
    };
    init();
  }, [id, navigate]);

  const handlePageChange = useCallback(
    (page: number) => {
      setCurrentPage(page);
      saveProgress(page);
    },
    [saveProgress]
  );

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-gray-500">Loading book...</div>
      </div>
    );
  }

  if (!ebook || !fileUrl) return null;

  return (
    <div className="h-screen flex flex-col bg-gray-100">
      <div className="bg-white border-b border-gray-200 px-4 py-2 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <button onClick={() => navigate(-1)} className="text-gray-600 hover:text-gray-900">
            &larr; Back
          </button>
          <h1 className="text-lg font-semibold text-gray-900 truncate max-w-md">{ebook.title}</h1>
        </div>
        <div className="flex items-center gap-4">
          <BookmarkButton ebookId={ebook.id} currentPage={currentPage} />
          <div className="w-40">
            <ProgressBar currentPage={currentPage} totalPages={totalPages || ebook.total_pages} />
          </div>
        </div>
      </div>

      <PdfReader
        fileUrl={fileUrl}
        currentPage={currentPage}
        dualPage={dualPage}
        onPageChange={handlePageChange}
        onDocumentLoad={setTotalPages}
      />

      <PageControls
        currentPage={currentPage}
        totalPages={totalPages || ebook.total_pages}
        onPageChange={handlePageChange}
        dualPage={dualPage}
        onToggleDualPage={() => setDualPage(!dualPage)}
      />
    </div>
  );
}
```

- [ ] **Step 10: Update `frontend/src/router/index.tsx` — add ReaderPage**

```tsx
import ReaderPage from '../pages/ReaderPage';
```

Replace the reader route:
```tsx
<Route path="/read/:id" element={<ReaderPage />} />
```

Note: ReaderPage should not be wrapped in `<Layout>` since it uses the full viewport. Update the router to handle this — move the reader route outside the layout group:

```tsx
<Route
  path="/read/:id"
  element={
    <ProtectedRoute>
      <ReaderPage />
    </ProtectedRoute>
  }
/>
```

- [ ] **Step 11: Verify PDF reader works**

```bash
docker-compose up --build -d
```

Login, upload a PDF, click "Read". Expected: PDF renders, 2-page layout on desktop, page controls work, bookmark button works, progress bar updates on page change.

- [ ] **Step 12: Commit**

```bash
git add .
git commit -m "feat: PDF reader with dual-page view, bookmarks, and debounced progress saving"
```

---

### Task 11: Frontend — History Page

**Files:**
- Create: `frontend/src/pages/HistoryPage.tsx`
- Modify: `frontend/src/router/index.tsx`

**Interfaces:**
- Consumes: `historyApi.getHistory()` → `{reading: HistoryItem[], completed: HistoryItem[]}`
- Produces: HistoryPage with "In Progress" and "Completed" tabs showing book cards with progress

- [ ] **Step 1: Create `frontend/src/pages/HistoryPage.tsx`**

```tsx
import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { historyApi, HistoryItem } from '../api/history';
import EbookCard from '../components/ebook/EbookCard';
import EmptyState from '../components/common/EmptyState';

export default function HistoryPage() {
  const [reading, setReading] = useState<HistoryItem[]>([]);
  const [completed, setCompleted] = useState<HistoryItem[]>([]);
  const [tab, setTab] = useState<'reading' | 'completed'>('reading');
  const navigate = useNavigate();

  useEffect(() => {
    historyApi.getHistory().then((data) => {
      setReading(data.reading || []);
      setCompleted(data.completed || []);
    });
  }, []);

  const handleRead = async (id: string) => {
    await historyApi.openBook(id);
    navigate(`/read/${id}`);
  };

  const items = tab === 'reading' ? reading : completed;

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900 mb-6">Reading History</h1>

      <div className="flex gap-1 mb-6 bg-gray-100 p-1 rounded-lg w-fit">
        <button
          onClick={() => setTab('reading')}
          className={`px-4 py-2 text-sm rounded-md ${
            tab === 'reading' ? 'bg-white shadow text-gray-900' : 'text-gray-600'
          }`}
        >
          In Progress ({reading.length})
        </button>
        <button
          onClick={() => setTab('completed')}
          className={`px-4 py-2 text-sm rounded-md ${
            tab === 'completed' ? 'bg-white shadow text-gray-900' : 'text-gray-600'
          }`}
        >
          Completed ({completed.length})
        </button>
      </div>

      {items.length === 0 ? (
        <EmptyState
          title={tab === 'reading' ? 'No books in progress' : 'No completed books'}
          description={
            tab === 'reading'
              ? 'Start reading a book to see it here'
              : 'Finish reading a book to see it here'
          }
        />
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
          {items.map((item) => (
            <EbookCard
              key={item.ebook_id}
              ebook={{
                id: item.ebook_id,
                title: item.title,
                author: item.author,
                cover_url: item.cover_url,
                total_pages: item.total_pages,
                file_size: 0,
                created_at: item.last_opened,
              }}
              onRead={handleRead}
              progress={{ last_page: item.last_page, total_pages: item.total_pages }}
            />
          ))}
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 2: Update `frontend/src/router/index.tsx` — add HistoryPage**

```tsx
import HistoryPage from '../pages/HistoryPage';
```

Replace the history route placeholder:
```tsx
<Route path="/history" element={<HistoryPage />} />
```

- [ ] **Step 3: Verify with Docker**

```bash
docker-compose up --build -d
```

Login, navigate to `/history`. Expected: two tabs ("In Progress" and "Completed"), books appear after reading activity.

- [ ] **Step 4: Commit**

```bash
git add .
git commit -m "feat: reading history page with in-progress and completed tabs"
```

---

### Task 12: Mobile Responsive & Final Polish

**Files:**
- Modify: `frontend/src/components/layout/Navbar.tsx` — add mobile hamburger menu
- Modify: `frontend/src/components/reader/PdfReader.tsx` — auto 1-page on mobile
- Modify: `frontend/src/pages/ReaderPage.tsx` — responsive toolbar

**Interfaces:**
- Consumes: all existing components
- Produces: fully responsive UI across mobile, tablet, desktop

- [ ] **Step 1: Update `frontend/src/components/layout/Navbar.tsx` — mobile menu**

```tsx
import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';

export default function Navbar() {
  const { user, logout } = useAuth();
  const [mobileOpen, setMobileOpen] = useState(false);

  return (
    <nav className="bg-white border-b border-gray-200">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between h-16 items-center">
          <div className="flex items-center gap-8">
            <Link to="/" className="text-xl font-bold text-gray-900">Mbaca Buku</Link>
            <div className="hidden sm:flex gap-6">
              <Link to="/" className="text-gray-600 hover:text-gray-900">Dashboard</Link>
              <Link to="/ebooks" className="text-gray-600 hover:text-gray-900">Ebooks</Link>
              <Link to="/history" className="text-gray-600 hover:text-gray-900">History</Link>
            </div>
          </div>
          <div className="hidden sm:flex items-center gap-4">
            <span className="text-sm text-gray-600">{user?.name}</span>
            <button onClick={logout} className="text-sm text-red-600 hover:text-red-800">Logout</button>
          </div>
          <button
            onClick={() => setMobileOpen(!mobileOpen)}
            className="sm:hidden p-2 text-gray-600"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                d={mobileOpen ? 'M6 18L18 6M6 6l12 12' : 'M4 6h16M4 12h16M4 18h16'} />
            </svg>
          </button>
        </div>
      </div>

      {mobileOpen && (
        <div className="sm:hidden border-t border-gray-200 bg-white px-4 py-3 space-y-2">
          <Link to="/" onClick={() => setMobileOpen(false)} className="block py-2 text-gray-600">Dashboard</Link>
          <Link to="/ebooks" onClick={() => setMobileOpen(false)} className="block py-2 text-gray-600">Ebooks</Link>
          <Link to="/history" onClick={() => setMobileOpen(false)} className="block py-2 text-gray-600">History</Link>
          <div className="pt-2 border-t border-gray-100 flex justify-between items-center">
            <span className="text-sm text-gray-600">{user?.name}</span>
            <button onClick={logout} className="text-sm text-red-600">Logout</button>
          </div>
        </div>
      )}
    </nav>
  );
}
```

- [ ] **Step 2: Update PdfReader to auto-detect mobile and default to single page**

In `ReaderPage.tsx`, update the initial `dualPage` state:

```tsx
const [dualPage, setDualPage] = useState(window.innerWidth > 1024);
```

This already defaults to single page on mobile. Also add a resize listener:

```tsx
useEffect(() => {
  const handleResize = () => {
    if (window.innerWidth <= 1024) setDualPage(false);
  };
  window.addEventListener('resize', handleResize);
  return () => window.removeEventListener('resize', handleResize);
}, []);
```

- [ ] **Step 3: Make ReaderPage toolbar responsive**

Update the top bar in ReaderPage to stack on mobile:

```tsx
<div className="bg-white border-b border-gray-200 px-4 py-2 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2">
```

- [ ] **Step 4: Verify responsive behavior**

Open browser DevTools, toggle device toolbar. Check:
- Login page centers properly on mobile
- Navbar collapses to hamburger on mobile
- Ebook grid switches to 1 column on mobile
- Reader shows 1 page on mobile, 2 on desktop
- Page controls are usable on touch

- [ ] **Step 5: Commit**

```bash
git add .
git commit -m "feat: mobile responsive layout and reader polish"
```

---

### Task 13: Final Integration Test & Docker Verification

**Files:**
- No new files — this is a verification task

**Interfaces:**
- Consumes: everything from Tasks 1-12
- Produces: verified, runnable application via `docker-compose up --build`

- [ ] **Step 1: Clean build from scratch**

```bash
cd /Users/fredy/Documents/GitHub/mbaca-buku
docker-compose down -v
docker-compose up --build -d
```

- [ ] **Step 2: Verify all services are healthy**

```bash
docker-compose ps
```

Expected: 6 services running (postgres, redis, minio, backend, frontend, nginx).

- [ ] **Step 3: Test full user flow**

1. Open `http://localhost` → redirected to `/login`
2. Login with `admin@mbacabuku.com` / `12345` → Dashboard loads
3. Navigate to `/ebooks` → empty state
4. Upload a PDF → ebook card appears
5. Click "Read" → PDF reader opens
6. Verify 2-page layout on desktop
7. Toggle to 1-page mode → works
8. Navigate pages → progress bar updates
9. Bookmark a page → "Bookmarked" state shows
10. Close tab, reopen → same page resumes
11. Navigate to last page → status becomes "completed"
12. Go to `/history` → book appears in "Completed" tab

- [ ] **Step 4: Test API directly**

```bash
# Health check
curl http://localhost/api/health

# Login
TOKEN=$(curl -s -X POST http://localhost/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@mbacabuku.com","password":"12345"}' | jq -r '.data.token')

echo "Token: $TOKEN"

# List ebooks
curl http://localhost/api/ebooks -H "Authorization: Bearer $TOKEN"

# History
curl http://localhost/api/history -H "Authorization: Bearer $TOKEN"
```

- [ ] **Step 5: Verify data persists across restarts**

```bash
docker-compose down
docker-compose up -d
```

Login again → previous ebooks and progress still present (volumes persist).

- [ ] **Step 6: Commit final state**

```bash
git add .
git commit -m "chore: final integration verification"
```
