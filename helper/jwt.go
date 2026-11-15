package helper

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims menyimpan data identitas dan role pengguna di dalam payload token.
type JWTClaims struct {
	UserID   int     `json:"user_id"`
	Email    string  `json:"email"`
	Role     string  `json:"role"`
	RegionRT *string `json:"region_rt,omitempty"`
	jwt.RegisteredClaims
}

func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "kurbantrack-default-secret-key-2026"
	}
	return []byte(secret)
}

// GenerateToken menghasilkan token JWT baru yang berlaku selama 24 jam.
func GenerateToken(userID int, email, role string, regionRT *string) (string, error) {
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
	return token.SignedString(getJWTSecret())
}

// ParseToken memverifikasi tanda tangan token JWT dan mengembalikan claims-nya.
func ParseToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("metode signing token tidak valid")
		}
		return getJWTSecret(), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("token tidak valid atau sudah kedaluwarsa")
	}

	return claims, nil
}
