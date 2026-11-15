package service

import (
	"strings"
	"unicode/utf8"

	"kurbantrack-backend/app/model"
)

// isValidNIK memeriksa apakah string NIK terdiri dari tepat 16 digit angka (0-9).
func isValidNIK(nik string) bool {
	if len(nik) != 16 {
		return false
	}
	for i := 0; i < len(nik); i++ {
		if nik[i] < '0' || nik[i] > '9' {
			return false
		}
	}
	return true
}

// ValidateDistributionCreate memvalidasi input request pembuatan alokasi kupon distribusi baru.
func ValidateDistributionCreate(req *model.DistributionCreateRequest) error {
	if req == nil {
		return &ValidationError{Field: "request", Message: "Data request distribusi tidak boleh kosong"}
	}

	if req.AnimalID <= 0 {
		return &ValidationError{Field: "animal_id", Message: "ID hewan harus lebih besar dari 0"}
	}

	if req.CoordinatorID <= 0 {
		return &ValidationError{Field: "coordinator_id", Message: "ID koordinator harus lebih besar dari 0"}
	}

	nik := strings.TrimSpace(req.RecipientNIK)
	if nik == "" {
		return &ValidationError{Field: "recipient_nik", Message: "NIK penerima wajib diisi"}
	}
	if !isValidNIK(nik) {
		return &ValidationError{Field: "recipient_nik", Message: "NIK penerima harus terdiri dari tepat 16 digit angka"}
	}

	name := strings.TrimSpace(req.RecipientName)
	if name == "" {
		return &ValidationError{Field: "recipient_name", Message: "Nama penerima wajib diisi"}
	}
	if utf8.RuneCountInString(name) > 100 {
		return &ValidationError{Field: "recipient_name", Message: "Nama penerima maksimal 100 karakter"}
	}

	rt := strings.TrimSpace(req.RecipientRT)
	if rt == "" {
		return &ValidationError{Field: "recipient_rt", Message: "RT penerima wajib diisi"}
	}
	if utf8.RuneCountInString(rt) > 10 {
		return &ValidationError{Field: "recipient_rt", Message: "RT penerima maksimal 10 karakter"}
	}

	return nil
}

// ValidateAnimalForDistribution memvalidasi kelayakan status hewan dan ketersediaan kuota paket daging.
func ValidateAnimalForDistribution(animal *model.Animal) error {
	if animal == nil {
		return &ValidationError{Field: "animal_id", Message: "Data hewan qurban tidak ditemukan"}
	}

	if animal.Status != model.AnimalStatusSlaughtered {
		return &ValidationError{
			Field:   "animal_id",
			Message: "Alokasi kupon hanya dapat dilakukan untuk hewan berstatus 'slaughtered'",
		}
	}

	remainingQuota := animal.EstimatedPackages - animal.AllocatedPackages
	if remainingQuota <= 0 {
		return &ValidationError{
			Field:   "animal_id",
			Message: "Kuota paket daging untuk hewan ini sudah habis",
		}
	}

	return nil
}

// ValidateCouponCode memvalidasi parameter kode kupon.
func ValidateCouponCode(couponCode string) error {
	trimmed := strings.TrimSpace(couponCode)
	if trimmed == "" {
		return &ValidationError{Field: "coupon_code", Message: "Kode kupon wajib diisi"}
	}
	if len(trimmed) > 30 {
		return &ValidationError{Field: "coupon_code", Message: "Kode kupon maksimal 30 karakter"}
	}
	return nil
}

// ValidateActorID memvalidasi identitas pengguna (actor) yang melakukan aksi perubahan status kupon.
func ValidateActorID(actorID int) error {
	if actorID <= 0 {
		return &ValidationError{Field: "actor_id", Message: "ID aktor harus lebih besar dari 0"}
	}
	return nil
}

// ValidateDistributionRedeem memvalidasi apakah kupon memenuhi syarat untuk ditebus.
func ValidateDistributionRedeem(dist *model.Distribution) error {
	if dist == nil {
		return &ValidationError{Field: "coupon_code", Message: "Data kupon distribusi tidak ditemukan"}
	}

	if dist.Status == model.DistributionStatusRedeemed {
		return &ValidationError{
			Field:   "status",
			Message: "Kupon sudah ditebus sebelumnya dan tidak dapat ditebus kembali",
		}
	}

	if dist.Status == model.DistributionStatusCancelled {
		return &ValidationError{
			Field:   "status",
			Message: "Kupon yang sudah dibatalkan tidak dapat ditebus",
		}
	}

	if dist.Status != model.DistributionStatusAllocated {
		return &ValidationError{
			Field:   "status",
			Message: "Hanya kupon berstatus 'allocated' yang dapat ditebus",
		}
	}

	return nil
}

// ValidateDistributionCancel memvalidasi apakah kupon memenuhi syarat untuk dibatalkan.
func ValidateDistributionCancel(dist *model.Distribution) error {
	if dist == nil {
		return &ValidationError{Field: "coupon_code", Message: "Data kupon distribusi tidak ditemukan"}
	}

	if dist.Status == model.DistributionStatusRedeemed {
		return &ValidationError{
			Field:   "status",
			Message: "Kupon yang sudah ditebus tidak dapat dibatalkan",
		}
	}

	if dist.Status == model.DistributionStatusCancelled {
		return &ValidationError{
			Field:   "status",
			Message: "Kupon sudah dibatalkan sebelumnya dan tidak dapat dibatalkan kembali",
		}
	}

	if dist.Status != model.DistributionStatusAllocated {
		return &ValidationError{
			Field:   "status",
			Message: "Hanya kupon berstatus 'allocated' yang dapat dibatalkan",
		}
	}

	return nil
}
