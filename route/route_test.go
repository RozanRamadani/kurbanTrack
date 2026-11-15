package route_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kurbantrack-backend/app/model"
	"kurbantrack-backend/app/repository"
	"kurbantrack-backend/app/service"
	"kurbantrack-backend/config"
	"kurbantrack-backend/helper"

	"github.com/gofiber/fiber/v2"
)

// mockAnimalRepoForHTTP adalah mock repository untuk menguji HTTP handler tanpa PostgreSQL
type mockAnimalRepoForHTTP struct {
	animals           map[int]*model.Animal
	activeAllocations map[int]int
	nextID            int
	createErr         error
	findErr           error
	updateErr         error
	deleteErr         error
}

func newMockAnimalRepoForHTTP() *mockAnimalRepoForHTTP {
	return &mockAnimalRepoForHTTP{
		animals:           make(map[int]*model.Animal),
		activeAllocations: make(map[int]int),
		nextID:            1,
	}
}

func (m *mockAnimalRepoForHTTP) Create(ctx context.Context, animal *model.Animal) error {
	if m.createErr != nil {
		return m.createErr
	}
	for _, a := range m.animals {
		if a.Code == animal.Code {
			return repository.ErrConflict
		}
	}
	animal.ID = m.nextID
	m.nextID++
	animal.CreatedAt = time.Now()
	m.animals[animal.ID] = animal
	return nil
}

func (m *mockAnimalRepoForHTTP) FindByID(ctx context.Context, id int) (*model.Animal, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	animal, ok := m.animals[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	copied := *animal
	return &copied, nil
}

func (m *mockAnimalRepoForHTTP) FindAll(ctx context.Context) ([]model.Animal, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	list := make([]model.Animal, 0, len(m.animals))
	for _, a := range m.animals {
		list = append(list, *a)
	}
	return list, nil
}

func (m *mockAnimalRepoForHTTP) Update(ctx context.Context, animal *model.Animal) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if _, ok := m.animals[animal.ID]; !ok {
		return repository.ErrNotFound
	}
	for id, a := range m.animals {
		if a.Code == animal.Code && id != animal.ID {
			return repository.ErrConflict
		}
	}
	m.animals[animal.ID] = animal
	return nil
}

func (m *mockAnimalRepoForHTTP) Delete(ctx context.Context, id int) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if _, ok := m.animals[id]; !ok {
		return repository.ErrNotFound
	}
	delete(m.animals, id)
	return nil
}

func (m *mockAnimalRepoForHTTP) CountAllocationsByAnimalID(ctx context.Context, animalID int) (int, error) {
	return m.activeAllocations[animalID], nil
}

type mockDistributionRepoForHTTP struct {
	distributions map[int]*model.Distribution
	animalRepo    *mockAnimalRepoForHTTP
	nextID        int
}

func newMockDistributionRepoForHTTP(animalRepo *mockAnimalRepoForHTTP) *mockDistributionRepoForHTTP {
	return &mockDistributionRepoForHTTP{
		distributions: make(map[int]*model.Distribution),
		animalRepo:    animalRepo,
		nextID:        1,
	}
}

func (m *mockDistributionRepoForHTTP) Create(ctx context.Context, dist *model.Distribution) error {
	return m.AllocateWithQuotaTx(ctx, dist)
}

