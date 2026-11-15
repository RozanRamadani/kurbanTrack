package config

import (
	"kurbantrack-backend/app/service"
	"kurbantrack-backend/route"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func NewApp(
	healthService *service.HealthService,
	authService service.AuthService,
	animalService service.AnimalService,
	distributionService service.DistributionService,
) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName: "KurbanTrack Backend API v1",
	})

	// Middleware bawaan Fiber
	app.Use(recover.New())
	app.Use(logger.New())

	// Registrasi routes melalui dependency service
	route.SetupRoutes(app, healthService, authService, animalService, distributionService)

	return app
}
