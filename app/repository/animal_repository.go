package repository

import (
	"context"
	"errors"

	"kurbantrack-backend/app/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AnimalRepository mendefinisikan interface operasi database untuk entitas Animal.
type AnimalRepository interface {
	Create(ctx context.Context, animal *model.Animal) error
	FindByID(ctx context.Context, id int) (*model.Animal, error)
	FindAll(ctx context.Context) ([]model.Animal, error)
	Update(ctx context.Context, animal *model.Animal) error
	Delete(ctx context.Context, id int) error
	CountAllocationsByAnimalID(ctx context.Context, animalID int) (int, error)
}

type animalRepository struct {
	db *pgxpool.Pool
}

// NewAnimalRepository menginisialisasi repository Animal dengan connection pool pgxpool.
func NewAnimalRepository(db *pgxpool.Pool) AnimalRepository {
	return &animalRepository{db: db}
}

func (r *animalRepository) Create(ctx context.Context, animal *model.Animal) error {
	query := `
		INSERT INTO animals (code, species, sohibul_name, weight_kg, estimated_packages, allocated_packages, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`

	if animal.Status == "" {
		animal.Status = model.AnimalStatusRegistered
	}

	err := r.db.QueryRow(ctx, query,
		animal.Code,
		animal.Species,
		animal.SohibulName,
		animal.WeightKg,
		animal.EstimatedPackages,
		animal.AllocatedPackages,
		animal.Status,
	).Scan(&animal.ID, &animal.CreatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrConflict
		}
		return err
	}

	return nil
}

func (r *animalRepository) FindByID(ctx context.Context, id int) (*model.Animal, error) {
	query := `
		SELECT id, code, species, sohibul_name, weight_kg, estimated_packages, allocated_packages, status, created_at
		FROM animals
		WHERE id = $1
	`

	var animal model.Animal
	err := r.db.QueryRow(ctx, query, id).Scan(
		&animal.ID,
		&animal.Code,
		&animal.Species,
		&animal.SohibulName,
		&animal.WeightKg,
		&animal.EstimatedPackages,
		&animal.AllocatedPackages,
		&animal.Status,
		&animal.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &animal, nil
}

func (r *animalRepository) FindAll(ctx context.Context) ([]model.Animal, error) {
	query := `
		SELECT id, code, species, sohibul_name, weight_kg, estimated_packages, allocated_packages, status, created_at
		FROM animals
		ORDER BY id ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	animals := make([]model.Animal, 0)
	for rows.Next() {
		var animal model.Animal
		if err := rows.Scan(
			&animal.ID,
			&animal.Code,
			&animal.Species,
			&animal.SohibulName,
			&animal.WeightKg,
			&animal.EstimatedPackages,
			&animal.AllocatedPackages,
			&animal.Status,
			&animal.CreatedAt,
		); err != nil {
			return nil, err
		}
		animals = append(animals, animal)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return animals, nil
}

func (r *animalRepository) Update(ctx context.Context, animal *model.Animal) error {
	query := `
		UPDATE animals
		SET code = $1, species = $2, sohibul_name = $3, weight_kg = $4,
		    estimated_packages = $5, allocated_packages = $6, status = $7
		WHERE id = $8
		RETURNING created_at
	`

	err := r.db.QueryRow(ctx, query,
		animal.Code,
		animal.Species,
		animal.SohibulName,
		animal.WeightKg,
		animal.EstimatedPackages,
		animal.AllocatedPackages,
		animal.Status,
		animal.ID,
	).Scan(&animal.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrConflict
		}
		return err
	}

	return nil
}

func (r *animalRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM animals WHERE id = $1`

	cmdTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return ErrForeignKeyViolation
		}
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// CountAllocationsByAnimalID menghitung jumlah alokasi kupon berstatus aktif ('allocated')
// untuk hewan tertentu sesuai aturan bisnis: hewan tidak boleh dihapus jika masih memiliki alokasi aktif.
func (r *animalRepository) CountAllocationsByAnimalID(ctx context.Context, animalID int) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM distributions
		WHERE animal_id = $1 AND status = 'allocated'
	`

	var count int
	err := r.db.QueryRow(ctx, query, animalID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}
