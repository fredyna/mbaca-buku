package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/fredy/mbaca-buku/internal/repository"
)

// Reading status values stored in reading_progress.status. A book only becomes
// StatusCompleted when the reader says so from the last page.
const (
	StatusReading   = "reading"
	StatusCompleted = "completed"
)

var ErrInvalidStatus = errors.New("status must be reading or completed")

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

// OpenBook records the visit and reports where the reader left off, along with
// the status they last chose, so the reader can render the right completion
// control without a second request.
func (s *ReadingService) OpenBook(ctx context.Context, userID, ebookID string) (int, string, error) {
	_, err := s.ebookRepo.GetByID(ctx, ebookID)
	if err != nil {
		return 0, "", fmt.Errorf("ebook not found")
	}

	_ = s.historyRepo.LogOpen(ctx, userID, ebookID)

	lastPage := 1
	status := StatusReading

	progress, err := s.progressRepo.GetByUserAndEbook(ctx, userID, ebookID)
	if err == nil && progress != nil {
		lastPage = progress.LastPage
		status = progress.Status
	} else {
		_ = s.progressRepo.Upsert(ctx, userID, ebookID, 1, StatusReading)
	}

	// The cache holds pages that have not been flushed yet, so it wins over the
	// stored page whenever it is present.
	cached, err := s.rdb.Get(ctx, s.redisKey(userID, ebookID)).Result()
	if err == nil {
		if p, err := strconv.Atoi(cached); err == nil {
			lastPage = p
		}
	} else {
		s.rdb.Set(ctx, s.redisKey(userID, ebookID), lastPage, 24*time.Hour)
	}

	return lastPage, status, nil
}

// SetStatus records an explicit completion decision by the reader. The saved
// page is left untouched so a finished book reopens where it was left.
func (s *ReadingService) SetStatus(ctx context.Context, userID, ebookID, status string) (int, error) {
	if status != StatusReading && status != StatusCompleted {
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

func (s *ReadingService) GetProgress(ctx context.Context, userID, ebookID string) (int, string, error) {
	cached, err := s.rdb.Get(ctx, s.redisKey(userID, ebookID)).Result()
	if err == nil {
		if p, err := strconv.Atoi(cached); err == nil {
			progress, _ := s.progressRepo.GetByUserAndEbook(ctx, userID, ebookID)
			status := StatusReading
			if progress != nil {
				status = progress.Status
			}
			return p, status, nil
		}
	}

	progress, err := s.progressRepo.GetByUserAndEbook(ctx, userID, ebookID)
	if err != nil || progress == nil {
		return 1, StatusReading, nil
	}

	s.rdb.Set(ctx, s.redisKey(userID, ebookID), progress.LastPage, 24*time.Hour)
	return progress.LastPage, progress.Status, nil
}

func (s *ReadingService) UpdateProgress(ctx context.Context, userID, ebookID string, page int) error {
	ebook, err := s.ebookRepo.GetByID(ctx, ebookID)
	if err != nil {
		return fmt.Errorf("ebook not found")
	}

	// Keep progress within the document so no screen can report past 100%.
	if page < 1 {
		page = 1
	}
	if ebook.TotalPages > 0 && page > ebook.TotalPages {
		page = ebook.TotalPages
	}

	// Only the page moves here. Reaching the last page does not finish a book:
	// that is the reader's call, made through SetStatus.
	s.rdb.Set(ctx, s.redisKey(userID, ebookID), page, 24*time.Hour)

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

// FlushAll performs a one-off flush of all cached reading progress to the
// database. It is intended to be called during graceful shutdown, after the
// periodic flusher has been stopped, to avoid losing in-memory progress.
func (s *ReadingService) FlushAll(ctx context.Context) {
	log.Println("Flushing all cached reading progress to database...")
	s.flushToDB(ctx)
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

		parts := splitRedisKey(key)
		if parts == nil {
			continue
		}
		userID := parts[0]
		ebookID := parts[1]

		progress, _ := s.progressRepo.GetByUserAndEbook(ctx, userID, ebookID)
		status := StatusReading
		if progress != nil {
			status = progress.Status
		}

		_ = s.progressRepo.Upsert(ctx, userID, ebookID, page, status)
	}
}

// splitRedisKey extracts {userID, ebookID} from a Redis key formatted as
// "user:{userId}:book:{bookId}:last_page". Returns nil if the key doesn't
// match the expected format.
func splitRedisKey(key string) []string {
	const prefix = "user:"
	const bookSep = ":book:"
	const suffix = ":last_page"

	if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, suffix) {
		return nil
	}

	bookIdx := strings.Index(key, bookSep)
	if bookIdx < 0 {
		return nil
	}

	userID := key[len(prefix):bookIdx]

	bookStart := bookIdx + len(bookSep)
	bookEnd := len(key) - len(suffix)
	if bookEnd < bookStart {
		return nil
	}
	ebookID := key[bookStart:bookEnd]

	if userID == "" || ebookID == "" {
		return nil
	}

	return []string{userID, ebookID}
}
