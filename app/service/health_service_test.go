package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"kurbantrack-backend/app/service"

	"github.com/gofiber/fiber/v2"
)

// mockDBPinger adalah implementasi tiruan DBPinger untuk keperluan pengujian unit.
type mockDBPinger struct {
	pingFunc func(ctx context.Context) error
}

func (m *mockDBPinger) Ping(ctx context.Context) error {
	if m.pingFunc != nil {
		return m.pingFunc(ctx)
	}
	return nil
}

type healthResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Server        string `json:"server"`
		Database      string `json:"database"`
		DatabaseError string `json:"database_error,omitempty"`
	} `json:"data"`
	Error string `json:"error,omitempty"`
}

func TestHealthService_CheckHealth_Success(t *testing.T) {
	// Kondisi 1: Database berhasil di-ping (Ping() mengembalikan nil) -> HTTP 200
	mockDB := &mockDBPinger{
		pingFunc: func(ctx context.Context) error {
			return nil
		},
	}

	svc := service.NewHealthService(mockDB)
	app := fiber.New()
	app.Get("/health", svc.CheckHealth)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Gagal melakukan request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Ekspektasi status code %d, didapat %d", http.StatusOK, resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var res healthResponse
	if err := json.Unmarshal(body, &res); err != nil {
		t.Fatalf("Gagal unmarshal response JSON: %v", err)
	}

	if !res.Success {
		t.Errorf("Ekspektasi success true, didapat false")
	}
	if res.Data.Server != "healthy" {
		t.Errorf("Ekspektasi server 'healthy', didapat '%s'", res.Data.Server)
	}
	if res.Data.Database != "connected" {
		t.Errorf("Ekspektasi database 'connected', didapat '%s'", res.Data.Database)
	}
}

func TestHealthService_CheckHealth_DatabaseError(t *testing.T) {
	// Kondisi 2: Database gagal di-ping (Ping() mengembalikan error) -> HTTP 503
	mockDB := &mockDBPinger{
		pingFunc: func(ctx context.Context) error {
			return errors.New("koneksi database ditolak")
		},
	}

	svc := service.NewHealthService(mockDB)
	app := fiber.New()
	app.Get("/health", svc.CheckHealth)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Gagal melakukan request: %v", err)
	}

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Ekspektasi status code %d, didapat %d", http.StatusServiceUnavailable, resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var res healthResponse
	if err := json.Unmarshal(body, &res); err != nil {
		t.Fatalf("Gagal unmarshal response JSON: %v", err)
	}

	if res.Success {
		t.Errorf("Ekspektasi success false, didapat true")
	}
	if res.Data.Database != "disconnected" {
		t.Errorf("Ekspektasi database 'disconnected', didapat '%s'", res.Data.Database)
	}
	if res.Data.DatabaseError != "koneksi database ditolak" {
		t.Errorf("Ekspektasi database_error 'koneksi database ditolak', didapat '%s'", res.Data.DatabaseError)
	}
}

func TestHealthService_CheckHealth_Timeout(t *testing.T) {
	// Kondisi 3: Database mengalami timeout -> respons gagal sesuai kontrak healthcheck (HTTP 503)
	mockDB := &mockDBPinger{
		pingFunc: func(ctx context.Context) error {
			return context.DeadlineExceeded
		},
	}

	svc := service.NewHealthService(mockDB)
	app := fiber.New()
	app.Get("/health", svc.CheckHealth)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Gagal melakukan request: %v", err)
	}

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Ekspektasi status code %d, didapat %d", http.StatusServiceUnavailable, resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var res healthResponse
	if err := json.Unmarshal(body, &res); err != nil {
		t.Fatalf("Gagal unmarshal response JSON: %v", err)
	}

	if res.Success {
		t.Errorf("Ekspektasi success false, didapat true")
	}
	if res.Data.Database != "disconnected" {
		t.Errorf("Ekspektasi database 'disconnected', didapat '%s'", res.Data.Database)
	}
}
