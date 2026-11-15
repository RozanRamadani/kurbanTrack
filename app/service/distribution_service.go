package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"kurbantrack-backend/app/model"
	"kurbantrack-backend/app/repository"
	"kurbantrack-backend/helper"

	"github.com/gofiber/fiber/v2"
)

// DistributionRepository mendefinisikan kontrak repository distribusi beserta operasi
// transaksional atomik yang disepakati dengan layer repository.
type DistributionRepository interface {
	repository.DistributionRepository
	AllocateWithQuotaTx(ctx context.Context, dist *model.Distribution) error
	RedeemCouponTx(ctx context.Context, couponCode string, redeemedAt time.Time, actorID int) (*model.Distribution, error)
	CancelCouponTx(ctx context.Context, couponCode string, actorID int) (*model.Distribution, error)
}

// DistributionService mendefinisikan interface bisnis dan HTTP-facing untuk pengelolaan distribusi kupon.
type DistributionService interface {
	// Business methods (dapat diuji tanpa ketergantungan Fiber)
	CreateDistribution(ctx context.Context, req *model.DistributionCreateRequest) (*model.Distribution, error)
	GetAllDistributions(ctx context.Context) ([]model.Distribution, error)
	RedeemDistribution(ctx context.Context, couponCode string, actorID int) (*model.Distribution, error)
	CancelDistribution(ctx context.Context, couponCode string, actorID int) (*model.Distribution, error)

	// Fiber HTTP-facing methods (sesuai Modul 4)
	Create(c *fiber.Ctx) error
	FindAll(c *fiber.Ctx) error
	Redeem(c *fiber.Ctx) error
	Cancel(c *fiber.Ctx) error
}

type distributionService struct {
	repo       DistributionRepository
	animalRepo repository.AnimalRepository
}

// NewDistributionService menginisialisasi DistributionService dengan dependency injection
// DistributionRepository dan AnimalRepository secara eksplisit.
func NewDistributionService(
	repo DistributionRepository,
	animalRepo repository.AnimalRepository,
) DistributionService {
	return &distributionService{
		repo:       repo,
		animalRepo: animalRepo,
	}
}

// generateCouponCode menghasilkan kode kupon acak unik menggunakan crypto/rand dengan panjang <= 30 karakter.
func generateCouponCode() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "KBN-" + strings.ToUpper(hex.EncodeToString(b)), nil
}

// ----------------------------------------------------------------------------
// BUSINESS METHODS
// ----------------------------------------------------------------------------

func (s *distributionService) CreateDistribution(ctx context.Context, req *model.DistributionCreateRequest) (*model.Distribution, error) {
	// 1. Validasi aturan bisnis pada input request
	if err := ValidateDistributionCreate(req); err != nil {
		return nil, err
	}

	// 2. Periksa keberadaan hewan qurban
	animal, err := s.animalRepo.FindByID(ctx, req.AnimalID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// 3. Validasi awal status hewan ('slaughtered') dan ketersediaan kuota paket
	if err := ValidateAnimalForDistribution(animal); err != nil {
		return nil, err
	}

	// 4. Buat kode kupon acak menggunakan crypto/rand (maksimal 30 karakter)
	couponCode, err := generateCouponCode()
	if err != nil {
		return nil, err
	}

	// 5. Siapkan entitas model Distribution
	dist := &model.Distribution{
		CouponCode:    couponCode,
		AnimalID:      req.AnimalID,
		CoordinatorID: req.CoordinatorID,
		RecipientNIK:  strings.TrimSpace(req.RecipientNIK),
		RecipientName: strings.TrimSpace(req.RecipientName),
		RecipientRT:   strings.TrimSpace(req.RecipientRT),
		Status:        model.DistributionStatusAllocated,
	}

	// 6. Eksekusi alokasi kupon, penambahan allocated_packages, dan log secara transaksional
	if err := s.repo.AllocateWithQuotaTx(ctx, dist); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrConflict
		}
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		if errors.Is(err, repository.ErrForeignKeyViolation) {
			return nil, ErrForeignKeyViolation
		}
		return nil, err
	}

	return dist, nil
}

func (s *distributionService) GetAllDistributions(ctx context.Context) ([]model.Distribution, error) {
	return s.repo.FindAll(ctx)
}

func (s *distributionService) RedeemDistribution(ctx context.Context, couponCode string, actorID int) (*model.Distribution, error) {
	// 1. Validasi kode kupon dan actor ID
	if err := ValidateCouponCode(couponCode); err != nil {
		return nil, err
	}
	if err := ValidateActorID(actorID); err != nil {
		return nil, err
	}

	code := strings.TrimSpace(couponCode)

	// 2. Pastikan kupon distribusi ditemukan
	existing, err := s.repo.FindByCouponCode(ctx, code)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// 3. Validasi status kupon (hanya 'allocated' yang dapat ditebus)
	if err := ValidateDistributionRedeem(existing); err != nil {
		return nil, err
	}

	// 4. Eksekusi penebusan kupon dan pencatatan distribution_logs secara transaksional
	redeemedAt := time.Now()
	updated, err := s.repo.RedeemCouponTx(ctx, code, redeemedAt, actorID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrConflict
		}
		return nil, err
	}

	return updated, nil
}

