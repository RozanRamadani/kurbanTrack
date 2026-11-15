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

// ----------------------------------------------------------------------------
// MOCK REPOSITORIES
// ----------------------------------------------------------------------------

type mockUserRepoForHTTP struct {
	users  map[int]*model.User
	nextID int
}

func newMockUserRepoForHTTP() *mockUserRepoForHTTP {
	return &mockUserRepoForHTTP{
		users:  make(map[int]*model.User),
		nextID: 1,
	}
}

func (m *mockUserRepoForHTTP) Create(ctx context.Context, user *model.User) error {
	for _, u := range m.users {
		if u.Email == user.Email {
			return repository.ErrConflict
		}
	}
	user.ID = m.nextID
	m.nextID++
	user.CreatedAt = time.Now()
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepoForHTTP) FindByID(ctx context.Context, id int) (*model.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepoForHTTP) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (m *mockUserRepoForHTTP) FindAll(ctx context.Context) ([]model.User, error) {
	list := make([]model.User, 0, len(m.users))
	for _, u := range m.users {
		list = append(list, *u)
	}
	return list, nil
}

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
	nextID        int
}

func newMockDistributionRepoForHTTP() *mockDistributionRepoForHTTP {
	return &mockDistributionRepoForHTTP{
		distributions: make(map[int]*model.Distribution),
		nextID:        1,
	}
}

func (m *mockDistributionRepoForHTTP) Create(ctx context.Context, dist *model.Distribution) error {
	dist.ID = m.nextID
	m.nextID++
	m.distributions[dist.ID] = dist
	return nil
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

// ----------------------------------------------------------------------------
// SETUP HELPER & TOKEN GENERATOR
// ----------------------------------------------------------------------------

func setupTestApp() (*fiber.App, *mockAnimalRepoForHTTP) {
	app, animalRepo, _ := setupTestAppWithDist()
	return app, animalRepo
}

func setupTestAppWithDist() (*fiber.App, *mockAnimalRepoForHTTP, *mockDistributionRepoForHTTP) {
	userRepo := newMockUserRepoForHTTP()
	repo := newMockAnimalRepoForHTTP()
	distRepo := newMockDistributionRepoForHTTP()

	authService := service.NewAuthService(userRepo)
	animalService := service.NewAnimalService(repo)
	distributionService := service.NewDistributionService(distRepo)
	healthService := service.NewHealthService(nil)

	app := config.NewApp(healthService, authService, animalService, distributionService)
	return app, repo, distRepo
}

func generateTestToken(t *testing.T, role string) string {
	t.Helper()
	token, err := helper.GenerateToken(1, "tester@kurbantrack.local", role, nil)
	if err != nil {
		t.Fatalf("Gagal generate test token untuk role %s: %v", role, err)
	}
	return token
}

// ----------------------------------------------------------------------------
// TEST CASES: HEALTH & AUTH ENDPOINTS
// ----------------------------------------------------------------------------

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

func TestHTTP_Auth_RegisterAndLogin_Success(t *testing.T) {
	app, _ := setupTestApp()

	// 1. Register
	regPayload := model.UserCreateRequest{
		Name:     "Admin Masjid",
		Email:    "admin@masjid.local",
		Password: "password123",
		Role:     model.RoleAdmin,
	}
	regBody, _ := json.Marshal(regPayload)

	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(regBody))
	regReq.Header.Set("Content-Type", "application/json")

	regResp, err := app.Test(regReq, -1)
	if err != nil {
		t.Fatalf("Gagal request register: %v", err)
	}
	if regResp.StatusCode != http.StatusCreated {
		t.Fatalf("Ekspektasi status 201 Created saat register, didapat %d", regResp.StatusCode)
	}

	// 2. Login
	loginPayload := model.UserLoginRequest{
		Email:    "admin@masjid.local",
		Password: "password123",
	}
	loginBody, _ := json.Marshal(loginPayload)

	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")

	loginResp, err := app.Test(loginReq, -1)
	if err != nil {
		t.Fatalf("Gagal request login: %v", err)
	}
	if loginResp.StatusCode != http.StatusOK {
		t.Errorf("Ekspektasi status 200 OK saat login, didapat %d", loginResp.StatusCode)
	}
}

// ----------------------------------------------------------------------------
// TEST CASES: ANIMALS (DENGAN AUTH & RBAC)
// ----------------------------------------------------------------------------

