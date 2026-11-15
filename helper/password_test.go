package helper_test

import (
	"strings"
	"testing"

	"kurbantrack-backend/helper"
)

func TestHashPassword_Success(t *testing.T) {
	password := "RahasiaKurban2026!"
	hash, err := helper.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword gagal: %v", err)
	}

	if hash == "" || hash == password {
		t.Fatal("Hash tidak boleh kosong atau sama dengan plain password")
	}

	// Format hash bcrypt diawali dengan $2a$
	if !strings.HasPrefix(hash, "$2a$") {
		t.Errorf("Format hash bukan bcrypt ($2a$): %s", hash)
	}

	// Verifikasi kecocokan password
	if !helper.CheckPasswordHash(password, hash) {
		t.Error("CheckPasswordHash seharusnya menghasilkan true untuk password yang benar")
	}
}

func TestHashPassword_DifferentSalts(t *testing.T) {
	password := "PasswordKuat123"
	hash1, err1 := helper.HashPassword(password)
	hash2, err2 := helper.HashPassword(password)

	if err1 != nil || err2 != nil {
		t.Fatalf("Gagal hashing: %v, %v", err1, err2)
	}

	// Bcrypt harus menghasilkan hash berbeda karena salt unik setiap enkripsi
	if hash1 == hash2 {
		t.Error("Dua hash dari password yang sama tidak boleh identik (salt harus unik)")
	}

	// Kedua hash tetap valid untuk password yang sama
	if !helper.CheckPasswordHash(password, hash1) || !helper.CheckPasswordHash(password, hash2) {
		t.Error("Kedua hash harus valid untuk password yang sama")
	}
}

func TestHashPassword_EmptyOrWhitespace(t *testing.T) {
	_, err := helper.HashPassword("")
	if err == nil {
		t.Error("HashPassword seharusnya menghasilkan error untuk password kosong")
	}

	_, err = helper.HashPassword("   ")
	if err == nil {
		t.Error("HashPassword seharusnya menghasilkan error untuk password yang hanya whitespace")
	}
}

func TestCheckPasswordHash_WrongPassword(t *testing.T) {
	password := "PasswordAsli"
	hash, err := helper.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword gagal: %v", err)
	}

	if helper.CheckPasswordHash("PasswordSalah", hash) {
		t.Error("CheckPasswordHash seharusnya menghasilkan false untuk password yang salah")
	}
}

func TestCheckPasswordHash_EmptyInput(t *testing.T) {
	if helper.CheckPasswordHash("", "$2a$10$abcdefghijklmnopqrstuu") {
		t.Error("CheckPasswordHash harus false jika password kosong")
	}

	if helper.CheckPasswordHash("password", "") {
		t.Error("CheckPasswordHash harus false jika hash kosong")
	}
}
