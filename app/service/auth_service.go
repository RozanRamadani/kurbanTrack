package service

import (
	"context"
	"errors"
	"strings"

	"kurbantrack-backend/app/model"
	"kurbantrack-backend/app/repository"
	"kurbantrack-backend/helper"

	"github.com/gofiber/fiber/v2"
)

// AuthService mendefinisikan interface bisnis dan HTTP-facing untuk autentikasi pengguna.
type AuthService interface {
	RegisterUser(ctx context.Context, req *model.UserCreateRequest) (*model.UserResponse, error)
	LoginUser(ctx context.Context, req *model.UserLoginRequest) (*model.AuthResponse, error)

	Register(c *fiber.Ctx) error
	Login(c *fiber.Ctx) error
}

type authService struct {
	userRepo repository.UserRepository
}

// NewAuthService menginisialisasi AuthService dengan dependency injection UserRepository.
func NewAuthService(userRepo repository.UserRepository) AuthService {
	return &authService{userRepo: userRepo}
}

// ----------------------------------------------------------------------------
// BUSINESS METHODS
// ----------------------------------------------------------------------------

func (s *authService) RegisterUser(ctx context.Context, req *model.UserCreateRequest) (*model.UserResponse, error) {
	name := strings.TrimSpace(req.Name)
	email := strings.TrimSpace(strings.ToLower(req.Email))

	if name == "" || email == "" || len(req.Password) < 6 {
		return nil, &ValidationError{
			Field:   "credentials",
			Message: "Nama, email wajib diisi dan password minimal 6 karakter",
		}
	}

	if req.Role != model.RoleAdmin && req.Role != model.RoleKoordinatorWilayah && req.Role != model.RolePanitiaDistribusi {
		return nil, &ValidationError{
			Field:   "role",
			Message: "Role harus admin, koordinator_wilayah, atau panitia_distribusi",
		}
	}

	hashed, err := helper.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Name:         name,
		Email:        email,
		PasswordHash: hashed,
		Role:         req.Role,
		RegionRT:     req.RegionRT,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrConflict
		}
		return nil, err
	}

	return &model.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		RegionRT:  user.RegionRT,
		CreatedAt: user.CreatedAt,
	}, nil
}

func (s *authService) LoginUser(ctx context.Context, req *model.UserLoginRequest) (*model.AuthResponse, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" || req.Password == "" {
		return nil, &ValidationError{
			Field:   "credentials",
			Message: "Email dan password wajib diisi",
		}
	}

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}

	if !helper.CheckPasswordHash(req.Password, user.PasswordHash) {
		return nil, ErrUnauthorized
	}

	token, err := helper.GenerateToken(user.ID, user.Email, user.Role, user.RegionRT)
	if err != nil {
		return nil, err
	}

	return &model.AuthResponse{
		Token: token,
		User: model.UserResponse{
			ID:        user.ID,
			Name:      user.Name,
			Email:     user.Email,
			Role:      user.Role,
			RegionRT:  user.RegionRT,
			CreatedAt: user.CreatedAt,
		},
	}, nil
}

// ----------------------------------------------------------------------------
// FIBER HTTP-FACING METHODS
// ----------------------------------------------------------------------------

func (s *authService) Register(c *fiber.Ctx) error {
	var req model.UserCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			helper.ErrorResponse("Format request body tidak valid", err.Error()),
		)
	}

	res, err := s.RegisterUser(c.Context(), &req)
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
		if errors.Is(err, ErrConflict) {
			return c.Status(fiber.StatusConflict).JSON(
				helper.ErrorResponse("Email sudah terdaftar", nil),
			)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(
			helper.ErrorResponse("Terjadi kesalahan internal pada server", nil),
		)
	}

	return c.Status(fiber.StatusCreated).JSON(
		helper.SuccessResponse("Registrasi pengguna berhasil", res),
	)
}

func (s *authService) Login(c *fiber.Ctx) error {
	var req model.UserLoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			helper.ErrorResponse("Format request body tidak valid", err.Error()),
		)
	}

	res, err := s.LoginUser(c.Context(), &req)
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
		if errors.Is(err, ErrUnauthorized) {
			return c.Status(fiber.StatusUnauthorized).JSON(
				helper.ErrorResponse("Email atau password salah", nil),
			)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(
			helper.ErrorResponse("Terjadi kesalahan internal pada server", nil),
		)
	}

	return c.Status(fiber.StatusOK).JSON(
		helper.SuccessResponse("Login berhasil", res),
	)
}
