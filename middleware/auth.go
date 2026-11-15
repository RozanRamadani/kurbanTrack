package middleware

import (
	"strings"

	"kurbantrack-backend/helper"

	"github.com/gofiber/fiber/v2"
)

// RequireAuth memvalidasi header Authorization: Bearer <token> dan menyimpan claims ke Locals.
func RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "header Authorization wajib disertakan",
			})
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "format Authorization harus Bearer <token>",
			})
		}

		claims, err := helper.ParseToken(parts[1])
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "token tidak valid atau sudah kedaluwarsa",
			})
		}

		c.Locals("user_id", claims.UserID)
		c.Locals("email", claims.Email)
		c.Locals("role", claims.Role)
		c.Locals("region_rt", claims.RegionRT)

		return c.Next()
	}
}

// RequireRole membatasi akses endpoint hanya untuk role tertentu (RBAC).
func RequireRole(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		roleVal := c.Locals("role")
		userRole, ok := roleVal.(string)
		if !ok || userRole == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "identitas role pengguna tidak ditemukan",
			})
		}

		for _, allowed := range allowedRoles {
			if userRole == allowed {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "error",
			"message": "akses ditolak: role Anda tidak memiliki izin untuk aksi ini",
		})
	}
}
