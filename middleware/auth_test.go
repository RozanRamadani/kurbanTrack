package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"kurbantrack-backend/app/model"
	"kurbantrack-backend/helper"
	"kurbantrack-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

func setupTestApp() *fiber.App {
	app := fiber.New()

	// Simulasi rute yang hanya boleh diakses oleh Admin
	app.Post("/api/v1/animals",
		middleware.RequireAuth(),
		middleware.RequireRole(model.RoleAdmin),
		func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusCreated).JSON(fiber.Map{"status": "created"})
		},
	)

	return app
}

func TestAuthMiddleware_NoToken_Returns401(t *testing.T) {
	app := setupTestApp()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/animals", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("gagal menjalankan request: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status code = %d, seharusnya 401 Unauthorized", resp.StatusCode)
	}
}

func TestRoleMiddleware_ForbiddenRole_Returns403(t *testing.T) {
	app := setupTestApp()

	// Generate token dengan role panitia_distribusi (bukan admin)
	token, err := helper.GenerateToken(2, "panitia@kurbantrack.local", model.RolePanitiaDistribusi, nil)
	if err != nil {
		t.Fatalf("gagal generate token: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/animals", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("gagal menjalankan request: %v", err)
	}

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status code = %d, seharusnya 403 Forbidden", resp.StatusCode)
	}
}

func TestRoleMiddleware_AllowedAdmin_Returns201(t *testing.T) {
	app := setupTestApp()

	// Generate token dengan role admin
	token, err := helper.GenerateToken(1, "admin@kurbantrack.local", model.RoleAdmin, nil)
	if err != nil {
		t.Fatalf("gagal generate token: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/animals", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("gagal menjalankan request: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status code = %d, seharusnya 201 Created", resp.StatusCode)
	}
}
