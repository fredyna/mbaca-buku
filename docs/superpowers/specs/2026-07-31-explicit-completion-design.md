# Explicit Reading Completion

Date: 2026-07-31

## Problem

A book is marked `completed` automatically the moment saved progress reaches the
last page (`ReadingService.UpdateProgress`). Merely paging to the end — or
jumping there to check something — files the book under Completed without the
reader ever saying they finished it. Completion should be a decision, not a side
effect of scrolling.

## Behaviour

1. **Completion is explicit.** The server never derives `completed` from a page
   number. The status changes only when the reader asks for it.
2. **The button appears on the last page only.** While the last page is on
   screen, the reader shows **Mark as Completed**. In dual-page mode the last
   page counts as visible when `currentPage + 1 >= totalPages`, so the button
   still appears on the final spread.
3. **Confirming is reversible.** Once completed, the same control reads **Mark
   as In Progress** and moves the book back.
4. **Reopening keeps the book completed.** A completed book stays in the
   Completed tab; its saved page keeps updating and represents re-read
   position.
5. **Re-reading is labelled as such.** In the Completed tab a card whose saved
   page is behind the last page shows `Re-read progress` with its bar. A card
   still parked on the last page — completed but not re-read — shows the plain
   text `Finished` and no bar, so a full bar never implies a re-read that did
   not happen.

## API

```
PUT /api/ebooks/:id/status
body:     {"status": "completed"} | {"status": "reading"}
200:      {"ebook_id": "...", "last_page": 12, "status": "completed"}
400:      any other status value
```

Setting a status never moves `last_page`; the reader keeps its position.

`POST /api/ebooks/:id/open` gains a `status` field so the reader knows which
button to render without a second request.

## Backend changes

| File | Change |
| --- | --- |
| `internal/service/reading_service.go` | `UpdateProgress` stops touching status; new `SetStatus(ctx, userID, ebookID, status)`; `OpenBook` also returns the status |
| `internal/handler/reading_handler.go` | `SetStatus` handler, validates the value |
| `internal/dto/progress_dto.go` | `SetStatusRequest`; `status` added to `OpenBookResponse` |
| `internal/router/router.go` | route registration |

The 30-second flusher already carries the stored status forward when it writes
cached pages, so background flushes cannot silently un-complete a book.

## Frontend changes

| File | Change |
| --- | --- |
| `api/progress.ts` | `setStatus(ebookId, status)` |
| `pages/ReaderPage.tsx` | holds `status` from the open call; passes status and handler down |
| `components/reader/PageControls.tsx` | renders the completion button when the last page is visible |
| `components/ebook/EbookCard.tsx` | `progressLabel` prop (default `Progress`) and a `note` prop for text shown in place of the bar |
| `pages/HistoryPage.tsx` | Completed tab passes `Re-read progress`, or `note="Finished"` when the reader is still on the last page |

## Verification

`pdfutil` aside, this repository has no test harness, and the completion logic
lives in a service wired to a live database and Redis. Verification is therefore
end-to-end against the running stack:

- `PUT /status` with `completed`, `reading`, and a bogus value (expect 400)
- reaching the last page leaves `status` unchanged until the button is pressed
- a completed book stays completed after `POST /open`
- the History tabs sort the book correctly after each transition

## Out of scope

Adding a JavaScript test runner, and any change to how progress pages are
flushed to the database.
