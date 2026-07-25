package service

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
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

		parts := splitRedisKey(key)
		if parts == nil {
			continue
		}
		userID := parts[0]
		ebookID := parts[1]

		progress, _ := s.progressRepo.GetByUserAndEbook(ctx, userID, ebookID)
		status := "reading"
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
