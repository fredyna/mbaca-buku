# Mbaca Buku — Design Specification

> **Sebagian usang.** Sejak 2026-08-03 storage berpindah dari MinIO ke
> Cloudflare R2, dan nginx dipublikasikan di port 6900. Lihat
> [2026-08-03-migrasi-r2-design.md](2026-08-03-migrasi-r2-design.md).
> Dokumen ini dipertahankan sebagai catatan desain awal.

## Overview

Mbaca Buku is a full-stack ebook reading platform where users can manage PDF ebooks, read them with a realistic book-like experience, automatically track reading progress, and maintain reading history.

## Decisions

- **Architecture:** Monorepo (`backend/` + `frontend/`), single Docker Compose
- **Backend:** Go + Gin framework, clean architecture (handler → service → repository)
- **Frontend:** React + Vite + TypeScript, Tailwind CSS, react-pdf for PDF rendering
- **Database:** PostgreSQL 16
- **Cache:** Redis 7 (reading progress hot cache)
- **Storage:** MinIO (S3-compatible, for PDF and cover files)
- **Auth:** JWT-based (email/password registration + login)
- **Default user:** Seeded on first run — username: `admin`, password: `12345`
- **Proxy:** Nginx reverse proxy (serves frontend, proxies `/api/*` to backend)

