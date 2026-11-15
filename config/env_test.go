package config_test

import (
	"testing"

	"kurbantrack-backend/config"
)

func TestConfig_Validate_MissingJWTSecret(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "",
	}

	if err := cfg.Validate(); err == nil {
		t.Error("seharusnya menghasilkan error saat JWTSecret kosong")
	}

	cfgWhitespace := &config.Config{
		JWTSecret: "   ",
	}

	if err := cfgWhitespace.Validate(); err == nil {
		t.Error("seharusnya menghasilkan error saat JWTSecret hanya berupa whitespace")
	}
}

func TestConfig_Validate_ValidJWTSecret(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "super-secret-jwt-key",
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("tidak seharusnya error saat JWTSecret terisi: %v", err)
	}
}
