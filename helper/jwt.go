package helper

import (
	"errors"
	"os"
	"strings"
	"time"

	"kurbantrack-backend/app/model"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrEmptyJWTSecret       = errors.New("JWT_SECRET wajib dikonfigurasi dan tidak boleh kosong")
	ErrInvalidSigningMethod = errors.New("metode signing token tidak valid: wajib menggunakan HS256")
	ErrInvalidToken         = errors.New("token tidak valid atau sudah kedaluwarsa")
	ErrInvalidRole          = errors.New("role token tidak valid")
)

// JWTClaims menyimpan data identitas dan role pengguna di dalam payload token.
type JWTClaims struct {
	UserID   int     `json:"user_id"`
	Email    string  `json:"email"`
	Role     string  `json:"role"`
	RegionRT *string `json:"region_rt,omitempty"`
	jwt.RegisteredClaims
}

func getJWTSecret() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if strings.TrimSpace(secret) == "" {
		return nil, ErrEmptyJWTSecret
	}
	return []byte(secret), nil
}

// GenerateToken menghasilkan token JWT baru yang berlaku selama 24 jam menggunakan algoritma HS256.
func GenerateToken(userID int, email, role string, regionRT *string) (string, error) {
	secret, err := getJWTSecret()
	if err != nil {
		return "", err
	}

	if !model.IsValidRole(role) {
		return "", ErrInvalidRole
	}

	claims := JWTClaims{
		UserID:   userID,
		Email:    email,
		Role:     role,
		RegionRT: regionRT,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ParseToken memverifikasi tanda tangan token JWT menggunakan HS256 dan mengembalikan claims-nya.
func ParseToken(tokenString string) (*JWTClaims, error) {
	secret, err := getJWTSecret()
	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseWithClaims(
		tokenString,
		&JWTClaims{},
		func(token *jwt.Token) (interface{}, error) {
			if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, ErrInvalidSigningMethod
			}
			return secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
	)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	if !model.IsValidRole(claims.Role) {
		return nil, ErrInvalidRole
	}

	return claims, nil
}
