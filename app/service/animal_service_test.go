package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"kurbantrack-backend/app/model"
	"kurbantrack-backend/app/repository"
	"kurbantrack-backend/app/service"
)

// mockAnimalRepo mengimplementasikan interface repository.AnimalRepository untuk pengujian unit service.
type mockAnimalRepo struct {
	animals           map[int]*model.Animal
	activeAllocations map[int]int
	nextID            int
	createErr         error
	findErr           error
	updateErr         error
	deleteErr         error
}

func newMockAnimalRepo() *mockAnimalRepo {
	return &mockAnimalRepo{
		animals:           make(map[int]*model.Animal),
		activeAllocations: make(map[int]int),
		nextID:            1,
	}
}

func (m *mockAnimalRepo) Create(ctx context.Context, animal *model.Animal) error {
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

func (m *mockAnimalRepo) FindByID(ctx context.Context, id int) (*model.Animal, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	animal, ok := m.animals[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	// Salin objek untuk menghindari modifikasi pointer langsung
	copied := *animal
	return &copied, nil
}

func (m *mockAnimalRepo) FindAll(ctx context.Context) ([]model.Animal, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	list := make([]model.Animal, 0, len(m.animals))
	for _, a := range m.animals {
		list = append(list, *a)
	}
	return list, nil
}

func (m *mockAnimalRepo) Update(ctx context.Context, animal *model.Animal) error {
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

func (m *mockAnimalRepo) Delete(ctx context.Context, id int) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if _, ok := m.animals[id]; !ok {
		return repository.ErrNotFound
	}
	delete(m.animals, id)
	return nil
}

func (m *mockAnimalRepo) CountAllocationsByAnimalID(ctx context.Context, animalID int) (int, error) {
	return m.activeAllocations[animalID], nil
}

// ----------------------------------------------------------------------------
// UNIT TESTS UNTUK ANIMAL SERVICE
// ----------------------------------------------------------------------------

func TestAnimalService_Create_Valid(t *testing.T) {
	repo := newMockAnimalRepo()
	svc := service.NewAnimalService(repo)
	ctx := context.Background()

	req := &model.AnimalCreateRequest{
		Code:              "SP-001",
		Species:           model.SpeciesSapi,
		SohibulName:       "Haji Ahmad",
		WeightKg:          350.50,
		EstimatedPackages: 100,
	}

	res, err := svc.CreateAnimal(ctx, req)
	if err != nil {
		t.Fatalf("Ekspektasi create berhasil, didapat error: %v", err)
	}
	if res.ID != 1 {
		t.Errorf("Ekspektasi ID 1, didapat %d", res.ID)
	}
	if res.Status != model.AnimalStatusRegistered {
		t.Errorf("Ekspektasi status awal 'registered', didapat '%s'", res.Status)
	}
	if res.AllocatedPackages != 0 {
		t.Errorf("Ekspektasi allocated_packages 0, didapat %d", res.AllocatedPackages)
	}
}

func TestAnimalService_Create_InvalidSpecies(t *testing.T) {
	repo := newMockAnimalRepo()
	svc := service.NewAnimalService(repo)
	ctx := context.Background()

	req := &model.AnimalCreateRequest{
		Code:              "AY-001",
		Species:           "ayam", // Spesies tidak didukung
		SohibulName:       "Haji Ahmad",
		WeightKg:          2.5,
		EstimatedPackages: 1,
	}

	_, err := svc.CreateAnimal(ctx, req)
	if err == nil {
		t.Fatal("Ekspektasi error validasi spesies, namun sukses")
	}

	var valErr *service.ValidationError
	if !errors.As(err, &valErr) || valErr.Field != "species" {
		t.Errorf("Ekspektasi ValidationError pada field 'species', didapat: %v", err)
	}
}

func TestAnimalService_Create_InvalidWeight_ZeroOrNegative(t *testing.T) {
	repo := newMockAnimalRepo()
	svc := service.NewAnimalService(repo)
	ctx := context.Background()

	testWeights := []float64{0, -10.5}
	for _, w := range testWeights {
		req := &model.AnimalCreateRequest{
			Code:              "SP-002",
			Species:           model.SpeciesSapi,
			SohibulName:       "Budi",
			WeightKg:          w,
			EstimatedPackages: 50,
		}

		_, err := svc.CreateAnimal(ctx, req)
		if err == nil {
			t.Fatalf("Ekspektasi error untuk weight %f, namun sukses", w)
		}
		var valErr *service.ValidationError
		if !errors.As(err, &valErr) || valErr.Field != "weight_kg" {
			t.Errorf("Ekspektasi ValidationError pada field 'weight_kg', didapat: %v", err)
		}
	}
}

func TestAnimalService_Create_Weight_ExceedsMax(t *testing.T) {
	repo := newMockAnimalRepo()
	svc := service.NewAnimalService(repo)
	ctx := context.Background()

	req := &model.AnimalCreateRequest{
		Code:              "SP-003",
		Species:           model.SpeciesSapi,
		SohibulName:       "Budi",
		WeightKg:          10000.00, // Melebihi batas 9999.99 kg
		EstimatedPackages: 50,
	}

	_, err := svc.CreateAnimal(ctx, req)
	if err == nil {
		t.Fatal("Ekspektasi error bobot melebihi batas, namun sukses")
	}
	var valErr *service.ValidationError
	if !errors.As(err, &valErr) || valErr.Field != "weight_kg" {
		t.Errorf("Ekspektasi ValidationError pada field 'weight_kg', didapat: %v", err)
	}
}

func TestAnimalService_Create_Weight_MoreThanTwoDecimals(t *testing.T) {
	repo := newMockAnimalRepo()
	svc := service.NewAnimalService(repo)
	ctx := context.Background()

	req := &model.AnimalCreateRequest{
		Code:              "SP-004",
		Species:           model.SpeciesSapi,
		SohibulName:       "Budi",
		WeightKg:          45.555, // Memiliki 3 angka desimal
		EstimatedPackages: 50,
	}

	_, err := svc.CreateAnimal(ctx, req)
	if err == nil {
		t.Fatal("Ekspektasi error desimal > 2, namun sukses")
	}
	var valErr *service.ValidationError
	if !errors.As(err, &valErr) || valErr.Field != "weight_kg" {
		t.Errorf("Ekspektasi ValidationError pada field 'weight_kg', didapat: %v", err)
	}
}

func TestAnimalService_Create_InvalidEstimatedPackages(t *testing.T) {
	repo := newMockAnimalRepo()
	svc := service.NewAnimalService(repo)
	ctx := context.Background()

	testPackages := []int{0, -5}
	for _, p := range testPackages {
		req := &model.AnimalCreateRequest{
			Code:              "SP-005",
			Species:           model.SpeciesSapi,
			SohibulName:       "Budi",
			WeightKg:          300.0,
			EstimatedPackages: p,
		}

		_, err := svc.CreateAnimal(ctx, req)
		if err == nil {
			t.Fatalf("Ekspektasi error untuk estimated_packages %d, namun sukses", p)
		}
		var valErr *service.ValidationError
		if !errors.As(err, &valErr) || valErr.Field != "estimated_packages" {
			t.Errorf("Ekspektasi ValidationError pada field 'estimated_packages', didapat: %v", err)
		}
	}
}

func TestAnimalService_Update_Valid(t *testing.T) {
	repo := newMockAnimalRepo()
	svc := service.NewAnimalService(repo)
	ctx := context.Background()

	// Buat hewan awal
	initial, _ := svc.CreateAnimal(ctx, &model.AnimalCreateRequest{
		Code:              "KB-001",
		Species:           model.SpeciesKambing,
		SohibulName:       "Siti",
		WeightKg:          40.0,
		EstimatedPackages: 20,
	})

	updateReq := &model.AnimalUpdateRequest{
		Code:              "KB-001-REV",
		Species:           model.SpeciesKambing,
		SohibulName:       "Siti Aminah",
		WeightKg:          42.50,
		EstimatedPackages: 22,
		Status:            model.AnimalStatusSlaughtered,
	}

	updated, err := svc.UpdateAnimal(ctx, initial.ID, updateReq)
	if err != nil {
		t.Fatalf("Ekspektasi update berhasil, didapat error: %v", err)
	}

	if updated.Code != "KB-001-REV" || updated.Status != model.AnimalStatusSlaughtered {
		t.Errorf("Data update tidak sesuai: %+v", updated)
	}
}

func TestAnimalService_Update_EstimatedPackages_BelowActiveAllocations(t *testing.T) {
	repo := newMockAnimalRepo()
	svc := service.NewAnimalService(repo)
	ctx := context.Background()

	// Buat hewan awal
	initial, _ := svc.CreateAnimal(ctx, &model.AnimalCreateRequest{
		Code:              "DM-001",
		Species:           model.SpeciesDomba,
		SohibulName:       "Farhan",
		WeightKg:          50.0,
		EstimatedPackages: 20,
	})

	// Simulasikan sudah ada 15 alokasi aktif
	repo.activeAllocations[initial.ID] = 15

	// Coba update estimated packages menjadi 10 (lebih kecil dari 15)
	updateReq := &model.AnimalUpdateRequest{
		Code:              "DM-001",
		Species:           model.SpeciesDomba,
		SohibulName:       "Farhan",
		WeightKg:          50.0,
		EstimatedPackages: 10,
	}

	_, err := svc.UpdateAnimal(ctx, initial.ID, updateReq)
	if err == nil {
		t.Fatal("Ekspektasi error saat estimated_packages di bawah alokasi aktif, namun sukses")
	}

	var valErr *service.ValidationError
	if !errors.As(err, &valErr) || valErr.Field != "estimated_packages" {
		t.Errorf("Ekspektasi ValidationError pada 'estimated_packages', didapat: %v", err)
	}
}

func TestAnimalService_Delete_WithActiveAllocations(t *testing.T) {
	repo := newMockAnimalRepo()
	svc := service.NewAnimalService(repo)
	ctx := context.Background()

	animal, _ := svc.CreateAnimal(ctx, &model.AnimalCreateRequest{
		Code:              "SP-010",
		Species:           model.SpeciesSapi,
		SohibulName:       "Rahmat",
		WeightKg:          400.0,
		EstimatedPackages: 120,
	})

	// Set terdapat alokasi aktif
	repo.activeAllocations[animal.ID] = 5

	err := svc.DeleteAnimal(ctx, animal.ID)
	if !errors.Is(err, service.ErrHasActiveAllocations) {
		t.Errorf("Ekspektasi ErrHasActiveAllocations, didapat: %v", err)
	}
}

func TestAnimalService_Delete_NotFound(t *testing.T) {
	repo := newMockAnimalRepo()
	svc := service.NewAnimalService(repo)
	ctx := context.Background()

	err := svc.DeleteAnimal(ctx, 9999)
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("Ekspektasi ErrNotFound saat delete ID tidak ada, didapat: %v", err)
	}
}

func TestAnimalService_FindByID_And_FindAll(t *testing.T) {
	repo := newMockAnimalRepo()
	svc := service.NewAnimalService(repo)
	ctx := context.Background()

	// 1. Uji FindByID Not Found
	_, err := svc.GetAnimalByID(ctx, 999)
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("Ekspektasi ErrNotFound, didapat: %v", err)
	}

	// 2. Tambah 2 data
	a1, _ := svc.CreateAnimal(ctx, &model.AnimalCreateRequest{
		Code:              "SP-101",
		Species:           model.SpeciesSapi,
		SohibulName:       "Ali",
		WeightKg:          300.0,
		EstimatedPackages: 80,
	})
	_, _ = svc.CreateAnimal(ctx, &model.AnimalCreateRequest{
		Code:              "KB-102",
		Species:           model.SpeciesKambing,
		SohibulName:       "Umar",
		WeightKg:          40.0,
		EstimatedPackages: 15,
	})

	// 3. Uji FindByID sukses
	found, err := svc.GetAnimalByID(ctx, a1.ID)
	if err != nil {
		t.Fatalf("Gagal GetAnimalByID: %v", err)
	}
	if found.Code != "SP-101" {
		t.Errorf("Ekspektasi code 'SP-101', didapat '%s'", found.Code)
	}

	// 4. Uji FindAll sukses
	all, err := svc.GetAllAnimals(ctx)
	if err != nil {
		t.Fatalf("Gagal GetAllAnimals: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("Ekspektasi 2 animals, didapat %d", len(all))
	}
}

func TestAnimalService_Create_DuplicateCode(t *testing.T) {
	repo := newMockAnimalRepo()
	svc := service.NewAnimalService(repo)
	ctx := context.Background()

	req := &model.AnimalCreateRequest{
		Code:              "SP-DUP",
		Species:           model.SpeciesSapi,
		SohibulName:       "Ahmad",
		WeightKg:          350.0,
		EstimatedPackages: 100,
	}

	_, err := svc.CreateAnimal(ctx, req)
	if err != nil {
		t.Fatalf("Create pertama gagal: %v", err)
	}

	// Coba buat dengan kode yang sama
	_, err = svc.CreateAnimal(ctx, req)
	if !errors.Is(err, service.ErrConflict) {
		t.Errorf("Ekspektasi ErrConflict pada duplikasi kode, didapat: %v", err)
	}
}

func TestAnimalService_Update_EstimatedPackages_BelowAllocatedPackages(t *testing.T) {
	repo := newMockAnimalRepo()
	svc := service.NewAnimalService(repo)
	ctx := context.Background()

	// Buat hewan
	a, _ := svc.CreateAnimal(ctx, &model.AnimalCreateRequest{
		Code:              "SP-ALLOC",
		Species:           model.SpeciesSapi,
		SohibulName:       "Siti",
		WeightKg:          300.0,
		EstimatedPackages: 50,
	})

	// Simulasikan di DB hewan ini sudah memiliki allocated_packages = 30
	repo.animals[a.ID].AllocatedPackages = 30

	// Coba update estimated_packages menjadi 25 (< 30)
	updateReq := &model.AnimalUpdateRequest{
		Code:              "SP-ALLOC",
		Species:           model.SpeciesSapi,
		SohibulName:       "Siti",
		WeightKg:          300.0,
		EstimatedPackages: 25,
	}

	_, err := svc.UpdateAnimal(ctx, a.ID, updateReq)
	if err == nil {
		t.Fatal("Ekspektasi error saat estimasi < allocated_packages, namun sukses")
	}

	var valErr *service.ValidationError
	if !errors.As(err, &valErr) || valErr.Field != "estimated_packages" {
		t.Errorf("Ekspektasi ValidationError pada 'estimated_packages', didapat: %v", err)
	}
}

func TestAnimalService_Update_InvalidStatus(t *testing.T) {
	repo := newMockAnimalRepo()
	svc := service.NewAnimalService(repo)
	ctx := context.Background()

	a, _ := svc.CreateAnimal(ctx, &model.AnimalCreateRequest{
		Code:              "SP-STAT",
		Species:           model.SpeciesSapi,
		SohibulName:       "Budi",
		WeightKg:          300.0,
		EstimatedPackages: 50,
	})

	// Coba update dengan status tidak valid
	updateReq := &model.AnimalUpdateRequest{
		Code:              "SP-STAT",
		Species:           model.SpeciesSapi,
		SohibulName:       "Budi",
		WeightKg:          300.0,
		EstimatedPackages: 50,
		Status:            "invalid_status",
	}

	_, err := svc.UpdateAnimal(ctx, a.ID, updateReq)
	if err == nil {
		t.Fatal("Ekspektasi error status tidak valid, namun sukses")
	}

	var valErr *service.ValidationError
	if !errors.As(err, &valErr) || valErr.Field != "status" {
		t.Errorf("Ekspektasi ValidationError pada 'status', didapat: %v", err)
	}
}

func TestAnimalService_Delete_ForeignKeyViolation(t *testing.T) {
	repo := newMockAnimalRepo()
	svc := service.NewAnimalService(repo)
	ctx := context.Background()

	a, _ := svc.CreateAnimal(ctx, &model.AnimalCreateRequest{
		Code:              "SP-FK",
		Species:           model.SpeciesSapi,
		SohibulName:       "Farhan",
		WeightKg:          350.0,
		EstimatedPackages: 80,
	})

	// Simulasikan repository mengembalikan ErrForeignKeyViolation saat delete
	repo.deleteErr = repository.ErrForeignKeyViolation

	err := svc.DeleteAnimal(ctx, a.ID)
	if !errors.Is(err, service.ErrForeignKeyViolation) {
		t.Errorf("Ekspektasi ErrForeignKeyViolation, didapat: %v", err)
	}
}

func TestAnimalService_RepositoryGenericError(t *testing.T) {
	repo := newMockAnimalRepo()
	svc := service.NewAnimalService(repo)
	ctx := context.Background()

	dbErr := errors.New("database connection refused")
	repo.createErr = dbErr

	_, err := svc.CreateAnimal(ctx, &model.AnimalCreateRequest{
		Code:              "SP-ERR",
		Species:           model.SpeciesSapi,
		SohibulName:       "Test",
		WeightKg:          200.0,
		EstimatedPackages: 50,
	})

	if !errors.Is(err, dbErr) {
		t.Errorf("Ekspektasi generic database error diteruskan, didapat: %v", err)
	}
}

func TestAnimalRules_Weight_PrecisionAudit(t *testing.T) {
	// Audit nilai khusus: 45.50, 45.55, 45.555, 9999.99, dan 10000
	cases := []struct {
		weight  float64
		wantErr bool
	}{
		{weight: 45.50, wantErr: false},   // Valid (2 desimal)
		{weight: 45.55, wantErr: false},   // Valid (2 desimal)
		{weight: 45.555, wantErr: true},   // Invalid (3 desimal)
		{weight: 9999.99, wantErr: false}, // Valid (maksimum bobot)
		{weight: 10000.00, wantErr: true}, // Invalid (melebihi batas 9999.99)
	}

	for _, tc := range cases {
		req := &model.AnimalCreateRequest{
			Code:              "SP-PREC",
			Species:           model.SpeciesSapi,
			SohibulName:       "Audit",
			WeightKg:          tc.weight,
			EstimatedPackages: 50,
		}

		err := service.ValidateAnimalCreate(req)
		if tc.wantErr && err == nil {
			t.Errorf("Bobot %f seharusnya gagal validasi, namun berhasil", tc.weight)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("Bobot %f seharusnya valid, namun gagal: %v", tc.weight, err)
		}
	}
}
