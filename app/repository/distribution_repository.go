package repository

import (
	"context"
	"errors"
	"time"

	"kurbantrack-backend/app/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DistributionRepository mendefinisikan interface operasi database untuk entitas Distribution.
type DistributionRepository interface {
	Create(ctx context.Context, dist *model.Distribution) error
	FindByID(ctx context.Context, id int) (*model.Distribution, error)
	FindByCouponCode(ctx context.Context, couponCode string) (*model.Distribution, error)
	FindAll(ctx context.Context) ([]model.Distribution, error)
	UpdateStatus(ctx context.Context, id int, status string, redeemedAt *time.Time) error
}

type distributionRepository struct {
	db *pgxpool.Pool
}

// NewDistributionRepository menginisialisasi repository Distribution dengan connection pool pgxpool.
func NewDistributionRepository(db *pgxpool.Pool) DistributionRepository {
	return &distributionRepository{db: db}
}

func (r *distributionRepository) Create(ctx context.Context, dist *model.Distribution) error {
	query := `
		INSERT INTO distributions (coupon_code, animal_id, coordinator_id, recipient_nik, recipient_name, recipient_rt, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`

	if dist.Status == "" {
		dist.Status = model.DistributionStatusAllocated
	}

	err := r.db.QueryRow(ctx, query,
		dist.CouponCode,
		dist.AnimalID,
		dist.CoordinatorID,
		dist.RecipientNIK,
		dist.RecipientName,
		dist.RecipientRT,
		dist.Status,
	).Scan(&dist.ID, &dist.CreatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrConflict
		}
		return err
	}

	return nil
}

func (r *distributionRepository) FindByID(ctx context.Context, id int) (*model.Distribution, error) {
	query := `
		SELECT id, coupon_code, animal_id, coordinator_id, recipient_nik, recipient_name, recipient_rt, status, redeemed_at, created_at
		FROM distributions
		WHERE id = $1
	`

	var dist model.Distribution
	err := r.db.QueryRow(ctx, query, id).Scan(
		&dist.ID,
		&dist.CouponCode,
		&dist.AnimalID,
		&dist.CoordinatorID,
		&dist.RecipientNIK,
		&dist.RecipientName,
		&dist.RecipientRT,
		&dist.Status,
		&dist.RedeemedAt,
		&dist.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &dist, nil
}

func (r *distributionRepository) FindByCouponCode(ctx context.Context, couponCode string) (*model.Distribution, error) {
	query := `
		SELECT id, coupon_code, animal_id, coordinator_id, recipient_nik, recipient_name, recipient_rt, status, redeemed_at, created_at
		FROM distributions
		WHERE coupon_code = $1
	`

	var dist model.Distribution
	err := r.db.QueryRow(ctx, query, couponCode).Scan(
		&dist.ID,
		&dist.CouponCode,
		&dist.AnimalID,
		&dist.CoordinatorID,
		&dist.RecipientNIK,
		&dist.RecipientName,
		&dist.RecipientRT,
		&dist.Status,
		&dist.RedeemedAt,
		&dist.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &dist, nil
}

func (r *distributionRepository) FindAll(ctx context.Context) ([]model.Distribution, error) {
	query := `
		SELECT id, coupon_code, animal_id, coordinator_id, recipient_nik, recipient_name, recipient_rt, status, redeemed_at, created_at
		FROM distributions
		ORDER BY id ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dists := make([]model.Distribution, 0)
	for rows.Next() {
		var dist model.Distribution
		if err := rows.Scan(
			&dist.ID,
			&dist.CouponCode,
			&dist.AnimalID,
			&dist.CoordinatorID,
			&dist.RecipientNIK,
			&dist.RecipientName,
			&dist.RecipientRT,
			&dist.Status,
			&dist.RedeemedAt,
			&dist.CreatedAt,
		); err != nil {
			return nil, err
		}
		dists = append(dists, dist)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return dists, nil
}

func (r *distributionRepository) UpdateStatus(ctx context.Context, id int, status string, redeemedAt *time.Time) error {
	query := `
		UPDATE distributions
		SET status = $1, redeemed_at = $2
		WHERE id = $3
	`

	cmdTag, err := r.db.Exec(ctx, query, status, redeemedAt, id)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}
