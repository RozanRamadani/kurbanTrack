package service_test

import (
	"strings"
	"testing"

	"kurbantrack-backend/app/model"
	"kurbantrack-backend/app/service"
)

func TestValidateDistributionCreate_Valid(t *testing.T) {
	req := &model.DistributionCreateRequest{
		AnimalID:      1,
		CoordinatorID: 2,
		RecipientNIK:  "3201012345678901",
		RecipientName: "Budi Santoso",
		RecipientRT:   "RT03",
	}

	if err := service.ValidateDistributionCreate(req); err != nil {
		t.Errorf("Valid request tidak boleh menghasilkan error: %v", err)
	}
}

func TestValidateDistributionCreate_NilRequest(t *testing.T) {
	err := service.ValidateDistributionCreate(nil)
	if err == nil {
		t.Error("Request nil seharusnya menghasilkan error")
	}
}

func TestValidateDistributionCreate_InvalidIDs(t *testing.T) {
	req := &model.DistributionCreateRequest{
		AnimalID:      0,
		CoordinatorID: 1,
		RecipientNIK:  "3201012345678901",
		RecipientName: "Budi",
		RecipientRT:   "RT01",
	}

	if err := service.ValidateDistributionCreate(req); err == nil {
		t.Error("AnimalID <= 0 seharusnya menghasilkan error")
	}

	req.AnimalID = 1
	req.CoordinatorID = 0
	if err := service.ValidateDistributionCreate(req); err == nil {
		t.Error("CoordinatorID <= 0 seharusnya menghasilkan error")
	}
}

func TestValidateDistributionCreate_InvalidNIK(t *testing.T) {
	invalidNIKs := []string{
		"",
		"   ",
		"12345",             // Kurang dari 16 digit
		"12345678901234567", // Lebih dari 16 digit
		"320101234567890a",  // Mengandung huruf
		"320101234567890-",  // Mengandung tanda baca
	}

	for _, nik := range invalidNIKs {
		req := &model.DistributionCreateRequest{
			AnimalID:      1,
			CoordinatorID: 1,
			RecipientNIK:  nik,
			RecipientName: "Budi",
			RecipientRT:   "RT01",
		}

		if err := service.ValidateDistributionCreate(req); err == nil {
			t.Errorf("NIK %q seharusnya menghasilkan error validasi", nik)
		}
	}
}

func TestValidateDistributionCreate_InvalidName(t *testing.T) {
	req := &model.DistributionCreateRequest{
		AnimalID:      1,
		CoordinatorID: 1,
		RecipientNIK:  "3201012345678901",
		RecipientName: "",
		RecipientRT:   "RT01",
	}

	if err := service.ValidateDistributionCreate(req); err == nil {
		t.Error("Nama penerima kosong seharusnya menghasilkan error")
	}

	req.RecipientName = strings.Repeat("A", 101)
	if err := service.ValidateDistributionCreate(req); err == nil {
		t.Error("Nama penerima lebih dari 100 karakter seharusnya menghasilkan error")
	}
}

func TestValidateDistributionCreate_InvalidRT(t *testing.T) {
	req := &model.DistributionCreateRequest{
		AnimalID:      1,
		CoordinatorID: 1,
		RecipientNIK:  "3201012345678901",
		RecipientName: "Budi",
		RecipientRT:   "",
	}

	if err := service.ValidateDistributionCreate(req); err == nil {
		t.Error("RT penerima kosong seharusnya menghasilkan error")
	}

	req.RecipientRT = "RT0123456789" // 12 chars
	if err := service.ValidateDistributionCreate(req); err == nil {
		t.Error("RT penerima lebih dari 10 karakter seharusnya menghasilkan error")
	}
}

func TestValidateAnimalForDistribution_Valid(t *testing.T) {
	animal := &model.Animal{
		ID:                1,
		Status:            model.AnimalStatusSlaughtered,
		EstimatedPackages: 100,
		AllocatedPackages: 50,
	}

	if err := service.ValidateAnimalForDistribution(animal); err != nil {
		t.Errorf("Hewan layak distribusi tidak boleh menghasilkan error: %v", err)
	}
}

