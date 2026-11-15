package service

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"

	"kurbantrack-backend/app/model"
	"kurbantrack-backend/app/repository"
	"kurbantrack-backend/helper"

	"github.com/gofiber/fiber/v2"
)

// AnimalService mendefinisikan interface bisnis dan HTTP-facing untuk pengelolaan entitas Animal.
type AnimalService interface {
	// Business methods (dapat diuji tanpa ketergantungan Fiber)
	CreateAnimal(ctx context.Context, req *model.AnimalCreateRequest) (*model.Animal, error)
	GetAnimalByID(ctx context.Context, id int) (*model.Animal, error)
	GetAllAnimals(ctx context.Context) ([]model.Animal, error)
	UpdateAnimal(ctx context.Context, id int, req *model.AnimalUpdateRequest) (*model.Animal, error)
	DeleteAnimal(ctx context.Context, id int) error

	// Fiber HTTP-facing methods (sesuai Modul 4)
	Create(c *fiber.Ctx) error
	FindAll(c *fiber.Ctx) error
	FindByID(c *fiber.Ctx) error
	Update(c *fiber.Ctx) error
	Delete(c *fiber.Ctx) error
}

type animalService struct {
	repo repository.AnimalRepository
}

// NewAnimalService menginisialisasi AnimalService dengan dependency injection AnimalRepository.
func NewAnimalService(repo repository.AnimalRepository) AnimalService {
	return &animalService{repo: repo}
}

// ----------------------------------------------------------------------------
// BUSINESS METHODS
// ----------------------------------------------------------------------------

func (s *animalService) CreateAnimal(ctx context.Context, req *model.AnimalCreateRequest) (*model.Animal, error) {
	// 1. Validasi aturan bisnis
	if err := ValidateAnimalCreate(req); err != nil {
		return nil, err
	}

	// 2. Siapkan entitas model
	animal := &model.Animal{
		Code:              strings.TrimSpace(req.Code),
		Species:           req.Species,
		SohibulName:       strings.TrimSpace(req.SohibulName),
		WeightKg:          math.Round(req.WeightKg*100) / 100,
		EstimatedPackages: req.EstimatedPackages,
		AllocatedPackages: 0,
		Status:            model.AnimalStatusRegistered,
	}

	// 3. Simpan ke database via repository
	err := s.repo.Create(ctx, animal)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrConflict
		}
		return nil, err
	}

	return animal, nil
}

func (s *animalService) GetAnimalByID(ctx context.Context, id int) (*model.Animal, error) {
	animal, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return animal, nil
}

func (s *animalService) GetAllAnimals(ctx context.Context) ([]model.Animal, error) {
	return s.repo.FindAll(ctx)
}

func (s *animalService) UpdateAnimal(ctx context.Context, id int, req *model.AnimalUpdateRequest) (*model.Animal, error) {
	// 1. Pastikan hewan ada
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// 2. Periksa jumlah alokasi aktif yang sudah ada
	activeAllocations, err := s.repo.CountAllocationsByAnimalID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 3. Validasi aturan bisnis pembaruan
	if err := ValidateAnimalUpdate(req, existing.AllocatedPackages, activeAllocations); err != nil {
		return nil, err
	}

	// 4. Update field (ID, CreatedAt, dan AllocatedPackages tidak boleh diubah sembarangan via request)
	existing.Code = strings.TrimSpace(req.Code)
	existing.Species = req.Species
	existing.SohibulName = strings.TrimSpace(req.SohibulName)
	existing.WeightKg = math.Round(req.WeightKg*100) / 100
	existing.EstimatedPackages = req.EstimatedPackages
	if req.Status != "" {
		existing.Status = req.Status
	}

	// 5. Simpan pembaruan via repository
	err = s.repo.Update(ctx, existing)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrConflict
		}
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return existing, nil
}

func (s *animalService) DeleteAnimal(ctx context.Context, id int) error {
	// 1. Pastikan hewan ada
	_, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}

	// 2. Validasi aturan bisnis: tidak boleh dihapus jika masih ada alokasi aktif
	activeCount, err := s.repo.CountAllocationsByAnimalID(ctx, id)
	if err != nil {
		return err
	}
	if activeCount > 0 {
		return ErrHasActiveAllocations
	}

	// 3. Eksekusi penghapusan di repository
	err = s.repo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		if errors.Is(err, repository.ErrForeignKeyViolation) {
			return ErrForeignKeyViolation
		}
		return err
	}

	return nil
}

