package helper_test

import (
	"os"
	"testing"
	"time"

	"kurbantrack-backend/app/model"
	"kurbantrack-backend/helper"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "unit-test-secret-key-for-kurbantrack"

func TestJWT_EmptySecret_ReturnsError(t *testing.T) {
	// Simpan nilai env sebelumnya
	origSecret := os.Getenv("JWT_SECRET")
	defer os.Setenv("JWT_SECRET", origSecret)

	// Set secret kosong
	os.Setenv("JWT_SECRET", "")

	_, err := helper.GenerateToken(1, "admin@test.local", model.RoleAdmin, nil)
	if err == nil {
		t.Error("GenerateToken seharusnya gagal saat JWT_SECRET kosong")
	}

	_, err = helper.ParseToken("dummy.token.string")
	if err == nil {
		t.Error("ParseToken seharusnya gagal saat JWT_SECRET kosong")
	}
}

func TestJWT_InvalidRole_ReturnsError(t *testing.T) {
	origSecret := os.Getenv("JWT_SECRET")
	defer os.Setenv("JWT_SECRET", origSecret)
	os.Setenv("JWT_SECRET", testSecret)

	_, err := helper.GenerateToken(1, "user@test.local", "invalid_role", nil)
	if err != helper.ErrInvalidRole {
		t.Errorf("GenerateToken dengan invalid role harus return ErrInvalidRole, dapat: %v", err)
	}
}

func TestJWT_ExpiredToken_FailsParsing(t *testing.T) {
	origSecret := os.Getenv("JWT_SECRET")
	defer os.Setenv("JWT_SECRET", origSecret)
	os.Setenv("JWT_SECRET", testSecret)

	// Buat token yang sudah kedaluwarsa 1 jam lalu
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
	tokenStr, err := token.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("gagal membuat token kadaluarsa: %v", err)
	}

	_, err = helper.ParseToken(tokenStr)
	if err == nil {
		t.Error("ParseToken seharusnya gagal pada expired token")
	}
}

func TestJWT_InvalidSignature_FailsParsing(t *testing.T) {
	origSecret := os.Getenv("JWT_SECRET")
	defer os.Setenv("JWT_SECRET", origSecret)

	// Tandatangani dengan secret A
	claims := helper.JWTClaims{
		UserID: 1,
		Email:  "admin@test.local",
		Role:   model.RoleAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte("secret-key-A"))
	if err != nil {
		t.Fatalf("gagal membuat token: %v", err)
	}

	// Coba verifikasi dengan secret B
	os.Setenv("JWT_SECRET", "secret-key-B")
	_, err = helper.ParseToken(tokenStr)
	if err == nil {
		t.Error("ParseToken seharusnya gagal pada token dengan signature yang tidak cocok")
	}
}

func TestJWT_InvalidSigningAlgorithm_FailsParsing(t *testing.T) {
	origSecret := os.Getenv("JWT_SECRET")
	defer os.Setenv("JWT_SECRET", origSecret)
	os.Setenv("JWT_SECRET", testSecret)

	// Buat token menggunakan algoritma HS512 (bukan HS256)
	claims := helper.JWTClaims{
		UserID: 1,
		Email:  "admin@test.local",
		Role:   model.RoleAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	tokenHS512 := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	tokenStr, err := tokenHS512.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("gagal membuat token HS512: %v", err)
	}

	_, err = helper.ParseToken(tokenStr)
	if err == nil {
		t.Error("ParseToken seharusnya menolak token yang ditandatangani dengan algoritma selain HS256")
	}
}

func TestJWT_InvalidRoleInClaims_FailsParsing(t *testing.T) {
	origSecret := os.Getenv("JWT_SECRET")
	defer os.Setenv("JWT_SECRET", origSecret)
	os.Setenv("JWT_SECRET", testSecret)

	// Buat token langsung dengan role palsu
	claims := helper.JWTClaims{
		UserID: 1,
		Email:  "intruder@test.local",
		Role:   "superadmin_palsu",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("gagal membuat token: %v", err)
	}

	_, err = helper.ParseToken(tokenStr)
	if err != helper.ErrInvalidRole {
		t.Errorf("ParseToken harus menolak token dengan role tidak valid di claims, dapat: %v", err)
	}
}

func TestJWT_ValidToken_Success(t *testing.T) {
	origSecret := os.Getenv("JWT_SECRET")
	defer os.Setenv("JWT_SECRET", origSecret)
	os.Setenv("JWT_SECRET", testSecret)

	rt := "RT05"
	tokenStr, err := helper.GenerateToken(10, "koordinator@test.local", model.RoleKoordinatorWilayah, &rt)
	if err != nil {
		t.Fatalf("GenerateToken gagal: %v", err)
	}

	claims, err := helper.ParseToken(tokenStr)
	if err != nil {
		t.Fatalf("ParseToken gagal: %v", err)
	}

	if claims.UserID != 10 || claims.Email != "koordinator@test.local" || claims.Role != model.RoleKoordinatorWilayah {
		t.Errorf("Claims tidak sesuai: %+v", claims)
	}
	if claims.RegionRT == nil || *claims.RegionRT != "RT05" {
		t.Errorf("RegionRT tidak sesuai: %v", claims.RegionRT)
	}
}