func (m *mockDistributionRepoForHTTP) FindByID(ctx context.Context, id int) (*model.Distribution, error) {
	d, ok := m.distributions[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return d, nil
}

func (m *mockDistributionRepoForHTTP) FindByCouponCode(ctx context.Context, code string) (*model.Distribution, error) {
	for _, d := range m.distributions {
		if d.CouponCode == code {
			return d, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (m *mockDistributionRepoForHTTP) FindAll(ctx context.Context) ([]model.Distribution, error) {
	list := make([]model.Distribution, 0, len(m.distributions))
	for _, d := range m.distributions {
		list = append(list, *d)
	}
	return list, nil
}

func (m *mockDistributionRepoForHTTP) UpdateStatus(ctx context.Context, id int, status string, redeemedAt *time.Time) error {
	d, ok := m.distributions[id]
	if !ok {
		return repository.ErrNotFound
	}
	d.Status = status
	d.RedeemedAt = redeemedAt
	return nil
}

func (m *mockDistributionRepoForHTTP) AllocateWithQuotaTx(ctx context.Context, dist *model.Distribution) error {
	for _, existing := range m.distributions {
		if existing.CouponCode == dist.CouponCode || existing.RecipientNIK == dist.RecipientNIK {
			return repository.ErrConflict
		}
	}
	if m.animalRepo != nil {
		animal, ok := m.animalRepo.animals[dist.AnimalID]
		if !ok {
			return repository.ErrNotFound
		}
		animal.AllocatedPackages++
	}
	dist.ID = m.nextID
	m.nextID++
	dist.CreatedAt = time.Now()
	m.distributions[dist.ID] = dist
	return nil
}

func (m *mockDistributionRepoForHTTP) RedeemCouponTx(ctx context.Context, couponCode string, redeemedAt time.Time, actorID int) (*model.Distribution, error) {
	for _, d := range m.distributions {
		if d.CouponCode == couponCode {
			if d.Status != model.DistributionStatusAllocated {
				return nil, repository.ErrConflict
			}
			d.Status = model.DistributionStatusRedeemed
			d.RedeemedAt = &redeemedAt
			return d, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (m *mockDistributionRepoForHTTP) CancelCouponTx(ctx context.Context, couponCode string, actorID int) (*model.Distribution, error) {
	for _, d := range m.distributions {
		if d.CouponCode == couponCode {
			if d.Status != model.DistributionStatusAllocated {
				return nil, repository.ErrConflict
			}
			d.Status = model.DistributionStatusCancelled
			d.RedeemedAt = nil
			if m.animalRepo != nil {
				if animal, ok := m.animalRepo.animals[d.AnimalID]; ok && animal.AllocatedPackages > 0 {
					animal.AllocatedPackages--
				}
			}
			return d, nil
		}
	}
	return nil, repository.ErrNotFound
}

// ----------------------------------------------------------------------------
// TEST CASES UNTUK ROUTE & HTTP INTEGRATION
// ----------------------------------------------------------------------------

func setupTestApp() (*fiber.App, *mockAnimalRepoForHTTP) {
	app, animalRepo, _ := setupTestAppWithDist()
	return app, animalRepo
}

func setupTestAppWithDist() (*fiber.App, *mockAnimalRepoForHTTP, *mockDistributionRepoForHTTP) {
	repo := newMockAnimalRepoForHTTP()
	distRepo := newMockDistributionRepoForHTTP(repo)
	animalService := service.NewAnimalService(repo)
	distributionService := service.NewDistributionService(distRepo, repo)
	healthService := service.NewHealthService(nil)
	app := config.NewApp(healthService, animalService, distributionService)
	return app, repo, distRepo
}

func TestHTTP_HealthEndpoint(t *testing.T) {
	app, _ := setupTestApp()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Gagal request /health: %v", err)
	}

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Ekspektasi status 503 saat DB uninitialized, didapat %d", resp.StatusCode)
	}
}

func TestHTTP_CreateAnimal_Success(t *testing.T) {
	app, _ := setupTestApp()

	payload := model.AnimalCreateRequest{
		Code:              "SP-HTTP-01",
		Species:           model.SpeciesSapi,
		SohibulName:       "Haji Salim",
		WeightKg:          350.50,
		EstimatedPackages: 100,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/animals", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Gagal melakukan request: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Ekspektasi status 201 Created, didapat %d", resp.StatusCode)
	}

	var res helper.Response
	respBytes, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(respBytes, &res)

	if !res.Success {
		t.Errorf("Ekspektasi response success true, didapat false")
	}
}

func TestHTTP_CreateAnimal_InvalidJSON(t *testing.T) {
	app, _ := setupTestApp()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/animals", bytes.NewReader([]byte("{invalid-json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Gagal request: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Ekspektasi status 400 Bad Request, didapat %d", resp.StatusCode)
	}
}

func TestHTTP_CreateAnimal_ValidationError(t *testing.T) {
	app, _ := setupTestApp()

	payload := model.AnimalCreateRequest{
		Code:              "", // Kode kosong
		Species:           "kucing",
		SohibulName:       "",
		WeightKg:          -5,
		EstimatedPackages: 0,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/animals", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Gagal request: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Ekspektasi status 400 untuk input tidak valid, didapat %d", resp.StatusCode)
	}
}

func TestHTTP_CreateAnimal_Conflict(t *testing.T) {
	app, repo := setupTestApp()

	// Masukkan hewan awal
	repo.animals[1] = &model.Animal{
		ID:                1,
		Code:              "SP-EXIST",
		Species:           model.SpeciesSapi,
		SohibulName:       "Ahmad",
		WeightKg:          300.0,
		EstimatedPackages: 80,
		Status:            model.AnimalStatusRegistered,
	}
	repo.nextID = 2

	// Coba buat dengan kode yang sama
	payload := model.AnimalCreateRequest{
		Code:              "SP-EXIST",
		Species:           model.SpeciesSapi,
		SohibulName:       "Budi",
		WeightKg:          320.0,
		EstimatedPackages: 90,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/animals", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Gagal request: %v", err)
	}

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("Ekspektasi status 409 Conflict, didapat %d", resp.StatusCode)
	}
}

func TestHTTP_CreateAnimal_InternalError(t *testing.T) {
	app, repo := setupTestApp()
	repo.createErr = errors.New("db disk failure")

	payload := model.AnimalCreateRequest{
		Code:              "SP-ERR",
		Species:           model.SpeciesSapi,
		SohibulName:       "Test",
		WeightKg:          300.0,
		EstimatedPackages: 80,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/animals", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Ekspektasi status 500 Internal Server Error, didapat %d", resp.StatusCode)
	}
}