// ----------------------------------------------------------------------------
// FIBER HTTP-FACING METHODS (Modul 4)
// ----------------------------------------------------------------------------

func (s *animalService) Create(c *fiber.Ctx) error {
	var req model.AnimalCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			helper.ErrorResponse("Format request body tidak valid", err.Error()),
		)
	}

	animal, err := s.CreateAnimal(c.Context(), &req)
	if err != nil {
		var valErr *ValidationError
		if errors.As(err, &valErr) {
			return c.Status(fiber.StatusBadRequest).JSON(
				helper.ErrorResponse(valErr.Message, fiber.Map{
					"field":  valErr.Field,
					"detail": valErr.Message,
				}),
			)
		}
		if errors.Is(err, ErrConflict) {
			return c.Status(fiber.StatusConflict).JSON(
				helper.ErrorResponse("Kode hewan sudah terdaftar", nil),
			)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(
			helper.ErrorResponse("Terjadi kesalahan internal pada server", nil),
		)
	}

	return c.Status(fiber.StatusCreated).JSON(
		helper.SuccessResponse("Data hewan qurban berhasil ditambahkan", animal),
	)
}

func (s *animalService) FindAll(c *fiber.Ctx) error {
	animals, err := s.GetAllAnimals(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			helper.ErrorResponse("Terjadi kesalahan internal pada server", nil),
		)
	}

	return c.Status(fiber.StatusOK).JSON(
		helper.SuccessResponse("Daftar hewan qurban berhasil diambil", animals),
	)
}

func (s *animalService) FindByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(
			helper.ErrorResponse("ID hewan tidak valid", "ID harus berupa integer positif"),
		)
	}

	animal, err := s.GetAnimalByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(
				helper.ErrorResponse("Data hewan qurban tidak ditemukan", nil),
			)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(
			helper.ErrorResponse("Terjadi kesalahan internal pada server", nil),
		)
	}

	return c.Status(fiber.StatusOK).JSON(
		helper.SuccessResponse("Detail hewan qurban berhasil diambil", animal),
	)
}

func (s *animalService) Update(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(
			helper.ErrorResponse("ID hewan tidak valid", "ID harus berupa integer positif"),
		)
	}

	var req model.AnimalUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			helper.ErrorResponse("Format request body tidak valid", err.Error()),
		)
	}

	updated, err := s.UpdateAnimal(c.Context(), id, &req)
	if err != nil {
		var valErr *ValidationError
		if errors.As(err, &valErr) {
			return c.Status(fiber.StatusBadRequest).JSON(
				helper.ErrorResponse(valErr.Message, fiber.Map{
					"field":  valErr.Field,
					"detail": valErr.Message,
				}),
			)
		}
		if errors.Is(err, ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(
				helper.ErrorResponse("Data hewan qurban tidak ditemukan", nil),
			)
		}
		if errors.Is(err, ErrConflict) {
			return c.Status(fiber.StatusConflict).JSON(
				helper.ErrorResponse("Kode hewan sudah terdaftar", nil),
			)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(
			helper.ErrorResponse("Terjadi kesalahan internal pada server", nil),
		)
	}

	return c.Status(fiber.StatusOK).JSON(
		helper.SuccessResponse("Data hewan qurban berhasil diperbarui", updated),
	)
}

func (s *animalService) Delete(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(
			helper.ErrorResponse("ID hewan tidak valid", "ID harus berupa integer positif"),
		)
	}

	err = s.DeleteAnimal(c.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(
				helper.ErrorResponse("Data hewan qurban tidak ditemukan", nil),
			)
		}
		if errors.Is(err, ErrHasActiveAllocations) {
			return c.Status(fiber.StatusConflict).JSON(
				helper.ErrorResponse("Hewan tidak dapat dihapus karena masih memiliki alokasi aktif", nil),
			)
		}
		if errors.Is(err, ErrForeignKeyViolation) {
			return c.Status(fiber.StatusConflict).JSON(
				helper.ErrorResponse("Hewan tidak dapat dihapus karena masih memiliki data riwayat terkait", nil),
			)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(
			helper.ErrorResponse("Terjadi kesalahan internal pada server", nil),
		)
	}

	return c.Status(fiber.StatusOK).JSON(
		helper.SuccessResponse("Data hewan qurban berhasil dihapus", nil),
	)
}
