package route

import (
	"kurbantrack-backend/app/service"

	"github.com/gofiber/fiber/v2"
)

// SetupRoutes mendaftarkan seluruh endpoint aplikasi dan memetakannya ke method di layer service.
func SetupRoutes(
	app *fiber.App,
	healthService *service.HealthService,
	animalService service.AnimalService,
	distributionService service.DistributionService,
) {
	// Health check endpoint
	app.Get("/health", healthService.CheckHealth)

	// API v1 routes
	v1 := app.Group("/api/v1")
	{
		// Animals CRUD
		animals := v1.Group("/animals")
		animals.Post("/", animalService.Create)
		animals.Get("/", animalService.FindAll)
		animals.Get("/:id", animalService.FindByID)
		animals.Put("/:id", animalService.Update)
		animals.Delete("/:id", animalService.Delete)

		// Distribution Scaffold (sesuai spesifikasi SRS)
		distributions := v1.Group("/distributions")
		distributions.Post("/", distributionService.Create)
		distributions.Get("/", distributionService.FindAll)
		distributions.Patch("/:code/redeem", distributionService.Redeem)
		distributions.Patch("/:code/cancel", distributionService.Cancel)
	}
}
