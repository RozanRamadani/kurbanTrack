package route

import (
	"kurbantrack-backend/app/model"
	"kurbantrack-backend/app/service"
	"kurbantrack-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupRoutes mendaftarkan seluruh endpoint aplikasi, middleware autentikasi/otorisasi, dan service.
func SetupRoutes(
	app *fiber.App,
	healthService *service.HealthService,
	authService service.AuthService,
	animalService service.AnimalService,
	distributionService service.DistributionService,
) {
	// Health check endpoint (Publik)
	app.Get("/health", healthService.CheckHealth)

	// API v1 routes
	v1 := app.Group("/api/v1")
	{
		// Auth endpoints (Publik)
		auth := v1.Group("/auth")
		auth.Post("/register", authService.Register)
		auth.Post("/login", authService.Login)

		// Animals CRUD (Wajib Login, mutasi data khusus Admin)
		animals := v1.Group("/animals", middleware.RequireAuth())
		animals.Post("/", middleware.RequireRole(model.RoleAdmin), animalService.Create)
		animals.Get("/", animalService.FindAll)
		animals.Get("/:id", animalService.FindByID)
		animals.Put("/:id", middleware.RequireRole(model.RoleAdmin), animalService.Update)
		animals.Delete("/:id", middleware.RequireRole(model.RoleAdmin), animalService.Delete)

		// Distribution endpoints (Wajib Login, otorisasi sesuai role SRS)
		distributions := v1.Group("/distributions", middleware.RequireAuth())
		distributions.Post("/", middleware.RequireRole(model.RoleKoordinatorWilayah), distributionService.Create)
		distributions.Get("/", middleware.RequireRole(model.RoleAdmin, model.RoleKoordinatorWilayah, model.RolePanitiaDistribusi), distributionService.FindAll)
		distributions.Patch("/:code/redeem", middleware.RequireRole(model.RolePanitiaDistribusi), distributionService.Redeem)
		distributions.Patch("/:code/cancel", middleware.RequireRole(model.RoleAdmin, model.RoleKoordinatorWilayah), distributionService.Cancel)
	}
}
