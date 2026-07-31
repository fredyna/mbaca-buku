# Explicit Reading Completion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A book becomes `completed` only when the reader confirms it from the last page, and the Completed tab distinguishes a finished book from one being re-read.

**Architecture:** The server stops inferring completion from page numbers and exposes an explicit status endpoint. The reader keeps the current status in component state and renders a single toggle button, shown only while the last page is visible. The history cards gain a caption prop so the same card renders `Progress`, `Re-read progress`, or a plain `Finished` note.

**Tech Stack:** Go 1.25 + Gin + Postgres + Redis (backend); React 19 + TypeScript + Tailwind + Vite (frontend); Docker Compose for the running stack.

## Global Constraints

- Status values are exactly `completed` and `reading`. Any other value is rejected with HTTP 400 and code `VALIDATION_ERROR`.
- Changing status must never modify `last_page`.
- All user-facing copy is English: `Mark as Completed`, `Mark as In Progress`, `Re-read progress`, `Finished`, `Progress`.
- The last page counts as visible when `currentPage >= totalPages || currentPage + 1 >= totalPages` in dual-page mode.
- Repository has no JS test runner and no Go test harness for DB/Redis-backed services; those tasks are verified end-to-end against the Docker stack, per the spec.

---

### Task 1: Explicit status in the backend

**Files:**
- Modify: `backend/internal/dto/progress_dto.go`
- Modify: `backend/internal/service/reading_service.go`
- Modify: `backend/internal/handler/reading_handler.go`
- Modify: `backend/internal/router/router.go`

**Interfaces:**
- Produces: `PUT /api/ebooks/:id/status` accepting `{"status":"completed"|"reading"}` and returning `{"ebook_id","last_page","status"}`; `POST /api/ebooks/:id/open` response gains `"status"`.
- Produces: `ReadingService.SetStatus(ctx, userID, ebookID, status string) (int, error)` returning the unchanged last page; `ReadingService.OpenBook(ctx, userID, ebookID) (int, string, error)`.

- [ ] **Step 1: Add the request DTO and the status field**

In `progress_dto.go`:

```go
type SetStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=reading completed"`
}
```

and add `Status string \`json:"status"\`` to `OpenBookResponse`.

- [ ] **Step 2: Stop auto-completing and add SetStatus**

In `reading_service.go`, `UpdateProgress` keeps the clamp and the Redis write but returns after them — delete the `status`/`Upsert` block. `OpenBook` returns the stored status alongside the page (`"reading"` when no row exists yet). Add:

```go
// SetStatus records an explicit completion decision by the reader. The saved
// page is left untouched so a finished book reopens where it was left.
func (s *ReadingService) SetStatus(ctx context.Context, userID, ebookID, status string) (int, error) {
	if status != "reading" && status != "completed" {
		return 0, ErrInvalidStatus
	}
	if _, err := s.ebookRepo.GetByID(ctx, ebookID); err != nil {
		return 0, fmt.Errorf("ebook not found")
	}
	page, _, err := s.GetProgress(ctx, userID, ebookID)
	if err != nil {
		return 0, err
	}
	if err := s.progressRepo.Upsert(ctx, userID, ebookID, page, status); err != nil {
		return 0, err
	}
	return page, nil
}
```

with `var ErrInvalidStatus = errors.New("status must be reading or completed")`.

- [ ] **Step 3: Add the handler and route**

`SetStatus` handler binds `dto.SetStatusRequest`, maps `service.ErrInvalidStatus` to 400 `VALIDATION_ERROR`, and returns `dto.ProgressResponse`. `OpenBook` handler passes the new status through. Register `ebooks.PUT("/:id/status", cfg.ReadingHandler.SetStatus)` next to the progress routes in `router.go`.

- [ ] **Step 4: Build and vet**

Run: `cd backend && go build ./... && go vet ./... && go test ./...`
Expected: no output from build/vet, `ok` for `internal/pdfutil`.

- [ ] **Step 5: Verify against the stack**

Run: `docker compose build backend && docker compose up -d backend`, then with a fresh token:

