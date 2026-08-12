package service

import (
	"context"
	"log"
	"time"

	"github.com/fredy/mbaca-buku/internal/model"
	"github.com/fredy/mbaca-buku/pkg/useragent"
)

const (
	// activeWindow is how long one 'active' row stands for. The admin UI uses
	// the same span to decide whether a user counts as online, so somebody who
	// is actively reading never renders as offline.
	activeWindow = 5 * time.Minute

	activityRetention       = 90 * 24 * time.Hour
	activityCleanupInterval = 24 * time.Hour

	// activityWriteTimeout bounds a write started from a goroutine that has no
	// request context left to inherit a deadline from.
	activityWriteTimeout = 5 * time.Second
)

// ActivityStore is the subset of the activity repository this service needs.
type ActivityStore interface {
	Insert(ctx context.Context, a *model.UserActivity) error
	DeleteOlderThan(ctx context.Context, age time.Duration) (int64, error)
}

// Throttler rate-limits by key. Implemented by cache.RedisThrottle; declared
// here as an interface so the throttling behaviour can be tested without Redis.
type Throttler interface {
	Allow(ctx context.Context, key string, window time.Duration) bool
}

type ActivityService struct {
	repo      ActivityStore
	throttler Throttler
}

func NewActivityService(repo ActivityStore, throttler Throttler) *ActivityService {
	return &ActivityService{repo: repo, throttler: throttler}
}

// Record stores one activity row.
//
// 'active' events are throttled to one row per user per activeWindow, so an
// idle user costs nothing and a busy one costs about twelve rows an hour.
// 'login' events always land: each sign-in is a distinct thing worth seeing,
// and there are few of them.
//
// It returns nothing on purpose. This is bookkeeping attached to somebody
// else's request, and no failure here should be visible to them — problems go
// to the server log instead.
func (s *ActivityService) Record(ctx context.Context, userID, event string, meta model.RequestMeta) {
	if userID == "" {
		return
	}

	if event == model.EventActive && !s.throttler.Allow(ctx, "activity:seen:"+userID, activeWindow) {
		return
	}

	info := useragent.Parse(meta.UserAgent)
	err := s.repo.Insert(ctx, &model.UserActivity{
		UserID:    userID,
		Event:     event,
		OS:        info.OS,
		Browser:   info.Browser,
		Device:    info.Device,
		IPAddress: meta.IP,
		UserAgent: meta.UserAgent,
	})
	if err != nil {
		log.Printf("activity log: could not record %s for user %s: %v", event, userID, err)
	}
}

// RecordAsync records off the request's critical path, so a slow database never
// slows a page down.
//
// The caller must already have copied meta out of its *gin.Context, which does
// not outlive the request; the request's context is deliberately not inherited
// for the same reason — it is cancelled the moment the response is written.
// The goroutine count is bounded by the throttle for 'active' events and by the
// login rate for 'login' events.
func (s *ActivityService) RecordAsync(userID, event string, meta model.RequestMeta) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), activityWriteTimeout)
		defer cancel()
		s.Record(ctx, userID, event, meta)
	}()
}

// StartCleanup prunes logs past the retention window: once now, then daily
// until ctx is cancelled. Running it in-process means retention survives a
// container restart without any external scheduler, and follows the same shape
// as ReadingService.StartFlusher.
func (s *ActivityService) StartCleanup(ctx context.Context) {
	go func() {
		s.cleanup(ctx)

		ticker := time.NewTicker(activityCleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.cleanup(ctx)
			}
		}
	}()
	log.Printf("Activity log cleanup started (%d day retention, daily sweep)",
		int(activityRetention.Hours()/24))
}

func (s *ActivityService) cleanup(ctx context.Context) {
	removed, err := s.repo.DeleteOlderThan(ctx, activityRetention)
	if err != nil {
		log.Printf("activity log: cleanup failed: %v", err)
		return
	}
	if removed > 0 {
		log.Printf("activity log: removed %d rows older than %d days",
			removed, int(activityRetention.Hours()/24))
	}
}
