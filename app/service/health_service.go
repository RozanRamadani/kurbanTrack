package service

import (
	"context"
	"time"

	"kurbantrack-backend/helper"

	"github.com/gofiber/fiber/v2"
)

// DBPinger mendefinisikan interface sederhana untuk pengecekan koneksi database.
type DBPinger interface {
	Ping(ctx context.Context) error
}

type HealthService struct {
	db DBPinger
}

// NewHealthService membuat instance HealthService dengan dependency injection DBPinger.
func NewHealthService(db DBPinger) *HealthService {
	return &HealthService{
		db: db,
	}
}

// CheckHealth menangani endpoint healthcheck HTTP untuk memeriksa status server dan database.
func (s *HealthService) CheckHealth(c *fiber.Ctx) error {
	dbStatus := "connected"
	var dbErrDetail string

	if s.db == nil {
		dbStatus = "uninitialized"
		dbErrDetail = "PostgreSQL pool is not initialized"
	} else {
		ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
		defer cancel()

		if err := s.db.Ping(ctx); err != nil {
			dbStatus = "disconnected"
			dbErrDetail = err.Error()
		}
	}

	healthData := fiber.Map{
		"server":   "healthy",
		"database": dbStatus,
	}

	if dbErrDetail != "" {
		healthData["database_error"] = dbErrDetail
		return c.Status(fiber.StatusServiceUnavailable).JSON(helper.Response{
			Success: false,
			Message: "Service partially unavailable",
			Data:    healthData,
			Error:   dbErrDetail,
		})
	}

	return c.Status(fiber.StatusOK).JSON(
		helper.SuccessResponse("Service is healthy", healthData),
	)
}
