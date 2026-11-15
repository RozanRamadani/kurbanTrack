package repository

import (
	"context"

	"kurbantrack-backend/app/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DistributionLogRepository mendefinisikan interface operasi database untuk entitas DistributionLog.
type DistributionLogRepository interface {
	Create(ctx context.Context, log *model.DistributionLog) error
	FindByDistributionID(ctx context.Context, distributionID int) ([]model.DistributionLog, error)
}

type distributionLogRepository struct {
	db *pgxpool.Pool
}

// NewDistributionLogRepository menginisialisasi repository DistributionLog dengan pgxpool.
func NewDistributionLogRepository(db *pgxpool.Pool) DistributionLogRepository {
	return &distributionLogRepository{db: db}
}

func (r *distributionLogRepository) Create(ctx context.Context, log *model.DistributionLog) error {
	query := `
		INSERT INTO distribution_logs (distribution_id, actor_id, from_status, to_status, note)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	return r.db.QueryRow(ctx, query,
		log.DistributionID,
		log.ActorID,
		log.FromStatus,
		log.ToStatus,
		log.Note,
	).Scan(&log.ID, &log.CreatedAt)
}

func (r *distributionLogRepository) FindByDistributionID(ctx context.Context, distributionID int) ([]model.DistributionLog, error) {
	query := `
		SELECT id, distribution_id, actor_id, from_status, to_status, note, created_at
		FROM distribution_logs
		WHERE distribution_id = $1
		ORDER BY id ASC
	`

	rows, err := r.db.Query(ctx, query, distributionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]model.DistributionLog, 0)
	for rows.Next() {
		var log model.DistributionLog
		if err := rows.Scan(
			&log.ID,
			&log.DistributionID,
			&log.ActorID,
			&log.FromStatus,
			&log.ToStatus,
			&log.Note,
			&log.CreatedAt,
		); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return logs, nil
}
