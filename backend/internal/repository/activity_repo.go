package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/fredy/mbaca-buku/internal/model"
)

type ActivityRepository struct {
	db *sql.DB
}

func NewActivityRepository(db *sql.DB) *ActivityRepository {
	return &ActivityRepository{db: db}
}

// Insert writes one activity row. id and created_at come from column defaults;
// nothing reads them back, so the round trip of a RETURNING clause is skipped.
func (r *ActivityRepository) Insert(ctx context.Context, a *model.UserActivity) error {
	query := `INSERT INTO user_activity_logs
		(user_id, event, os, browser, device, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, query,
		a.UserID, a.Event, a.OS, a.Browser, a.Device, a.IPAddress, a.UserAgent)
	return err
}

// DeleteOlderThan drops rows past the retention window and reports how many it
// removed, so the caller can log something meaningful instead of a bare "done".
// The cutoff is computed in Go rather than as a SQL INTERVAL literal, keeping
// the retention period a single Go constant.
func (r *ActivityRepository) DeleteOlderThan(ctx context.Context, age time.Duration) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM user_activity_logs WHERE created_at < $1`, time.Now().Add(-age))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