## System Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Docker Compose                        │
│                                                          │
│  ┌──────────┐    ┌──────────┐    ┌──────────────────┐   │
│  │ Frontend │───▶│  Nginx   │───▶│   Go Backend     │   │
│  │ React    │    │ (proxy)  │    │   (Gin + REST)   │   │
│  │ :5173    │    │  :80     │    │   :8080          │   │
│  └──────────┘    └──────────┘    └────┬───┬───┬─────┘   │
│                                       │   │   │         │
│                          ┌────────────┘   │   └───────┐ │
│                          ▼                ▼           ▼ │
│                    ┌──────────┐   ┌─────────┐  ┌──────┐│
│                    │PostgreSQL│   │  Redis   │  │MinIO ││
│                    │  :5432   │   │  :6379   │  │:9000 ││
│                    └──────────┘   └─────────┘  └──────┘│
│                                                          │
└─────────────────────────────────────────────────────────┘
```

Request flow:
1. User opens app → Nginx serves React SPA
2. React makes API calls → Nginx proxies `/api/*` to Go backend on `:8080`
3. Backend handles business logic, talks to PostgreSQL, Redis, and MinIO

In development: React runs on Vite dev server (`:5173`) with HMR, proxies API calls directly to backend.

## Database Schema

### users
| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PK |
| name | VARCHAR(255) | NOT NULL |
| email | VARCHAR(255) | UNIQUE, NOT NULL |
| password_hash | VARCHAR(255) | NOT NULL |
| created_at | TIMESTAMP | DEFAULT NOW() |
| updated_at | TIMESTAMP | DEFAULT NOW() |

### ebooks
| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PK |
| title | VARCHAR(255) | NOT NULL |
| author | VARCHAR(255) | |
| cover_url | TEXT | MinIO URL to cover image |
| file_url | TEXT | NOT NULL, MinIO URL to PDF |
| file_size | BIGINT | bytes |
| total_pages | INT | NOT NULL |
| uploaded_by | UUID | FK → users.id |
| created_at | TIMESTAMP | DEFAULT NOW() |
| updated_at | TIMESTAMP | DEFAULT NOW() |

### reading_progress
| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PK |
| user_id | UUID | FK → users.id |
| ebook_id | UUID | FK → ebooks.id |
| last_page | INT | NOT NULL, DEFAULT 1 |
| status | VARCHAR(20) | 'reading' or 'completed' |
| updated_at | TIMESTAMP | DEFAULT NOW() |
| | | UNIQUE(user_id, ebook_id) |

### bookmarks
| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PK |
| user_id | UUID | FK → users.id |
| ebook_id | UUID | FK → ebooks.id |
| page_number | INT | NOT NULL |
| note | TEXT | optional |
| created_at | TIMESTAMP | DEFAULT NOW() |
| | | UNIQUE(user_id, ebook_id, page_number) |

### history
| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PK |
| user_id | UUID | FK → users.id |
| ebook_id | UUID | FK → ebooks.id |
| opened_at | TIMESTAMP | DEFAULT NOW() |
| | | INDEX(user_id, opened_at DESC) |

## Backend Structure

```
backend/
├── cmd/server/main.go
├── internal/
│   ├── config/config.go
│   ├── middleware/auth.go, cors.go
│   ├── model/user.go, ebook.go, reading_progress.go, bookmark.go, history.go
│   ├── dto/auth_dto.go, ebook_dto.go, progress_dto.go, bookmark_dto.go
│   ├── repository/user_repo.go, ebook_repo.go, progress_repo.go, bookmark_repo.go, history_repo.go
│   ├── service/auth_service.go, ebook_service.go, reading_service.go, bookmark_service.go, history_service.go
│   ├── handler/auth_handler.go, ebook_handler.go, reading_handler.go, bookmark_handler.go, history_handler.go
│   ├── router/router.go
│   └── storage/minio.go
├── pkg/
│   ├── database/postgres.go
│   ├── cache/redis.go
│   └── utils/jwt.go, hash.go, response.go
├── migrations/001_init.sql
├── Dockerfile
└── go.mod
```

Layer responsibilities:
- **Handler** — parse HTTP request, validate via DTOs, call service, return response
- **Service** — business logic, orchestrates repository + cache + storage
- **Repository** — pure database access, one method = one query
- **Storage** — MinIO abstraction for upload/download/presigned URLs

## API Design

### Auth (no JWT required)
| Method | Endpoint | Body / Response |
|--------|----------|-----------------|
| POST | `/api/auth/register` | `{name, email, password}` → `{user, token}` |
| POST | `/api/auth/login` | `{email, password}` → `{user, token}` |
| GET | `/api/auth/me` | → `{user}` (JWT required) |

### Ebooks (JWT required)
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/ebooks` | List all ebooks (paginated) |
| GET | `/api/ebooks/:id` | Get ebook detail |
| POST | `/api/ebooks` | Upload PDF (multipart: file + title + author) |
| PUT | `/api/ebooks/:id` | Update metadata |
| DELETE | `/api/ebooks/:id` | Delete ebook + MinIO file |
| GET | `/api/ebooks/:id/file` | Get presigned MinIO URL for PDF |

### Reading Progress (JWT required)
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/ebooks/:id/progress` | Get last read page (Redis → DB fallback) |
| PUT | `/api/ebooks/:id/progress` | `{page}` → save page (Redis + async DB flush) |

### Bookmarks (JWT required)
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/ebooks/:id/bookmarks` | List bookmarks for a book |
| POST | `/api/ebooks/:id/bookmarks` | `{page_number, note?}` → create |
| DELETE | `/api/bookmarks/:id` | Remove bookmark |

### History (JWT required)
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/history` | List history (grouped by reading/completed) |
| POST | `/api/ebooks/:id/open` | Log open event + return last read page |

### Response format
```json
{ "success": true, "data": { ... } }
{ "success": false, "error": { "code": "VALIDATION_ERROR", "message": "..." } }
{ "success": true, "data": [...], "meta": { "page": 1, "per_page": 20, "total": 45 } }
```

## Redis Strategy

Key format: `user:{userId}:book:{bookId}:last_page` → integer

**Read flow (book open):**
1. Check Redis for cached page
2. If miss → query `reading_progress` table
3. Warm cache with DB value
4. Return page number

**Write flow (page turn):**
1. Frontend debounces page changes (3 seconds)
2. Write immediately to Redis
3. If `page >= total_pages` → set `status = 'completed'` in DB
4. Background goroutine flushes Redis → DB every 30 seconds

**TTL:** 24 hours per key, refreshed on every write.

**Tab close:** Frontend uses `navigator.sendBeacon` via `beforeunload` to fire a final progress save.

## Frontend Structure

```
frontend/src/
├── api/client.ts, auth.ts, ebooks.ts, progress.ts, bookmarks.ts, history.ts
├── components/
│   ├── layout/Navbar.tsx, Sidebar.tsx, Layout.tsx
│   ├── reader/PdfReader.tsx, PageControls.tsx, BookmarkButton.tsx, ProgressBar.tsx
│   ├── ebook/EbookCard.tsx, EbookUpload.tsx, EbookDetail.tsx
│   └── common/Button.tsx, Modal.tsx, Loading.tsx, EmptyState.tsx
├── pages/LoginPage.tsx, RegisterPage.tsx, DashboardPage.tsx, EbookListPage.tsx, ReaderPage.tsx, HistoryPage.tsx
├── hooks/useAuth.ts, useDebounce.ts, useBeforeUnload.ts
├── context/AuthContext.tsx
├── router/index.tsx
└── utils/format.ts
```

### Routing
| Path | Page | Auth |
|------|------|------|
| `/login` | LoginPage | No |
| `/register` | RegisterPage | No |
| `/` | DashboardPage | Yes |
| `/ebooks` | EbookListPage | Yes |
| `/read/:id` | ReaderPage | Yes |
| `/history` | HistoryPage | Yes |

### Key behaviors
- **Auth guard:** Unauthenticated users redirect to `/login`. JWT in `localStorage`, sent via axios interceptor.
- **PdfReader:** Uses `react-pdf` canvas rendering. Desktop (>1024px) = 2-page layout, mobile = 1-page. Toggle button to switch.
- **Debounced saves:** 3-second debounce on page turn before calling `PUT /progress`.
- **Dashboard:** "Continue Reading" (in progress) + "Recently Added" sections.
- **History page:** Two tabs — "In Progress" and "Completed" with book cards showing progress percentage.

## Docker Setup

Six services in `docker-compose.yml`:

| Service | Image | Port | Purpose |
|---------|-------|------|---------|
| postgres | postgres:16-alpine | 5432 | Database |
| redis | redis:7-alpine | 6379 | Cache |
| minio | minio/minio | 9000, 9001 | File storage |
| backend | ./backend (multi-stage) | 8080 | Go API |
| frontend | ./frontend (multi-stage) | — | React build |
| nginx | nginx:alpine | 80 | Reverse proxy |

Volumes: `pgdata` (PostgreSQL), `minio_data` (PDFs).

Startup: `docker-compose up --build` runs everything. Backend auto-migrates and seeds admin user.

## Default User Seed

On first run, the backend seeds:
- Name: `admin`
- Email: `admin@mbacabuku.com`
- Password: `12345` (bcrypt hashed)

Skip if user already exists (idempotent).
