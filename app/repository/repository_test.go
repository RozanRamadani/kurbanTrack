package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"kurbantrack-backend/app/model"
	"kurbantrack-backend/app/repository"
)

// MockAnimalRepository adalah mock sederhana pengujian interface AnimalRepository
type MockAnimalRepository struct {
	animals       map[int]*model.Animal
	distributions map[int]string // animalID -> status
	nextID        int
}

func NewMockAnimalRepository() *MockAnimalRepository {
	return &MockAnimalRepository{
		animals:       make(map[int]*model.Animal),
		distributions: make(map[int]string),
		nextID:        1,
	}
}

func (m *MockAnimalRepository) Create(ctx context.Context, animal *model.Animal) error {
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

func (m *MockAnimalRepository) FindByID(ctx context.Context, id int) (*model.Animal, error) {
	animal, ok := m.animals[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return animal, nil
}

func (m *MockAnimalRepository) FindAll(ctx context.Context) ([]model.Animal, error) {
	list := make([]model.Animal, 0, len(m.animals))
	for _, a := range m.animals {
		list = append(list, *a)
	}
	return list, nil
}

func (m *MockAnimalRepository) Update(ctx context.Context, animal *model.Animal) error {
	if _, ok := m.animals[animal.ID]; !ok {
		return repository.ErrNotFound
	}
	m.animals[animal.ID] = animal
	return nil
}

func (m *MockAnimalRepository) Delete(ctx context.Context, id int) error {
	if _, ok := m.animals[id]; !ok {
		return repository.ErrNotFound
	}
	delete(m.animals, id)
	return nil
}

func (m *MockAnimalRepository) CountAllocationsByAnimalID(ctx context.Context, animalID int) (int, error) {
	count := 0
	for aID, status := range m.distributions {
		if aID == animalID && status == model.DistributionStatusAllocated {
			count++
		}
	}
	return count, nil
}

// Verifikasi kepatuhan interface
var _ repository.AnimalRepository = (*MockAnimalRepository)(nil)

func TestAnimalRepository_MockBehavior(t *testing.T) {
	ctx := context.Background()
	repo := NewMockAnimalRepository()

	// 1. Uji Create
	animal := &model.Animal{
		Code:              "KB-001",
		Species:           model.SpeciesKambing,
		SohibulName:       "Budi",
		WeightKg:          45.5,
		EstimatedPackages: 15,
		AllocatedPackages: 0,
		Status:            model.AnimalStatusRegistered,
	}

	if err := repo.Create(ctx, animal); err != nil {
		t.Fatalf("Gagal create animal: %v", err)
	}
	if animal.ID != 1 {
		t.Errorf("Ekspektasi ID 1, didapat %d", animal.ID)
	}

	// 2. Uji Konflik Duplikat Kode
	duplicate := &model.Animal{
		Code:              "KB-001",
		Species:           model.SpeciesKambing,
		SohibulName:       "Siti",
		WeightKg:          50.0,
		EstimatedPackages: 18,
	}
	if err := repo.Create(ctx, duplicate); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("Ekspektasi ErrConflict pada duplikasi kode, didapat: %v", err)
	}

	// 3. Uji FindByID
	found, err := repo.FindByID(ctx, 1)
	if err != nil {
		t.Fatalf("Gagal find by ID: %v", err)
	}
	if found.Code != "KB-001" {
		t.Errorf("Ekspektasi code 'KB-001', didapat '%s'", found.Code)
	}

	// 4. Uji FindByID Not Found
	if _, err := repo.FindByID(ctx, 999); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("Ekspektasi ErrNotFound untuk ID non-existent, didapat: %v", err)
	}

	// 5. Uji Update
	found.WeightKg = 48.0
	if err := repo.Update(ctx, found); err != nil {
		t.Fatalf("Gagal update animal: %v", err)
	}

	// 6. Uji CountAllocationsByAnimalID (hanya alokasi aktif berstatus 'allocated' yang dihitung)
	repo.distributions[1] = model.DistributionStatusAllocated
	count, err := repo.CountAllocationsByAnimalID(ctx, 1)
	if err != nil {
		t.Fatalf("Gagal count allocations: %v", err)
	}
	if count != 1 {
		t.Errorf("Ekspektasi active allocations 1, didapat %d", count)
	}

	// Ubah status ke redeemed atau cancelled -> tidak boleh dihitung sebagai alokasi aktif
	repo.distributions[1] = model.DistributionStatusCancelled
	countCancelled, _ := repo.CountAllocationsByAnimalID(ctx, 1)
	if countCancelled != 0 {
		t.Errorf("Ekspektasi active allocations 0 saat cancelled, didapat %d", countCancelled)
	}

	// 7. Uji Delete
	if err := repo.Delete(ctx, 1); err != nil {
		t.Fatalf("Gagal delete animal: %v", err)
	}

	// Pastikan setelah di-delete tidak ditemukan
	if _, err := repo.FindByID(ctx, 1); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("Ekspektasi ErrNotFound setelah delete, didapat: %v", err)
	}
}
