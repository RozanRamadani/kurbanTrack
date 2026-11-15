package helper

import (
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword mengenkripsi plain password menggunakan bcrypt dengan cost standar (10).
func HashPassword(password string) (string, error) {
	if strings.TrimSpace(password) == "" {
		return "", errors.New("password tidak boleh kosong")
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPasswordHash membandingkan plain password dengan password_hash dari database.
func CheckPasswordHash(password, hash string) bool {
	if password == "" || hash == "" {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