func TestHTTP_GetAllAnimals(t *testing.T) {
	app, repo := setupTestApp()

	repo.animals[1] = &model.Animal{ID: 1, Code: "SP-01", Species: model.SpeciesSapi, SohibulName: "Ahmad", WeightKg: 300, EstimatedPackages: 80, Status: model.AnimalStatusRegistered}
	repo.animals[2] = &model.Animal{ID: 2, Code: "KB-02", Species: model.SpeciesKambing, SohibulName: "Umar", WeightKg: 40, EstimatedPackages: 15, Status: model.AnimalStatusRegistered}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/animals", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Gagal request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Ekspektasi status 200 OK, didapat %d", resp.StatusCode)
	}

	var res helper.Response
	respBytes, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(respBytes, &res)

	if !res.Success {
		t.Errorf("Ekspektasi success true, didapat false")
	}
}

func TestHTTP_GetAnimalByID_Success(t *testing.T) {
	app, repo := setupTestApp()

	repo.animals[1] = &model.Animal{ID: 1, Code: "SP-01", Species: model.SpeciesSapi, SohibulName: "Ahmad", WeightKg: 300, EstimatedPackages: 80, Status: model.AnimalStatusRegistered}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/animals/1", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Gagal request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Ekspektasi status 200 OK, didapat %d", resp.StatusCode)
	}
}

func TestHTTP_GetAnimalByID_NotFound(t *testing.T) {
	app, _ := setupTestApp()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/animals/999", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Gagal request: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Ekspektasi status 404 Not Found, didapat %d", resp.StatusCode)
	}
}

func TestHTTP_GetAnimalByID_InvalidID(t *testing.T) {
	app, _ := setupTestApp()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/animals/abc", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Ekspektasi status 400 Bad Request untuk ID non-integer, didapat %d", resp.StatusCode)
	}
}