func TestHTTP_CreateAnimal_Success(t *testing.T) {
	app, _ := setupTestApp()
	token := generateTestToken(t, model.RoleAdmin)

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
	req.Header.Set("Authorization", "Bearer "+token)

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
	token := generateTestToken(t, model.RoleAdmin)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/animals", bytes.NewReader([]byte("{invalid-json")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

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
	token := generateTestToken(t, model.RoleAdmin)

	payload := model.AnimalCreateRequest{
		Code:              "",
		Species:           "kucing",
		SohibulName:       "",
		WeightKg:          -5,
		EstimatedPackages: 0,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/animals", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

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
	token := generateTestToken(t, model.RoleAdmin)

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
	req.Header.Set("Authorization", "Bearer "+token)

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
	token := generateTestToken(t, model.RoleAdmin)
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
	req.Header.Set("Authorization", "Bearer "+token)

	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Ekspektasi status 500 Internal Server Error, didapat %d", resp.StatusCode)
	}
}

func TestHTTP_GetAllAnimals(t *testing.T) {
	app, repo := setupTestApp()
	token := generateTestToken(t, model.RolePanitiaDistribusi)

	repo.animals[1] = &model.Animal{ID: 1, Code: "SP-01", Species: model.SpeciesSapi, SohibulName: "Ahmad", WeightKg: 300, EstimatedPackages: 80, Status: model.AnimalStatusRegistered}
	repo.animals[2] = &model.Animal{ID: 2, Code: "KB-02", Species: model.SpeciesKambing, SohibulName: "Umar", WeightKg: 40, EstimatedPackages: 15, Status: model.AnimalStatusRegistered}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/animals", nil)
	req.Header.Set("Authorization", "Bearer "+token)

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
	token := generateTestToken(t, model.RoleKoordinatorWilayah)

	repo.animals[1] = &model.Animal{ID: 1, Code: "SP-01", Species: model.SpeciesSapi, SohibulName: "Ahmad", WeightKg: 300, EstimatedPackages: 80, Status: model.AnimalStatusRegistered}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/animals/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)

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
	token := generateTestToken(t, model.RoleAdmin)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/animals/999", nil)
	req.Header.Set("Authorization", "Bearer "+token)

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
	token := generateTestToken(t, model.RoleAdmin)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/animals/abc", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Ekspektasi status 400 Bad Request untuk ID non-integer, didapat %d", resp.StatusCode)
	}
}

func TestHTTP_UpdateAnimal_Success(t *testing.T) {
	app, repo := setupTestApp()
	token := generateTestToken(t, model.RoleAdmin)

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
	req.Header.Set("Authorization", "Bearer "+token)

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
	token := generateTestToken(t, model.RoleAdmin)

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
	req.Header.Set("Authorization", "Bearer "+token)

	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Ekspektasi status 404 Not Found, didapat %d", resp.StatusCode)
	}
}

func TestHTTP_DeleteAnimal_Success(t *testing.T) {
	app, repo := setupTestApp()
	token := generateTestToken(t, model.RoleAdmin)

	repo.animals[1] = &model.Animal{ID: 1, Code: "SP-01", Species: model.SpeciesSapi, SohibulName: "Ahmad", WeightKg: 300, EstimatedPackages: 80, Status: model.AnimalStatusRegistered}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/animals/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)

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
	token := generateTestToken(t, model.RoleAdmin)

	repo.animals[1] = &model.Animal{ID: 1, Code: "SP-01", Species: model.SpeciesSapi, SohibulName: "Ahmad", WeightKg: 300, EstimatedPackages: 80, Status: model.AnimalStatusRegistered}
	repo.activeAllocations[1] = 5

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/animals/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)

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
	token := generateTestToken(t, model.RoleAdmin)

	repo.animals[1] = &model.Animal{ID: 1, Code: "SP-01", Species: model.SpeciesSapi, SohibulName: "Ahmad", WeightKg: 300, EstimatedPackages: 80, Status: model.AnimalStatusRegistered}
	repo.deleteErr = repository.ErrForeignKeyViolation

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/animals/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)

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
	token := generateTestToken(t, model.RoleAdmin)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/animals/999", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Gagal request: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Ekspektasi status 404 Not Found saat ID tidak ditemukan, didapat %d", resp.StatusCode)
	}
}

// ----------------------------------------------------------------------------
// TEST CASES: DISTRIBUTIONS (DENGAN AUTH & RBAC)
// ----------------------------------------------------------------------------

func TestHTTP_Distribution_Create_Scaffold(t *testing.T) {
	app, _ := setupTestApp()
	token := generateTestToken(t, model.RoleKoordinatorWilayah)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/distributions", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Gagal request: %v", err)
	}

	if resp.StatusCode != http.StatusNotImplemented {
		t.Errorf("Ekspektasi status 501 Not Implemented untuk scaffold POST distributions, didapat %d", resp.StatusCode)
	}
}

func TestHTTP_Distribution_FindAll_Scaffold(t *testing.T) {
	app, _ := setupTestApp()
	token := generateTestToken(t, model.RolePanitiaDistribusi)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/distributions", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Gagal request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Ekspektasi status 200 OK untuk scaffold GET distributions, didapat %d", resp.StatusCode)
	}
}

func TestHTTP_Distribution_Redeem_Scaffold(t *testing.T) {
	app, _ := setupTestApp()
	token := generateTestToken(t, model.RolePanitiaDistribusi)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/distributions/CPN-123/redeem", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Gagal request: %v", err)
	}

	if resp.StatusCode != http.StatusNotImplemented {
		t.Errorf("Ekspektasi status 501 Not Implemented untuk scaffold PATCH redeem, didapat %d", resp.StatusCode)
	}
}

func TestHTTP_Distribution_Cancel_Scaffold(t *testing.T) {
	app, _ := setupTestApp()
	token := generateTestToken(t, model.RoleKoordinatorWilayah)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/distributions/CPN-123/cancel", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Gagal request: %v", err)
	}

	if resp.StatusCode != http.StatusNotImplemented {
		t.Errorf("Ekspektasi status 501 Not Implemented untuk scaffold PATCH cancel, didapat %d", resp.StatusCode)
	}
}
