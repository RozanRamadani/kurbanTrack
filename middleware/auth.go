package middleware

import (
	"strings"

	"kurbantrack-backend/app/model"
	"kurbantrack-backend/helper"

	"github.com/gofiber/fiber/v2"
)

// RequireAuth memvalidasi header Authorization: Bearer <token> dan menyimpan claims ke Locals.
func RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if strings.TrimSpace(authHeader) == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(helper.ErrorResponse("header Authorization wajib disertakan", nil))
		}

		fields := strings.Fields(authHeader)
		if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
			return c.Status(fiber.StatusUnauthorized).JSON(helper.ErrorResponse("format Authorization harus Bearer <token>", nil))
		}

		claims, err := helper.ParseToken(fields[1])
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(helper.ErrorResponse("token tidak valid atau sudah kedaluwarsa", err.Error()))
		}

		c.Locals("user_id", claims.UserID)
		c.Locals("email", claims.Email)
		c.Locals("role", claims.Role)
		c.Locals("region_rt", claims.RegionRT)

		return c.Next()
	}
}

// RequireRole membatasi akses endpoint hanya untuk role tertentu yang valid (RBAC).
func RequireRole(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		roleVal := c.Locals("role")
		userRole, ok := roleVal.(string)
		if !ok || userRole == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(helper.ErrorResponse("identitas role pengguna tidak ditemukan", nil))
		}

		if !model.IsValidRole(userRole) {
			return c.Status(fiber.StatusForbidden).JSON(helper.ErrorResponse("role pengguna tidak valid", nil))
		}

		for _, allowed := range allowedRoles {
			if userRole == allowed {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(helper.ErrorResponse("akses ditolak: role Anda tidak memiliki izin untuk aksi ini", nil))
	}
}