func TestValidateAnimalForDistribution_InvalidStatus(t *testing.T) {
	animal := &model.Animal{
		ID:                1,
		Status:            model.AnimalStatusRegistered,
		EstimatedPackages: 100,
		AllocatedPackages: 0,
	}

	if err := service.ValidateAnimalForDistribution(animal); err == nil {
		t.Error("Hewan berstatus 'registered' tidak boleh dialokasikan untuk distribusi")
	}
}

func TestValidateAnimalForDistribution_QuotaExceeded(t *testing.T) {
	animal := &model.Animal{
		ID:                1,
		Status:            model.AnimalStatusSlaughtered,
		EstimatedPackages: 50,
		AllocatedPackages: 50,
	}

	if err := service.ValidateAnimalForDistribution(animal); err == nil {
		t.Error("Hewan dengan kuota habis seharusnya menghasilkan error")
	}
}

func TestValidateCouponCode_ValidAndInvalid(t *testing.T) {
	if err := service.ValidateCouponCode("KBN-12345"); err != nil {
		t.Errorf("Kode kupon valid tidak boleh error: %v", err)
	}

	if err := service.ValidateCouponCode(""); err == nil {
		t.Error("Kode kupon kosong seharusnya error")
	}

	if err := service.ValidateCouponCode(strings.Repeat("X", 31)); err == nil {
		t.Error("Kode kupon > 30 karakter seharusnya error")
	}
}

func TestValidateActorID_ValidAndInvalid(t *testing.T) {
	if err := service.ValidateActorID(1); err != nil {
		t.Errorf("Actor ID > 0 tidak boleh error: %v", err)
	}

	if err := service.ValidateActorID(0); err == nil {
		t.Error("Actor ID <= 0 seharusnya error")
	}
}

func TestValidateDistributionRedeem_Valid(t *testing.T) {
	dist := &model.Distribution{
		CouponCode: "KBN-123",
		Status:     model.DistributionStatusAllocated,
	}

	if err := service.ValidateDistributionRedeem(dist); err != nil {
		t.Errorf("Kupon berstatus 'allocated' harus valid untuk ditebus: %v", err)
	}
}

func TestValidateDistributionRedeem_InvalidStatus(t *testing.T) {
	distRedeemed := &model.Distribution{
		CouponCode: "KBN-123",
		Status:     model.DistributionStatusRedeemed,
	}
	if err := service.ValidateDistributionRedeem(distRedeemed); err == nil {
		t.Error("Kupon yang sudah ditebus tidak boleh ditebus ulang")
	}

	distCancelled := &model.Distribution{
		CouponCode: "KBN-123",
		Status:     model.DistributionStatusCancelled,
	}
	if err := service.ValidateDistributionRedeem(distCancelled); err == nil {
		t.Error("Kupon yang sudah dibatalkan tidak boleh ditebus")
	}
}

func TestValidateDistributionCancel_Valid(t *testing.T) {
	dist := &model.Distribution{
		CouponCode: "KBN-123",
		Status:     model.DistributionStatusAllocated,
	}

	if err := service.ValidateDistributionCancel(dist); err != nil {
		t.Errorf("Kupon berstatus 'allocated' harus valid untuk dibatalkan: %v", err)
	}
}

func TestValidateDistributionCancel_InvalidStatus(t *testing.T) {
	distRedeemed := &model.Distribution{
		CouponCode: "KBN-123",
		Status:     model.DistributionStatusRedeemed,
	}
	if err := service.ValidateDistributionCancel(distRedeemed); err == nil {
		t.Error("Kupon yang sudah ditebus tidak boleh dibatalkan")
	}

	distCancelled := &model.Distribution{
		CouponCode: "KBN-123",
		Status:     model.DistributionStatusCancelled,
	}
	if err := service.ValidateDistributionCancel(distCancelled); err == nil {
		t.Error("Kupon yang sudah dibatalkan tidak boleh dibatalkan ulang")
	}
}