func (s *distributionService) CancelDistribution(ctx context.Context, couponCode string, actorID int) (*model.Distribution, error) {
	// 1. Validasi kode kupon dan actor ID
	if err := ValidateCouponCode(couponCode); err != nil {
		return nil, err
	}
	if err := ValidateActorID(actorID); err != nil {
		return nil, err
	}

	code := strings.TrimSpace(couponCode)

	// 2. Pastikan kupon distribusi ditemukan
	existing, err := s.repo.FindByCouponCode(ctx, code)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// 3. Validasi status kupon (hanya 'allocated' yang dapat dibatalkan)
	if err := ValidateDistributionCancel(existing); err != nil {
		return nil, err
	}

	// 4. Eksekusi pembatalan kupon, pengembalian allocated_packages, dan log secara transaksional
	updated, err := s.repo.CancelCouponTx(ctx, code, actorID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrConflict
		}
		return nil, err
	}

	return updated, nil
}

// ----------------------------------------------------------------------------
// FIBER HTTP-FACING METHODS (Modul 4)
// ----------------------------------------------------------------------------

// Create menangani endpoint POST /api/v1/distributions.
func (s *distributionService) Create(c *fiber.Ctx) error {
	actorID, ok := c.Locals("user_id").(int)
	if !ok || actorID <= 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(
			helper.ErrorResponse("Identitas pengguna belum terautentikasi", nil),
		)
	}

	var req model.DistributionCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			helper.ErrorResponse("Format request body tidak valid", err.Error()),
		)
	}

	// Gunakan user_id dari konteks autentikasi sebagai CoordinatorID (jangan percaya input bebas dari body)
	req.CoordinatorID = actorID

	dist, err := s.CreateDistribution(c.Context(), &req)
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
				helper.ErrorResponse("NIK penerima atau kode kupon sudah terdaftar", nil),
			)
		}
		if errors.Is(err, ErrQuotaExceeded) {
			return c.Status(fiber.StatusConflict).JSON(
				helper.ErrorResponse(ErrQuotaExceeded.Error(), nil),
			)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(
			helper.ErrorResponse("Terjadi kesalahan internal pada server", nil),
		)
	}

	return c.Status(fiber.StatusCreated).JSON(
		helper.SuccessResponse("Alokasi kupon distribusi berhasil dibuat", dist),
	)
}

// FindAll menangani endpoint GET /api/v1/distributions.
func (s *distributionService) FindAll(c *fiber.Ctx) error {
	dists, err := s.GetAllDistributions(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			helper.ErrorResponse("Terjadi kesalahan internal pada server", nil),
		)
	}

	return c.Status(fiber.StatusOK).JSON(
		helper.SuccessResponse("Daftar distribusi berhasil diambil", dists),
	)
}

// Redeem menangani endpoint PATCH /api/v1/distributions/:code/redeem.
func (s *distributionService) Redeem(c *fiber.Ctx) error {
	code := c.Params("code")
	if strings.TrimSpace(code) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			helper.ErrorResponse("Kode kupon wajib diisi", nil),
		)
	}

	actorID, ok := c.Locals("user_id").(int)
	if !ok || actorID <= 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(
			helper.ErrorResponse("Identitas pengguna belum terautentikasi", nil),
		)
	}

	dist, err := s.RedeemDistribution(c.Context(), code, actorID)
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
				helper.ErrorResponse("Data kupon distribusi tidak ditemukan", nil),
			)
		}
		if errors.Is(err, ErrConflict) || errors.Is(err, ErrInvalidCouponStatus) {
			return c.Status(fiber.StatusConflict).JSON(
				helper.ErrorResponse("Kupon tidak dapat ditebus karena konflik status", nil),
			)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(
			helper.ErrorResponse("Terjadi kesalahan internal pada server", nil),
		)
	}

	return c.Status(fiber.StatusOK).JSON(
		helper.SuccessResponse("Kupon distribusi berhasil ditebus", dist),
	)
}

// Cancel menangani endpoint PATCH /api/v1/distributions/:code/cancel.
func (s *distributionService) Cancel(c *fiber.Ctx) error {
	code := c.Params("code")
	if strings.TrimSpace(code) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			helper.ErrorResponse("Kode kupon wajib diisi", nil),
		)
	}

	actorID, ok := c.Locals("user_id").(int)
	if !ok || actorID <= 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(
			helper.ErrorResponse("Identitas pengguna belum terautentikasi", nil),
		)
	}

	dist, err := s.CancelDistribution(c.Context(), code, actorID)
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
				helper.ErrorResponse("Data kupon distribusi tidak ditemukan", nil),
			)
		}
		if errors.Is(err, ErrConflict) || errors.Is(err, ErrInvalidCouponStatus) {
			return c.Status(fiber.StatusConflict).JSON(
				helper.ErrorResponse("Kupon tidak dapat dibatalkan karena konflik status", nil),
			)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(
			helper.ErrorResponse("Terjadi kesalahan internal pada server", nil),
		)
	}

	return c.Status(fiber.StatusOK).JSON(
		helper.SuccessResponse("Kupon distribusi berhasil dibatalkan", dist),
	)
}
