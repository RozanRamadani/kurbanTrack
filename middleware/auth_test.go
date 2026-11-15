package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"kurbantrack-backend/app/model"
	"kurbantrack-backend/helper"
	"kurbantrack-backend/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

const testJWTSecret = "test-jwt-secret-kurbantrack-key"

func setupTestApp() *fiber.App {
	app := fiber.New()

	// Rute 1: Hanya boleh diakses oleh Admin
	app.Post("/api/v1/animals",
		middleware.RequireAuth(),
		middleware.RequireRole(model.RoleAdmin),
		func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusCreated).JSON(helper.SuccessResponse("hewan berhasil dibuat", nil))
		},
	)

	// Rute 2: Boleh diakses oleh Admin atau Koordinator Wilayah
	app.Get("/api/v1/distributions",
		middleware.RequireAuth(),
		middleware.RequireRole(model.RoleAdmin, model.RoleKoordinatorWilayah),
		func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusOK).JSON(helper.SuccessResponse("daftar distribusi", nil))
		},
	)

	return app
}

func parseErrorResponse(t *testing.T, resp *http.Response) helper.Response {
	t.Helper()
	var res helper.Response
	err := json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		t.Fatalf("gagal mendecode response body: %v", err)
	}
	return res
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

	body := parseErrorResponse(t, resp)
	if body.Success {
		t.Error("body.Success seharusnya false pada error response")
	}
	if body.Message != "header Authorization wajib disertakan" {
		t.Errorf("pesan error tidak sesuai: %s", body.Message)
	}
}

func TestAuthMiddleware_MalformedHeader_Returns401(t *testing.T) {
	app := setupTestApp()

	malformedHeaders := []string{
		"Basic abcdef12345",
		"Bearer",
		"Bearer ",
		"Token abcde.token.here",
		"Bearer token-part-1 token-part-2",
	}

	for _, headerVal := range malformedHeaders {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/animals", nil)
		req.Header.Set("Authorization", headerVal)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("gagal request dengan header %q: %v", headerVal, err)
		}

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("header %q: status code = %d, seharusnya 401", headerVal, resp.StatusCode)
		}

		body := parseErrorResponse(t, resp)
		if body.Success {
			t.Errorf("header %q: body.Success seharusnya false", headerVal)
		}
		if body.Message != "format Authorization harus Bearer <token>" {
			t.Errorf("header %q: pesan error tidak sesuai: %s", headerVal, body.Message)
		}
	}
}

func TestAuthMiddleware_ExpiredToken_Returns401(t *testing.T) {
	origSecret := os.Getenv("JWT_SECRET")
	defer os.Setenv("JWT_SECRET", origSecret)
	os.Setenv("JWT_SECRET", testJWTSecret)

	app := setupTestApp()

	claims := helper.JWTClaims{
		UserID: 1,
		Email:  "admin@test.local",
		Role:   model.RoleAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(testJWTSecret))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/animals", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("gagal menjalankan request: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status code = %d, seharusnya 401 Unauthorized", resp.StatusCode)
	}

	body := parseErrorResponse(t, resp)
	if body.Success {
		t.Error("body.Success seharusnya false")
	}
	if body.Message != "token tidak valid atau sudah kedaluwarsa" {
		t.Errorf("pesan error tidak sesuai: %s", body.Message)
	}
}

func TestAuthMiddleware_InvalidSignature_Returns401(t *testing.T) {
	origSecret := os.Getenv("JWT_SECRET")
	defer os.Setenv("JWT_SECRET", origSecret)

	// Buat token dengan secret lain
	claims := helper.JWTClaims{
		UserID: 1,
		Email:  "admin@test.local",
		Role:   model.RoleAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte("wrong-secret-key"))

	// Aplikasi menggunakan secret resmi
	os.Setenv("JWT_SECRET", testJWTSecret)
	app := setupTestApp()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/animals", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("gagal request: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status code = %d, seharusnya 401 Unauthorized", resp.StatusCode)
	}
}

func TestAuthMiddleware_InvalidAlgorithm_Returns401(t *testing.T) {
	origSecret := os.Getenv("JWT_SECRET")
	defer os.Setenv("JWT_SECRET", origSecret)
	os.Setenv("JWT_SECRET", testJWTSecret)

	app := setupTestApp()

	// Token ditandatangani dengan HS384 (bukan HS256)
	claims := helper.JWTClaims{
		UserID: 1,
		Email:  "admin@test.local",
		Role:   model.RoleAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS384, claims)
	tokenStr, _ := token.SignedString([]byte(testJWTSecret))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/animals", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("gagal request: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status code = %d, seharusnya 401 Unauthorized saat algoritma bukan HS256", resp.StatusCode)
	}
}