func TestHTTP_UpdateAnimal_Success(t *testing.T) {
	app, repo := setupTestApp()

	repo.animals[1] = &model.Animal{ID: 1, Code: "SP-01", Species: model.SpeciesSapi, SohibulName: "Ahmad", WeightKg: 300, EstimatedPackages: 80, Status: model.AnimalStatusRegistered}

	payload := model.AnimalUpdateRequest{
		Code:              "SP-01-REV",
		Species:           model.SpeciesSapi,
		SohibulName:       "Ahmad Dahlan",
		WeightKg:          310.50,
		EstimatedPackages: 85,
		Status:            model.AnimalStatusSlaughtered,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/animals/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Gagal request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Ekspektasi status 200 OK, didapat %d", resp.StatusCode)
	}
}

func TestHTTP_UpdateAnimal_NotFound(t *testing.T) {
	app, _ := setupTestApp()

	payload := model.AnimalUpdateRequest{
		Code:              "SP-01",
		Species:           model.SpeciesSapi,
		SohibulName:       "Ahmad",
		WeightKg:          300,
		EstimatedPackages: 80,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/animals/999", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Ekspektasi status 404 Not Found, didapat %d", resp.StatusCode)
	}
}

func TestHTTP_DeleteAnimal_Success(t *testing.T) {
	app, repo := setupTestApp()

	repo.animals[1] = &model.Animal{ID: 1, Code: "SP-01", Species: model.SpeciesSapi, SohibulName: "Ahmad", WeightKg: 300, EstimatedPackages: 80, Status: model.AnimalStatusRegistered}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/animals/1", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Gagal request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Ekspektasi status 200 OK, didapat %d", resp.StatusCode)
	}
}

func TestHTTP_DeleteAnimal_WithActiveAllocations(t *testing.T) {
	app, repo := setupTestApp()

	repo.animals[1] = &model.Animal{ID: 1, Code: "SP-01", Species: model.SpeciesSapi, SohibulName: "Ahmad", WeightKg: 300, EstimatedPackages: 80, Status: model.AnimalStatusRegistered}
	repo.activeAllocations[1] = 5 // Ada 5 alokasi aktif

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/animals/1", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Gagal request: %v", err)
	}

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("Ekspektasi status 409 Conflict saat ada alokasi aktif, didapat %d", resp.StatusCode)
	}
}

func TestHTTP_DeleteAnimal_ForeignKeyViolation(t *testing.T) {
	app, repo := setupTestApp()

	repo.animals[1] = &model.Animal{ID: 1, Code: "SP-01", Species: model.SpeciesSapi, SohibulName: "Ahmad", WeightKg: 300, EstimatedPackages: 80, Status: model.AnimalStatusRegistered}
	repo.deleteErr = repository.ErrForeignKeyViolation

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/animals/1", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Gagal request: %v", err)
	}

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("Ekspektasi status 409 Conflict saat ada pelanggaran FK, didapat %d", resp.StatusCode)
	}
}

func TestHTTP_DeleteAnimal_NotFound(t *testing.T) {
	app, _ := setupTestApp()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/animals/999", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Gagal request: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Ekspektasi status 404 Not Found saat ID tidak ditemukan, didapat %d", resp.StatusCode)
	}
}

func TestHTTP_Distribution_Create_Scaffold(t *testing.T) {
	app, _ := setupTestApp()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/distributions", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Gagal request: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Ekspektasi status 401 Unauthorized saat user_id belum di-set di Locals, didapat %d", resp.StatusCode)
	}
}

func TestHTTP_Distribution_FindAll_Scaffold(t *testing.T) {
	app, _ := setupTestApp()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/distributions", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Gagal request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Ekspektasi status 200 OK untuk GET distributions, didapat %d", resp.StatusCode)
	}
}

func TestHTTP_Distribution_Redeem_Scaffold(t *testing.T) {
	app, _ := setupTestApp()

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/distributions/CPN-123/redeem", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Gagal request: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Ekspektasi status 401 Unauthorized saat user_id belum di-set di Locals, didapat %d", resp.StatusCode)
	}
}

func TestHTTP_Distribution_Cancel_Scaffold(t *testing.T) {
	app, _ := setupTestApp()

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/distributions/CPN-123/cancel", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Gagal request: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Ekspektasi status 401 Unauthorized saat user_id belum di-set di Locals, didapat %d", resp.StatusCode)
	}
}
