package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"kurbantrack-backend/app/model"
	"kurbantrack-backend/app/repository"
	"kurbantrack-backend/app/service"
)

// mockDistributionRepo mengimplementasikan kontrak service.DistributionRepository
// beserta perilaku transaksional atomik untuk pengujian unit DistributionService.
type mockDistributionRepo struct {
	distributions map[int]*model.Distribution
	logs          []model.DistributionLog
	animalRepo    *mockAnimalRepo
	nextID        int
	nextLogID     int
	findErr       error
	allocateTxErr error
	redeemTxErr   error
	cancelTxErr   error
}

func newMockDistributionRepo(animalRepo *mockAnimalRepo) *mockDistributionRepo {
	return &mockDistributionRepo{
		distributions: make(map[int]*model.Distribution),
		logs:          make([]model.DistributionLog, 0),
		animalRepo:    animalRepo,
		nextID:        1,
		nextLogID:     1,
	}
}

func (m *mockDistributionRepo) Create(ctx context.Context, dist *model.Distribution) error {
	return m.AllocateWithQuotaTx(ctx, dist)
}

func (m *mockDistributionRepo) FindByID(ctx context.Context, id int) (*model.Distribution, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	d, ok := m.distributions[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	copied := *d
	return &copied, nil
}

func (m *mockDistributionRepo) FindByCouponCode(ctx context.Context, couponCode string) (*model.Distribution, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	for _, d := range m.distributions {
		if d.CouponCode == couponCode {
			copied := *d
			return &copied, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (m *mockDistributionRepo) FindAll(ctx context.Context) ([]model.Distribution, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	list := make([]model.Distribution, 0, len(m.distributions))
	for _, d := range m.distributions {
		list = append(list, *d)
	}
	return list, nil
}

func (m *mockDistributionRepo) UpdateStatus(ctx context.Context, id int, status string, redeemedAt *time.Time) error {
	d, ok := m.distributions[id]
	if !ok {
		return repository.ErrNotFound
	}
	d.Status = status
	d.RedeemedAt = redeemedAt
	return nil
}

// AllocateWithQuotaTx menyimulasikan transaksi atomik alokasi kupon:
// pengecekan ulang status & kuota hewan, increment allocated_packages, insert distribution, dan insert distribution_log.
func (m *mockDistributionRepo) AllocateWithQuotaTx(ctx context.Context, dist *model.Distribution) error {
	if m.allocateTxErr != nil {
		return m.allocateTxErr
	}

	// Cek duplikasi coupon_code atau recipient_nik sesuai constraint UNIQUE database
	for _, existing := range m.distributions {
		if existing.CouponCode == dist.CouponCode || existing.RecipientNIK == dist.RecipientNIK {
			return repository.ErrConflict
		}
	}

	// Pengecekan atomik hewan & kuota di dalam transaksi
	animal, ok := m.animalRepo.animals[dist.AnimalID]
	if !ok {
		return repository.ErrNotFound
	}
	if animal.Status != model.AnimalStatusSlaughtered || animal.AllocatedPackages >= animal.EstimatedPackages {
		return service.ErrQuotaExceeded
	}

	// Increment allocated_packages, insert distribution, dan insert distribution_log dalam satu kesatuan
	animal.AllocatedPackages++
	m.animalRepo.activeAllocations[animal.ID]++

	dist.ID = m.nextID
	m.nextID++
	dist.CreatedAt = time.Now()
	if dist.Status == "" {
		dist.Status = model.DistributionStatusAllocated
	}

	stored := *dist
	m.distributions[dist.ID] = &stored

	m.logs = append(m.logs, model.DistributionLog{
		ID:             m.nextLogID,
		DistributionID: dist.ID,
		ActorID:        dist.CoordinatorID,
		FromStatus:     nil,
		ToStatus:       model.DistributionStatusAllocated,
		CreatedAt:      dist.CreatedAt,
	})
	m.nextLogID++

	return nil
}

// RedeemCouponTx menyimulasikan transaksi atomik penebusan kupon:
// conditional update status (WHERE status = 'allocated') dan insert distribution_log.
func (m *mockDistributionRepo) RedeemCouponTx(ctx context.Context, couponCode string, redeemedAt time.Time, actorID int) (*model.Distribution, error) {
	if m.redeemTxErr != nil {
		return nil, m.redeemTxErr
	}

	var target *model.Distribution
	for _, d := range m.distributions {
		if d.CouponCode == couponCode {
			target = d
			break
		}
	}
	if target == nil {
		return nil, repository.ErrNotFound
	}
	if target.Status != model.DistributionStatusAllocated {
		return nil, repository.ErrConflict
	}

	fromStatus := target.Status
	target.Status = model.DistributionStatusRedeemed
	rTime := redeemedAt
	target.RedeemedAt = &rTime

	if m.animalRepo.activeAllocations[target.AnimalID] > 0 {
		m.animalRepo.activeAllocations[target.AnimalID]--
	}

	m.logs = append(m.logs, model.DistributionLog{
		ID:             m.nextLogID,
		DistributionID: target.ID,
		ActorID:        actorID,
		FromStatus:     &fromStatus,
		ToStatus:       model.DistributionStatusRedeemed,
		CreatedAt:      redeemedAt,
	})
	m.nextLogID++

	copied := *target
	return &copied, nil
}

// CancelCouponTx menyimulasikan transaksi atomik pembatalan kupon:
// conditional update status (WHERE status = 'allocated'), decrement allocated_packages, dan insert distribution_log.
func (m *mockDistributionRepo) CancelCouponTx(ctx context.Context, couponCode string, actorID int) (*model.Distribution, error) {
	if m.cancelTxErr != nil {
		return nil, m.cancelTxErr
	}

	var target *model.Distribution
	for _, d := range m.distributions {
		if d.CouponCode == couponCode {
			target = d
			break
		}
	}
	if target == nil {
		return nil, repository.ErrNotFound
	}
	if target.Status != model.DistributionStatusAllocated {
		return nil, repository.ErrConflict
	}

	fromStatus := target.Status
	target.Status = model.DistributionStatusCancelled
	target.RedeemedAt = nil

	if animal, ok := m.animalRepo.animals[target.AnimalID]; ok && animal.AllocatedPackages > 0 {
		animal.AllocatedPackages--
	}
	if m.animalRepo.activeAllocations[target.AnimalID] > 0 {
		m.animalRepo.activeAllocations[target.AnimalID]--
	}

	m.logs = append(m.logs, model.DistributionLog{
		ID:             m.nextLogID,
		DistributionID: target.ID,
		ActorID:        actorID,
		FromStatus:     &fromStatus,
		ToStatus:       model.DistributionStatusCancelled,
		CreatedAt:      time.Now(),
	})
	m.nextLogID++

	copied := *target
	return &copied, nil
}

var _ service.DistributionRepository = (*mockDistributionRepo)(nil)

func setupDistributionServiceTest() (service.DistributionService, *mockDistributionRepo, *mockAnimalRepo) {
	animalRepo := newMockAnimalRepo()
	distRepo := newMockDistributionRepo(animalRepo)
	svc := service.NewDistributionService(distRepo, animalRepo)
	return svc, distRepo, animalRepo
}

func seedSlaughteredAnimal(animalRepo *mockAnimalRepo, estimated, allocated int) *model.Animal {
	animal := &model.Animal{
		ID:                animalRepo.nextID,
		Code:              "SP-DIST-01",
		Species:           model.SpeciesSapi,
		SohibulName:       "Haji Sulaiman",
		WeightKg:          350.0,
		EstimatedPackages: estimated,
		AllocatedPackages: allocated,
		Status:            model.AnimalStatusSlaughtered,
		CreatedAt:         time.Now(),
	}
	animalRepo.nextID++
	animalRepo.animals[animal.ID] = animal
	return animal
}

// ----------------------------------------------------------------------------
// UNIT TESTS: CREATE DISTRIBUTION
// ----------------------------------------------------------------------------

func TestDistributionService_Create_Valid(t *testing.T) {
	svc, distRepo, animalRepo := setupDistributionServiceTest()
	ctx := context.Background()

	animal := seedSlaughteredAnimal(animalRepo, 10, 0)

	req := &model.DistributionCreateRequest{
		AnimalID:      animal.ID,
		CoordinatorID: 2,
		RecipientNIK:  "3201012345670001",
		RecipientName: "Pak RT Budi",
		RecipientRT:   "RT 01",
	}

	res, err := svc.CreateDistribution(ctx, req)
	if err != nil {
		t.Fatalf("Ekspektasi alokasi berhasil, didapat error: %v", err)
	}
	if res.ID != 1 {
		t.Errorf("Ekspektasi ID 1, didapat %d", res.ID)
	}
	if res.Status != model.DistributionStatusAllocated {
		t.Errorf("Ekspektasi status 'allocated', didapat '%s'", res.Status)
	}
	if res.CouponCode == "" || len(res.CouponCode) > 30 {
		t.Errorf("Ekspektasi coupon_code terisi dan <= 30 karakter, didapat '%s' (len=%d)", res.CouponCode, len(res.CouponCode))
	}
	if animalRepo.animals[animal.ID].AllocatedPackages != 1 {
		t.Errorf("Ekspektasi allocated_packages bertambah menjadi 1, didapat %d", animalRepo.animals[animal.ID].AllocatedPackages)
	}
	if len(distRepo.logs) != 1 || distRepo.logs[0].ActorID != 2 || distRepo.logs[0].ToStatus != model.DistributionStatusAllocated {
		t.Errorf("Ekspektasi distribution_logs tercatat untuk actor 2 dengan status 'allocated', didapat: %+v", distRepo.logs)
	}
}

func TestDistributionService_Create_InvalidAnimalID(t *testing.T) {
	svc, _, _ := setupDistributionServiceTest()
	ctx := context.Background()

	for _, invalidID := range []int{0, -5} {
		req := &model.DistributionCreateRequest{
			AnimalID:      invalidID,
			CoordinatorID: 1,
			RecipientNIK:  "3201012345670001",
			RecipientName: "Warga",
			RecipientRT:   "01",
		}

		_, err := svc.CreateDistribution(ctx, req)
		if err == nil {
			t.Fatalf("Ekspektasi error untuk animal_id %d, namun sukses", invalidID)
		}
		var valErr *service.ValidationError
		if !errors.As(err, &valErr) || valErr.Field != "animal_id" {
			t.Errorf("Ekspektasi ValidationError pada field 'animal_id', didapat: %v", err)
		}
	}
}

func TestDistributionService_Create_InvalidCoordinatorID(t *testing.T) {
	svc, _, _ := setupDistributionServiceTest()
	ctx := context.Background()

	for _, invalidCoordID := range []int{0, -2} {
		req := &model.DistributionCreateRequest{
			AnimalID:      1,
			CoordinatorID: invalidCoordID,
			RecipientNIK:  "3201012345670001",
			RecipientName: "Warga",
			RecipientRT:   "01",
		}

		_, err := svc.CreateDistribution(ctx, req)
		if err == nil {
			t.Fatalf("Ekspektasi error untuk coordinator_id %d, namun sukses", invalidCoordID)
		}
		var valErr *service.ValidationError
		if !errors.As(err, &valErr) || valErr.Field != "coordinator_id" {
			t.Errorf("Ekspektasi ValidationError pada field 'coordinator_id', didapat: %v", err)
		}
	}
}

func TestDistributionService_Create_InvalidNIK(t *testing.T) {
	svc, _, _ := setupDistributionServiceTest()
	ctx := context.Background()

	invalidNIKs := []string{
		"",
		"   ",
		"12345",
		"12345678901234567",
		"320101234567000A",
	}

	for _, nik := range invalidNIKs {
		req := &model.DistributionCreateRequest{
			AnimalID:      1,
			CoordinatorID: 1,
			RecipientNIK:  nik,
			RecipientName: "Warga",
			RecipientRT:   "01",
		}

		_, err := svc.CreateDistribution(ctx, req)
		if err == nil {
			t.Fatalf("Ekspektasi error untuk NIK '%s', namun sukses", nik)
		}
		var valErr *service.ValidationError
		if !errors.As(err, &valErr) || valErr.Field != "recipient_nik" {
			t.Errorf("Ekspektasi ValidationError pada field 'recipient_nik' untuk NIK '%s', didapat: %v", nik, err)
		}
	}
}

func TestDistributionService_Create_EmptyRecipientNameAndRT(t *testing.T) {
	svc, _, _ := setupDistributionServiceTest()
	ctx := context.Background()

	// Nama kosong
	_, err := svc.CreateDistribution(ctx, &model.DistributionCreateRequest{
		AnimalID:      1,
		CoordinatorID: 1,
		RecipientNIK:  "3201012345670001",
		RecipientName: "   ",
		RecipientRT:   "01",
	})
	var valErr *service.ValidationError
	if !errors.As(err, &valErr) || valErr.Field != "recipient_name" {
		t.Errorf("Ekspektasi ValidationError pada 'recipient_name', didapat: %v", err)
	}

	// RT kosong
	_, err = svc.CreateDistribution(ctx, &model.DistributionCreateRequest{
		AnimalID:      1,
		CoordinatorID: 1,
		RecipientNIK:  "3201012345670001",
		RecipientName: "Ahmad",
		RecipientRT:   "   ",
	})
	if !errors.As(err, &valErr) || valErr.Field != "recipient_rt" {
		t.Errorf("Ekspektasi ValidationError pada 'recipient_rt', didapat: %v", err)
	}
}

func TestDistributionService_Create_AnimalNotFound(t *testing.T) {
	svc, _, _ := setupDistributionServiceTest()
	ctx := context.Background()

	req := &model.DistributionCreateRequest{
		AnimalID:      999,
		CoordinatorID: 1,
		RecipientNIK:  "3201012345670001",
		RecipientName: "Ahmad",
		RecipientRT:   "01",
	}

	_, err := svc.CreateDistribution(ctx, req)
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("Ekspektasi ErrNotFound ketika hewan tidak ada, didapat: %v", err)
	}
}

func TestDistributionService_Create_AnimalStatusNotSlaughtered(t *testing.T) {
	svc, _, animalRepo := setupDistributionServiceTest()
	ctx := context.Background()

	for _, disallowedStatus := range []string{model.AnimalStatusRegistered, model.AnimalStatusDistributed} {
		animal := seedSlaughteredAnimal(animalRepo, 20, 0)
		animal.Status = disallowedStatus

		req := &model.DistributionCreateRequest{
			AnimalID:      animal.ID,
			CoordinatorID: 1,
			RecipientNIK:  "3201012345670001",
			RecipientName: "Ahmad",
			RecipientRT:   "01",
		}

		_, err := svc.CreateDistribution(ctx, req)
		if err == nil {
			t.Fatalf("Ekspektasi error untuk status hewan '%s', namun sukses", disallowedStatus)
		}
		var valErr *service.ValidationError
		if !errors.As(err, &valErr) || valErr.Field != "animal_id" {
			t.Errorf("Ekspektasi ValidationError pada 'animal_id' untuk status '%s', didapat: %v", disallowedStatus, err)
		}
	}
}

func TestDistributionService_Create_QuotaExhausted(t *testing.T) {
	svc, _, animalRepo := setupDistributionServiceTest()
	ctx := context.Background()

	animal := seedSlaughteredAnimal(animalRepo, 10, 10) // Sisa kuota = 0

	req := &model.DistributionCreateRequest{
		AnimalID:      animal.ID,
		CoordinatorID: 1,
		RecipientNIK:  "3201012345670001",
		RecipientName: "Ahmad",
		RecipientRT:   "01",
	}

	_, err := svc.CreateDistribution(ctx, req)
	if err == nil {
		t.Fatal("Ekspektasi error saat kuota habis, namun sukses")
	}
	var valErr *service.ValidationError
	if !errors.As(err, &valErr) || valErr.Field != "animal_id" {
		t.Errorf("Ekspektasi ValidationError pada 'animal_id' saat kuota habis, didapat: %v", err)
	}
}

func TestDistributionService_Create_DuplicateConflict(t *testing.T) {
	svc, _, animalRepo := setupDistributionServiceTest()
	ctx := context.Background()

	animal := seedSlaughteredAnimal(animalRepo, 10, 0)

	req := &model.DistributionCreateRequest{
		AnimalID:      animal.ID,
		CoordinatorID: 1,
		RecipientNIK:  "3201012345670001",
		RecipientName: "Ahmad",
		RecipientRT:   "01",
	}

	if _, err := svc.CreateDistribution(ctx, req); err != nil {
		t.Fatalf("Alokasi pertama gagal: %v", err)
	}

	// Alokasi kedua dengan NIK sama memicu repository.ErrConflict -> service.ErrConflict
	_, err := svc.CreateDistribution(ctx, req)
	if !errors.Is(err, service.ErrConflict) {
		t.Errorf("Ekspektasi ErrConflict pada duplikasi NIK/kode kupon, didapat: %v", err)
	}
}

func TestDistributionService_Create_RepositoryGenericError(t *testing.T) {
	svc, distRepo, animalRepo := setupDistributionServiceTest()
	ctx := context.Background()

	animal := seedSlaughteredAnimal(animalRepo, 10, 0)
	req := &model.DistributionCreateRequest{
		AnimalID:      animal.ID,
		CoordinatorID: 1,
		RecipientNIK:  "3201012345670001",
		RecipientName: "Ahmad",
		RecipientRT:   "01",
	}

	// 1. Error dari AllocateWithQuotaTx
	txErr := errors.New("tx commit failed")
	distRepo.allocateTxErr = txErr
	if _, err := svc.CreateDistribution(ctx, req); !errors.Is(err, txErr) {
		t.Errorf("Ekspektasi generic error dari AllocateWithQuotaTx diteruskan, didapat: %v", err)
	}

	// 2. Error dari AnimalRepository.FindByID
	distRepo.allocateTxErr = nil
	findErr := errors.New("db pool exhausted")
	animalRepo.findErr = findErr
	if _, err := svc.CreateDistribution(ctx, req); !errors.Is(err, findErr) {
		t.Errorf("Ekspektasi generic error dari AnimalRepository.FindByID diteruskan, didapat: %v", err)
	}
}

// ----------------------------------------------------------------------------
// UNIT TESTS: REDEEM DISTRIBUTION
// ----------------------------------------------------------------------------

func TestDistributionService_Redeem_Success(t *testing.T) {
	svc, distRepo, animalRepo := setupDistributionServiceTest()
	ctx := context.Background()

	animal := seedSlaughteredAnimal(animalRepo, 5, 0)
	created, err := svc.CreateDistribution(ctx, &model.DistributionCreateRequest{
		AnimalID:      animal.ID,
		CoordinatorID: 2,
		RecipientNIK:  "3201012345670002",
		RecipientName: "Siti",
		RecipientRT:   "02",
	})
	if err != nil {
		t.Fatalf("Gagal setup kupon: %v", err)
	}

	redeemed, err := svc.RedeemDistribution(ctx, created.CouponCode, 3)
	if err != nil {
		t.Fatalf("Ekspektasi redeem berhasil, didapat error: %v", err)
	}
	if redeemed.Status != model.DistributionStatusRedeemed {
		t.Errorf("Ekspektasi status 'redeemed', didapat '%s'", redeemed.Status)
	}
	if redeemed.RedeemedAt == nil {
		t.Error("Ekspektasi RedeemedAt terisi, didapat nil")
	}
	// Kuota paket yang teralokasi tidak boleh berkurang ketika kupon ditebus
	if animalRepo.animals[animal.ID].AllocatedPackages != 1 {
		t.Errorf("Ekspektasi allocated_packages tetap 1, didapat %d", animalRepo.animals[animal.ID].AllocatedPackages)
	}
	if len(distRepo.logs) != 2 || distRepo.logs[1].ActorID != 3 || distRepo.logs[1].ToStatus != model.DistributionStatusRedeemed {
		t.Errorf("Ekspektasi log redeem tercatat untuk actor 3, didapat: %+v", distRepo.logs)
	}
}

func TestDistributionService_Redeem_NotFound(t *testing.T) {
	svc, _, _ := setupDistributionServiceTest()
	ctx := context.Background()

	_, err := svc.RedeemDistribution(ctx, "KBN-NOTFOUND", 1)
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("Ekspektasi ErrNotFound, didapat: %v", err)
	}
}

func TestDistributionService_Redeem_AlreadyRedeemed(t *testing.T) {
	svc, _, animalRepo := setupDistributionServiceTest()
	ctx := context.Background()

	animal := seedSlaughteredAnimal(animalRepo, 5, 0)
	created, _ := svc.CreateDistribution(ctx, &model.DistributionCreateRequest{
		AnimalID:      animal.ID,
		CoordinatorID: 2,
		RecipientNIK:  "3201012345670003",
		RecipientName: "Umar",
		RecipientRT:   "03",
	})

	_, _ = svc.RedeemDistribution(ctx, created.CouponCode, 3)

	// Coba tebus kembali kupon yang sudah redeemed
	_, err := svc.RedeemDistribution(ctx, created.CouponCode, 3)
	var valErr *service.ValidationError
	if !errors.As(err, &valErr) || valErr.Field != "status" {
		t.Errorf("Ekspektasi ValidationError pada field 'status' saat redeem ulang, didapat: %v", err)
	}
}

func TestDistributionService_Redeem_CancelledCoupon(t *testing.T) {
	svc, _, animalRepo := setupDistributionServiceTest()
	ctx := context.Background()

	animal := seedSlaughteredAnimal(animalRepo, 5, 0)
	created, _ := svc.CreateDistribution(ctx, &model.DistributionCreateRequest{
		AnimalID:      animal.ID,
		CoordinatorID: 2,
		RecipientNIK:  "3201012345670004",
		RecipientName: "Ali",
		RecipientRT:   "04",
	})

	_, _ = svc.CancelDistribution(ctx, created.CouponCode, 2)

	// Coba tebus kupon yang sudah dibatalkan
	_, err := svc.RedeemDistribution(ctx, created.CouponCode, 3)
	var valErr *service.ValidationError
	if !errors.As(err, &valErr) || valErr.Field != "status" {
		t.Errorf("Ekspektasi ValidationError pada field 'status' saat redeem kupon cancelled, didapat: %v", err)
	}
}

func TestDistributionService_Redeem_InvalidActorID(t *testing.T) {
	svc, _, _ := setupDistributionServiceTest()
	ctx := context.Background()

	_, err := svc.RedeemDistribution(ctx, "KBN-123", 0)
	var valErr *service.ValidationError
	if !errors.As(err, &valErr) || valErr.Field != "actor_id" {
		t.Errorf("Ekspektasi ValidationError pada field 'actor_id', didapat: %v", err)
	}
}

func TestDistributionService_Redeem_RepositoryGenericError(t *testing.T) {
	svc, distRepo, animalRepo := setupDistributionServiceTest()
	ctx := context.Background()

	animal := seedSlaughteredAnimal(animalRepo, 5, 0)
	created, _ := svc.CreateDistribution(ctx, &model.DistributionCreateRequest{
		AnimalID:      animal.ID,
		CoordinatorID: 2,
		RecipientNIK:  "3201012345670005",
		RecipientName: "Hasan",
		RecipientRT:   "05",
	})

	dbErr := errors.New("redeem transaction failure")
	distRepo.redeemTxErr = dbErr

	_, err := svc.RedeemDistribution(ctx, created.CouponCode, 3)
	if !errors.Is(err, dbErr) {
		t.Errorf("Ekspektasi generic error dari RedeemCouponTx diteruskan, didapat: %v", err)
	}
}

// ----------------------------------------------------------------------------
// UNIT TESTS: CANCEL DISTRIBUTION
// ----------------------------------------------------------------------------

func TestDistributionService_Cancel_Success(t *testing.T) {
	svc, distRepo, animalRepo := setupDistributionServiceTest()
	ctx := context.Background()

	animal := seedSlaughteredAnimal(animalRepo, 5, 0)
	created, err := svc.CreateDistribution(ctx, &model.DistributionCreateRequest{
		AnimalID:      animal.ID,
		CoordinatorID: 2,
		RecipientNIK:  "3201012345670006",
		RecipientName: "Zainab",
		RecipientRT:   "06",
	})
	if err != nil {
		t.Fatalf("Gagal setup kupon: %v", err)
	}
	if animalRepo.animals[animal.ID].AllocatedPackages != 1 {
		t.Fatalf("Ekspektasi allocated_packages = 1 sebelum cancel, didapat %d", animalRepo.animals[animal.ID].AllocatedPackages)
	}

	cancelled, err := svc.CancelDistribution(ctx, created.CouponCode, 1)
	if err != nil {
		t.Fatalf("Ekspektasi cancel berhasil, didapat error: %v", err)
	}
	if cancelled.Status != model.DistributionStatusCancelled {
		t.Errorf("Ekspektasi status 'cancelled', didapat '%s'", cancelled.Status)
	}
	if cancelled.RedeemedAt != nil {
		t.Errorf("Ekspektasi RedeemedAt tetap nil, didapat %v", cancelled.RedeemedAt)
	}
	// Kuota harus dikembalikan sebanyak 1 paket
	if animalRepo.animals[animal.ID].AllocatedPackages != 0 {
		t.Errorf("Ekspektasi allocated_packages kembali menjadi 0 setelah cancel, didapat %d", animalRepo.animals[animal.ID].AllocatedPackages)
	}
	if len(distRepo.logs) != 2 || distRepo.logs[1].ActorID != 1 || distRepo.logs[1].ToStatus != model.DistributionStatusCancelled {
		t.Errorf("Ekspektasi log cancel tercatat untuk actor 1, didapat: %+v", distRepo.logs)
	}
}

func TestDistributionService_Cancel_NotFound(t *testing.T) {
	svc, _, _ := setupDistributionServiceTest()
	ctx := context.Background()

	_, err := svc.CancelDistribution(ctx, "KBN-MISSING", 1)
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("Ekspektasi ErrNotFound, didapat: %v", err)
	}
}

func TestDistributionService_Cancel_AlreadyRedeemed(t *testing.T) {
	svc, _, animalRepo := setupDistributionServiceTest()
	ctx := context.Background()

	animal := seedSlaughteredAnimal(animalRepo, 5, 0)
	created, _ := svc.CreateDistribution(ctx, &model.DistributionCreateRequest{
		AnimalID:      animal.ID,
		CoordinatorID: 2,
		RecipientNIK:  "3201012345670007",
		RecipientName: "Fatimah",
		RecipientRT:   "07",
	})

	_, _ = svc.RedeemDistribution(ctx, created.CouponCode, 3)

	// Coba batalkan kupon yang sudah redeemed
	_, err := svc.CancelDistribution(ctx, created.CouponCode, 1)
	var valErr *service.ValidationError
	if !errors.As(err, &valErr) || valErr.Field != "status" {
		t.Errorf("Ekspektasi ValidationError pada field 'status' saat membatalkan kupon redeemed, didapat: %v", err)
	}
}

func TestDistributionService_Cancel_AlreadyCancelled(t *testing.T) {
	svc, _, animalRepo := setupDistributionServiceTest()
	ctx := context.Background()

	animal := seedSlaughteredAnimal(animalRepo, 5, 0)
	created, _ := svc.CreateDistribution(ctx, &model.DistributionCreateRequest{
		AnimalID:      animal.ID,
		CoordinatorID: 2,
		RecipientNIK:  "3201012345670008",
		RecipientName: "Hamzah",
		RecipientRT:   "08",
	})

	_, _ = svc.CancelDistribution(ctx, created.CouponCode, 1)

	// Coba batalkan kembali kupon yang sudah cancelled
	_, err := svc.CancelDistribution(ctx, created.CouponCode, 1)
	var valErr *service.ValidationError
	if !errors.As(err, &valErr) || valErr.Field != "status" {
		t.Errorf("Ekspektasi ValidationError pada field 'status' saat cancel ulang, didapat: %v", err)
	}
}

func TestDistributionService_Cancel_InvalidActorID(t *testing.T) {
	svc, _, _ := setupDistributionServiceTest()
	ctx := context.Background()

	_, err := svc.CancelDistribution(ctx, "KBN-123", -1)
	var valErr *service.ValidationError
	if !errors.As(err, &valErr) || valErr.Field != "actor_id" {
		t.Errorf("Ekspektasi ValidationError pada field 'actor_id', didapat: %v", err)
	}
}

func TestDistributionService_Cancel_RepositoryGenericError(t *testing.T) {
	svc, distRepo, animalRepo := setupDistributionServiceTest()
	ctx := context.Background()

	animal := seedSlaughteredAnimal(animalRepo, 5, 0)
	created, _ := svc.CreateDistribution(ctx, &model.DistributionCreateRequest{
		AnimalID:      animal.ID,
		CoordinatorID: 2,
		RecipientNIK:  "3201012345670009",
		RecipientName: "Bilal",
		RecipientRT:   "09",
	})

	dbErr := errors.New("cancel transaction failure")
	distRepo.cancelTxErr = dbErr

	_, err := svc.CancelDistribution(ctx, created.CouponCode, 1)
	if !errors.Is(err, dbErr) {
		t.Errorf("Ekspektasi generic error dari CancelCouponTx diteruskan, didapat: %v", err)
	}
}

// ----------------------------------------------------------------------------
// UNIT TESTS: GET ALL DISTRIBUTIONS
// ----------------------------------------------------------------------------

func TestDistributionService_GetAll_SuccessAndError(t *testing.T) {
	svc, distRepo, animalRepo := setupDistributionServiceTest()
	ctx := context.Background()

	animal := seedSlaughteredAnimal(animalRepo, 5, 0)
	_, _ = svc.CreateDistribution(ctx, &model.DistributionCreateRequest{
		AnimalID:      animal.ID,
		CoordinatorID: 2,
		RecipientNIK:  "3201012345670010",
		RecipientName: "Warga 1",
		RecipientRT:   "01",
	})
	_, _ = svc.CreateDistribution(ctx, &model.DistributionCreateRequest{
		AnimalID:      animal.ID,
		CoordinatorID: 2,
		RecipientNIK:  "3201012345670011",
		RecipientName: "Warga 2",
		RecipientRT:   "02",
	})

	list, err := svc.GetAllDistributions(ctx)
	if err != nil {
		t.Fatalf("Ekspektasi GetAllDistributions berhasil, didapat error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("Ekspektasi 2 distribusi, didapat %d", len(list))
	}

	// Simulasikan error repository
	expectedErr := errors.New("query error")
	distRepo.findErr = expectedErr
	if _, err := svc.GetAllDistributions(ctx); !errors.Is(err, expectedErr) {
		t.Errorf("Ekspektasi error %v, didapat %v", expectedErr, err)
	}
}

// ----------------------------------------------------------------------------
// UNIT TESTS: DISTRIBUTION RULES DIRECT VALIDATION
// ----------------------------------------------------------------------------

func TestDistributionRules_ValidationDetails(t *testing.T) {
	// 1. Request valid
	validReq := &model.DistributionCreateRequest{
		AnimalID:      1,
		CoordinatorID: 1,
		RecipientNIK:  "3201012345670099",
		RecipientName: "Penerima Valid",
		RecipientRT:   "RT 05",
	}
	if err := service.ValidateDistributionCreate(validReq); err != nil {
		t.Errorf("Ekspektasi request valid lolos, didapat error: %v", err)
	}

	// 2. Nil request
	if err := service.ValidateDistributionCreate(nil); err == nil {
		t.Error("Ekspektasi nil request gagal validasi")
	}

	// 3. Panjang nama > 100 karakter
	longNameReq := *validReq
	longNameReq.RecipientName = strings.Repeat("A", 101)
	err := service.ValidateDistributionCreate(&longNameReq)
	var valErr *service.ValidationError
	if !errors.As(err, &valErr) || valErr.Field != "recipient_name" || valErr.Message == "" {
		t.Errorf("Ekspektasi ValidationError pada 'recipient_name' untuk nama > 100 karakter, didapat: %v", err)
	}

	// 4. Panjang RT > 10 karakter
	longRTReq := *validReq
	longRTReq.RecipientRT = strings.Repeat("1", 11)
	err = service.ValidateDistributionCreate(&longRTReq)
	if !errors.As(err, &valErr) || valErr.Field != "recipient_rt" || valErr.Message == "" {
		t.Errorf("Ekspektasi ValidationError pada 'recipient_rt' untuk RT > 10 karakter, didapat: %v", err)
	}

	// 5. ValidateCouponCode kosong & > 30 karakter
	if err := service.ValidateCouponCode(""); !errors.As(err, &valErr) || valErr.Field != "coupon_code" {
		t.Errorf("Ekspektasi ValidationError pada 'coupon_code' kosong, didapat: %v", err)
	}
	if err := service.ValidateCouponCode(strings.Repeat("X", 31)); !errors.As(err, &valErr) || valErr.Field != "coupon_code" {
		t.Errorf("Ekspektasi ValidationError pada 'coupon_code' > 30 karakter, didapat: %v", err)
	}

	// 6. ValidateActorID
	if err := service.ValidateActorID(0); !errors.As(err, &valErr) || valErr.Field != "actor_id" {
		t.Errorf("Ekspektasi ValidationError pada 'actor_id' <= 0, didapat: %v", err)
	}
	if err := service.ValidateActorID(5); err != nil {
		t.Errorf("Ekspektasi actor_id positif valid, didapat: %v", err)
	}

	// 7. ValidateAnimalForDistribution
	if err := service.ValidateAnimalForDistribution(nil); !errors.As(err, &valErr) || valErr.Field != "animal_id" {
		t.Errorf("Ekspektasi ValidationError pada nil animal, didapat: %v", err)
	}
}