func TestAuthMiddleware_EmptySecretInEnv_Returns401(t *testing.T) {
	origSecret := os.Getenv("JWT_SECRET")
	defer os.Setenv("JWT_SECRET", origSecret)

	// Buat token saat secret ada
	os.Setenv("JWT_SECRET", testJWTSecret)
	token, _ := helper.GenerateToken(1, "admin@test.local", model.RoleAdmin, nil)

	// Kosongkan secret di env
	os.Setenv("JWT_SECRET", "")
	app := setupTestApp()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/animals", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("gagal request: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status code = %d, seharusnya 401 Unauthorized saat JWT_SECRET kosong", resp.StatusCode)
	}
}

func TestRoleMiddleware_AllowedRole_Success(t *testing.T) {
	origSecret := os.Getenv("JWT_SECRET")
	defer os.Setenv("JWT_SECRET", origSecret)
	os.Setenv("JWT_SECRET", testJWTSecret)

	app := setupTestApp()

	token, err := helper.GenerateToken(1, "admin@kurbantrack.local", model.RoleAdmin, nil)
	if err != nil {
		t.Fatalf("gagal generate token: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/animals", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("gagal request: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status code = %d, seharusnya 201 Created", resp.StatusCode)
	}

	var body helper.Response
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if !body.Success {
		t.Error("body.Success seharusnya true")
	}
}

func TestRoleMiddleware_MultipleAllowedRoles(t *testing.T) {
	origSecret := os.Getenv("JWT_SECRET")
	defer os.Setenv("JWT_SECRET", origSecret)
	os.Setenv("JWT_SECRET", testJWTSecret)

	app := setupTestApp()

	// 1. Admin boleh mengakses GET /api/v1/distributions
	adminToken, _ := helper.GenerateToken(1, "admin@kurbantrack.local", model.RoleAdmin, nil)
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/distributions", nil)
	req1.Header.Set("Authorization", "Bearer "+adminToken)

	resp1, err := app.Test(req1)
	if err != nil {
		t.Fatalf("gagal request 1: %v", err)
	}
	if resp1.StatusCode != http.StatusOK {
		t.Errorf("Admin: status code = %d, seharusnya 200 OK", resp1.StatusCode)
	}

	// 2. Koordinator Wilayah juga boleh mengakses GET /api/v1/distributions
	rt := "RT01"
	korwilToken, _ := helper.GenerateToken(2, "korwil@kurbantrack.local", model.RoleKoordinatorWilayah, &rt)
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/distributions", nil)
	req2.Header.Set("Authorization", "Bearer "+korwilToken)

	resp2, err := app.Test(req2)
	if err != nil {
		t.Fatalf("gagal request 2: %v", err)
	}
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("Koordinator Wilayah: status code = %d, seharusnya 200 OK", resp2.StatusCode)
	}

	// 3. Panitia Distribusi DITOLAK mengakses GET /api/v1/distributions
	panitiaToken, _ := helper.GenerateToken(3, "panitia@kurbantrack.local", model.RolePanitiaDistribusi, nil)
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/distributions", nil)
	req3.Header.Set("Authorization", "Bearer "+panitiaToken)

	resp3, err := app.Test(req3)
	if err != nil {
		t.Fatalf("gagal request 3: %v", err)
	}
	if resp3.StatusCode != http.StatusForbidden {
		t.Errorf("Panitia Distribusi: status code = %d, seharusnya 403 Forbidden", resp3.StatusCode)
	}

	body := parseErrorResponse(t, resp3)
	if body.Success {
		t.Error("body.Success seharusnya false pada response 403")
	}
}

func TestRoleMiddleware_MissingRoleInContext_Returns401(t *testing.T) {
	app := fiber.New()
	// Middleware RequireRole dipanggil tanpa RequireAuth sebelumnya
	app.Get("/test-unauth", middleware.RequireRole(model.RoleAdmin), func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test-unauth", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("gagal request: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status code = %d, seharusnya 401 Unauthorized saat context role kosong", resp.StatusCode)
	}

	body := parseErrorResponse(t, resp)
	if body.Message != "identitas role pengguna tidak ditemukan" {
		t.Errorf("pesan error tidak sesuai: %s", body.Message)
	}
}

func TestRoleMiddleware_InvalidSystemRole_Returns403(t *testing.T) {
	app := fiber.New()
	// Simulasi context dengan role aneh/tidak terdaftar
	app.Get("/test-invalid-role",
		func(c *fiber.Ctx) error {
			c.Locals("role", "role_asing")
			return c.Next()
		},
		middleware.RequireRole(model.RoleAdmin),
		func(c *fiber.Ctx) error {
			return c.SendString("ok")
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/test-invalid-role", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("gagal request: %v", err)
	}

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status code = %d, seharusnya 403 Forbidden saat role di context tidak valid", resp.StatusCode)
	}

	body := parseErrorResponse(t, resp)
	if body.Message != "role pengguna tidak valid" {
		t.Errorf("pesan error tidak sesuai: %s", body.Message)
	}
}
