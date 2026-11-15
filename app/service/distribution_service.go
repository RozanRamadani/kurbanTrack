package service

import (
	"kurbantrack-backend/app/repository"
	"kurbantrack-backend/helper"

	"github.com/gofiber/fiber/v2"
)

// DistributionService mendefinisikan interface untuk endpoint scaffold fitur distribusi.
type DistributionService interface {
	Create(c *fiber.Ctx) error
	FindAll(c *fiber.Ctx) error
	Redeem(c *fiber.Ctx) error
	Cancel(c *fiber.Ctx) error
}

type distributionService struct {
	repo repository.DistributionRepository
}

// NewDistributionService menginisialisasi DistributionService dengan dependency injection DistributionRepository.
func NewDistributionService(repo repository.DistributionRepository) DistributionService {
	return &distributionService{repo: repo}
}

// Create menangani endpoint POST /api/v1/distributions (scaffold only).
func (s *distributionService) Create(c *fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(helper.Response{
		Success: false,
		Message: "Endpoint scaffold: Alokasi distribusi kupon belum diimplementasikan secara penuh",
		Data: fiber.Map{
			"endpoint": "POST /api/v1/distributions",
			"status":   "scaffold_only",
		},
	})
}

// FindAll menangani endpoint GET /api/v1/distributions.
func (s *distributionService) FindAll(c *fiber.Ctx) error {
	dists, err := s.repo.FindAll(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			helper.ErrorResponse("Terjadi kesalahan internal pada server", nil),
		)
	}

	return c.Status(fiber.StatusOK).JSON(helper.Response{
		Success: true,
		Message: "Daftar distribusi berhasil diambil (scaffold)",
		Data: fiber.Map{
			"distributions": dists,
			"status":        "scaffold_only",
		},
	})
}

// Redeem menangani endpoint PATCH /api/v1/distributions/:code/redeem (scaffold only).
func (s *distributionService) Redeem(c *fiber.Ctx) error {
	code := c.Params("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			helper.ErrorResponse("Kode kupon wajib diisi", nil),
		)
	}

	return c.Status(fiber.StatusNotImplemented).JSON(helper.Response{
		Success: false,
		Message: "Endpoint scaffold: Penebusan kupon distribusi belum diimplementasikan secara penuh",
		Data: fiber.Map{
			"coupon_code": code,
			"endpoint":    "PATCH /api/v1/distributions/:code/redeem",
			"status":      "scaffold_only",
		},
	})
}

// Cancel menangani endpoint PATCH /api/v1/distributions/:code/cancel (scaffold only).
func (s *distributionService) Cancel(c *fiber.Ctx) error {
	code := c.Params("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			helper.ErrorResponse("Kode kupon wajib diisi", nil),
		)
	}

	return c.Status(fiber.StatusNotImplemented).JSON(helper.Response{
		Success: false,
		Message: "Endpoint scaffold: Pembatalan kupon distribusi belum diimplementasikan secara penuh",
		Data: fiber.Map{
			"coupon_code": code,
			"endpoint":    "PATCH /api/v1/distributions/:code/cancel",
			"status":      "scaffold_only",
		},
	})
}
