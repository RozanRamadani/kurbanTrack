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

	// Atomic transactional operations
	AllocateCouponTx(ctx context.Context, dist *model.Distribution, actorID int, note string) error
	RedeemCouponTx(ctx context.Context, couponCode string, actorID int, note string) (*model.Distribution, error)
	CancelCouponTx(ctx context.Context, couponCode string, actorID int, note string) (*model.Distribution, error)
}

type distributionRepository struct {
	db *pgxpool.Pool
}

// NewDistributionRepository menginisialisasi repository Distribution dengan connection pool pgxpool.
func NewDistributionRepository(db *pgxpool.Pool) DistributionRepository {
	return &distributionRepository{db: db}
}

func mapPgError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return ErrConflict
		case "23503": // foreign_key_violation
			return ErrForeignKeyViolation
		case "23514": // check_violation
			return ErrConstraintViolation
		}
	}
	return err
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

func (r *distributionRepository) AllocateCouponTx(ctx context.Context, dist *model.Distribution, actorID int, note string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 1. Kunci baris hewan (row-level lock) dan validasi sisa kuota paket
	var estimatedPackages, allocatedPackages int
	lockAnimalQuery := `
		SELECT estimated_packages, allocated_packages
		FROM animals
		WHERE id = $1
		FOR UPDATE
	`
	if err := tx.QueryRow(ctx, lockAnimalQuery, dist.AnimalID).Scan(&estimatedPackages, &allocatedPackages); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	if allocatedPackages >= estimatedPackages {
		return ErrQuotaExceeded
	}

	// 2. Tambah counter allocated_packages pada tabel animals
	updateAnimalQuery := `
		UPDATE animals
		SET allocated_packages = allocated_packages + 1
		WHERE id = $1
	`
	if _, err := tx.Exec(ctx, updateAnimalQuery, dist.AnimalID); err != nil {
		return mapPgError(err)
	}

	// 3. Simpan kupon baru ke tabel distributions
	if dist.Status == "" {
		dist.Status = model.DistributionStatusAllocated
	}

	insertDistQuery := `
		INSERT INTO distributions (coupon_code, animal_id, coordinator_id, recipient_nik, recipient_name, recipient_rt, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`
	err = tx.QueryRow(ctx, insertDistQuery,
		dist.CouponCode,
		dist.AnimalID,
		dist.CoordinatorID,
		dist.RecipientNIK,
		dist.RecipientName,
		dist.RecipientRT,
		dist.Status,
	).Scan(&dist.ID, &dist.CreatedAt)
	if err != nil {
		return mapPgError(err)
	}

	// 4. Catat jejak audit awal ke distribution_logs (from_status = NULL -> allocated)
	insertLogQuery := `
		INSERT INTO distribution_logs (distribution_id, actor_id, from_status, to_status, note)
		VALUES ($1, $2, NULL, $3, $4)
	`
	if _, err := tx.Exec(ctx, insertLogQuery, dist.ID, actorID, dist.Status, note); err != nil {
		return mapPgError(err)
	}

	return tx.Commit(ctx)
}

func (r *distributionRepository) RedeemCouponTx(ctx context.Context, couponCode string, actorID int, note string) (*model.Distribution, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 1. Kunci baris distribusi berdasarkan coupon_code
	selectQuery := `
		SELECT id, coupon_code, animal_id, coordinator_id, recipient_nik, recipient_name, recipient_rt, status, redeemed_at, created_at
		FROM distributions
		WHERE coupon_code = $1
		FOR UPDATE
	`
	var dist model.Distribution
	err = tx.QueryRow(ctx, selectQuery, couponCode).Scan(
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

	// Kupon hanya boleh di-redeem jika status saat ini adalah 'allocated'
	if dist.Status != model.DistributionStatusAllocated {
		return nil, ErrInvalidStatusTransition
	}

	// 2. Perbarui status kupon menjadi 'redeemed' beserta timestamp penebusan
	now := time.Now().UTC()
	prevStatus := dist.Status
	dist.Status = model.DistributionStatusRedeemed
	dist.RedeemedAt = &now

	updateQuery := `
		UPDATE distributions
		SET status = $1, redeemed_at = $2
		WHERE id = $3
	`
	if _, err := tx.Exec(ctx, updateQuery, dist.Status, dist.RedeemedAt, dist.ID); err != nil {
		return nil, mapPgError(err)
	}

	// 3. Catat audit trail perubahan status (allocated -> redeemed)
	insertLogQuery := `
		INSERT INTO distribution_logs (distribution_id, actor_id, from_status, to_status, note)
		VALUES ($1, $2, $3, $4, $5)
	`
	if _, err := tx.Exec(ctx, insertLogQuery, dist.ID, actorID, prevStatus, dist.Status, note); err != nil {
		return nil, mapPgError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &dist, nil
}

func (r *distributionRepository) CancelCouponTx(ctx context.Context, couponCode string, actorID int, note string) (*model.Distribution, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 1. Kunci baris distribusi berdasarkan coupon_code
	selectQuery := `
		SELECT id, coupon_code, animal_id, coordinator_id, recipient_nik, recipient_name, recipient_rt, status, redeemed_at, created_at
		FROM distributions
		WHERE coupon_code = $1
		FOR UPDATE
	`
	var dist model.Distribution
	err = tx.QueryRow(ctx, selectQuery, couponCode).Scan(
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

	// Hanya kupon berstatus 'allocated' yang dapat dibatalkan
	if dist.Status != model.DistributionStatusAllocated {
		return nil, ErrInvalidStatusTransition
	}

	// 2. Ubah status kupon menjadi 'cancelled'
	prevStatus := dist.Status
	dist.Status = model.DistributionStatusCancelled

	updateDistQuery := `
		UPDATE distributions
		SET status = $1
		WHERE id = $2
	`
	if _, err := tx.Exec(ctx, updateDistQuery, dist.Status, dist.ID); err != nil {
		return nil, mapPgError(err)
	}

	// 3. Kembalikan 1 kuota paket ke tabel animals
	releaseQuotaQuery := `
		UPDATE animals
		SET allocated_packages = allocated_packages - 1
		WHERE id = $1 AND allocated_packages > 0
	`
	cmdTag, err := tx.Exec(ctx, releaseQuotaQuery, dist.AnimalID)
	if err != nil {
		return nil, mapPgError(err)
	}
	if cmdTag.RowsAffected() == 0 {
		return nil, ErrConstraintViolation
	}

	// 4. Catat audit trail pembatalan (allocated -> cancelled)
	insertLogQuery := `
		INSERT INTO distribution_logs (distribution_id, actor_id, from_status, to_status, note)
		VALUES ($1, $2, $3, $4, $5)
	`
	if _, err := tx.Exec(ctx, insertLogQuery, dist.ID, actorID, prevStatus, dist.Status, note); err != nil {
		return nil, mapPgError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &dist, nil
}