```bash
curl -X PUT localhost/api/ebooks/$ID/status -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"status":"completed"}'
curl -X PUT localhost/api/ebooks/$ID/status -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"status":"finished"}'
```

Expected: first returns `"status":"completed"` with `last_page` unchanged; second returns 400. Then `PUT /progress` with the last page and confirm `GET /progress` still reports the status the user set, not one derived from the page.

---

### Task 2: Completion button in the reader

**Files:**
- Modify: `frontend/src/api/progress.ts`
- Modify: `frontend/src/api/history.ts`
- Modify: `frontend/src/pages/ReaderPage.tsx`
- Modify: `frontend/src/components/reader/PageControls.tsx`

**Interfaces:**
- Consumes: the endpoint and `status` field from Task 1.
- Produces: `progressApi.setStatus(ebookId: string, status: 'reading' | 'completed'): Promise<void>`; `PageControls` props `isCompleted: boolean`, `onToggleCompleted: () => void`.

- [ ] **Step 1: Add the API call**

```ts
setStatus: async (ebookId: string, status: 'reading' | 'completed') => {
  await client.put(`/ebooks/${ebookId}/status`, { status });
},
```

and widen `historyApi.openBook`'s response type with `status: string`.

- [ ] **Step 2: Track status in ReaderPage**

Add `const [isCompleted, setIsCompleted] = useState(false);`, set it from `openData.status === 'completed'` in `init`, and add a handler that flips the value optimistically and calls `progressApi.setStatus`, reverting on failure. Pass `isCompleted` and the handler to `PageControls`.

- [ ] **Step 3: Render the button**

In `PageControls`, compute `const onLastPage = totalPages > 0 && (currentPage >= totalPages || (dualPage && currentPage + 1 >= totalPages));` and render, only when `onLastPage`, a button reading `Mark as Completed` (green) or `Mark as In Progress` (outlined) depending on `isCompleted`.

- [ ] **Step 4: Build**

Run: `cd frontend && npm run build`
Expected: build succeeds with no TypeScript errors.

- [ ] **Step 5: Verify in the browser**

Rebuild the frontend container, open a book, and confirm: no button on page 1; button appears on the final page and on the final spread in 2-page mode; clicking it moves the book between History tabs; the label flips and survives a reload.

---

### Task 3: Re-read wording in the history cards

**Files:**
- Modify: `frontend/src/components/ebook/EbookCard.tsx`
- Modify: `frontend/src/pages/HistoryPage.tsx`

**Interfaces:**
- Produces: `EbookCard` props `progressLabel?: string` (default `'Progress'`) and `note?: string` rendered in place of the progress block.

- [ ] **Step 1: Add the props to EbookCard**

Default `progressLabel = 'Progress'`, use it in both the grid and list layouts where the literal `Progress` appears now, and render `note` as `<p className="text-xs text-gray-500 mt-2">{note}</p>` when `progress` is absent.

- [ ] **Step 2: Wire up HistoryPage**

For the Completed tab, a card whose `last_page >= total_pages` gets `note="Finished"` and no `progress`; any other completed card gets `progress` plus `progressLabel="Re-read progress"`. The In Progress tab is unchanged.

- [ ] **Step 3: Build**

Run: `cd frontend && npm run build`
Expected: build succeeds.

- [ ] **Step 4: Verify in the browser**

Mark a book completed, check the Completed tab shows `Finished` with no bar, then reopen it, page backwards, and confirm the card switches to `Re-read progress` with a partial bar while staying in the Completed tab.

---

### Task 4: Full pass over the running stack

**Files:** none

- [ ] **Step 1: Rebuild everything**

Run: `docker compose up -d --build backend frontend nginx`

- [ ] **Step 2: Walk the whole flow**

Fresh book → read to the end → confirm it stays In Progress until the button is pressed → press it → Completed tab shows `Finished` → reopen and page back → `Re-read progress` → press `Mark as In Progress` → back in the In Progress tab.

- [ ] **Step 3: Confirm the database agrees**

Run: `docker compose exec -T postgres psql -c "SELECT ebook_id, last_page, status FROM reading_progress;"`
Expected: statuses match what was clicked, and `last_page` never jumped when a status changed.
