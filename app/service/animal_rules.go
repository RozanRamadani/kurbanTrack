package service

import (
	"math"
	"strings"

	"kurbantrack-backend/app/model"
)

// isValidTwoDecimals memvalidasi apakah angka floating-point memiliki maksimal dua angka di belakang koma.
func isValidTwoDecimals(val float64) bool {
	scaled := val * 100
	return math.Abs(scaled-math.Round(scaled)) < 1e-6
}

// ValidateAnimalCreate memvalidasi aturan bisnis untuk pembuatan data hewan baru.
func ValidateAnimalCreate(req *model.AnimalCreateRequest) error {
	if strings.TrimSpace(req.Code) == "" {
		return &ValidationError{Field: "code", Message: "Kode hewan wajib diisi"}
	}

	if req.Species != model.SpeciesSapi && req.Species != model.SpeciesKambing && req.Species != model.SpeciesDomba {
		return &ValidationError{Field: "species", Message: "Spesies hewan harus berupa 'sapi', 'kambing', atau 'domba'"}
	}

	if strings.TrimSpace(req.SohibulName) == "" {
		return &ValidationError{Field: "sohibul_name", Message: "Nama sohibul wajib diisi"}
	}

	if req.WeightKg <= 0 {
		return &ValidationError{Field: "weight_kg", Message: "Berat hewan harus lebih besar dari 0"}
	}

	if req.WeightKg > 9999.99 {
		return &ValidationError{Field: "weight_kg", Message: "Berat hewan tidak boleh melebihi 9999.99 kg"}
	}

	if !isValidTwoDecimals(req.WeightKg) {
		return &ValidationError{Field: "weight_kg", Message: "Berat hewan maksimal memiliki 2 angka desimal"}
	}

	if req.EstimatedPackages <= 0 {
		return &ValidationError{Field: "estimated_packages", Message: "Estimasi jumlah paket harus lebih besar dari 0"}
	}

	return nil
}

// ValidateAnimalUpdate memvalidasi aturan bisnis untuk pembaruan data hewan qurban.
func ValidateAnimalUpdate(req *model.AnimalUpdateRequest, currentAllocated int, activeAllocations int) error {
	if strings.TrimSpace(req.Code) == "" {
		return &ValidationError{Field: "code", Message: "Kode hewan wajib diisi"}
	}

	if req.Species != model.SpeciesSapi && req.Species != model.SpeciesKambing && req.Species != model.SpeciesDomba {
		return &ValidationError{Field: "species", Message: "Spesies hewan harus berupa 'sapi', 'kambing', atau 'domba'"}
	}

	if strings.TrimSpace(req.SohibulName) == "" {
		return &ValidationError{Field: "sohibul_name", Message: "Nama sohibul wajib diisi"}
	}

	if req.WeightKg <= 0 {
		return &ValidationError{Field: "weight_kg", Message: "Berat hewan harus lebih besar dari 0"}
	}

	if req.WeightKg > 9999.99 {
		return &ValidationError{Field: "weight_kg", Message: "Berat hewan tidak boleh melebihi 9999.99 kg"}
	}

	if !isValidTwoDecimals(req.WeightKg) {
		return &ValidationError{Field: "weight_kg", Message: "Berat hewan maksimal memiliki 2 angka desimal"}
	}

	if req.EstimatedPackages <= 0 {
		return &ValidationError{Field: "estimated_packages", Message: "Estimasi jumlah paket harus lebih besar dari 0"}
	}

	if req.EstimatedPackages < activeAllocations {
		return &ValidationError{Field: "estimated_packages", Message: "Estimasi paket tidak boleh lebih kecil daripada jumlah alokasi aktif yang sudah ada"}
	}

	if req.EstimatedPackages < currentAllocated {
		return &ValidationError{Field: "estimated_packages", Message: "Estimasi paket tidak boleh lebih kecil daripada jumlah paket yang sudah teralokasi"}
	}

	if req.Status != "" &&
		req.Status != model.AnimalStatusRegistered &&
		req.Status != model.AnimalStatusSlaughtered &&
		req.Status != model.AnimalStatusDistributed {
		return &ValidationError{Field: "status", Message: "Status hewan harus berupa 'registered', 'slaughtered', atau 'distributed'"}
	}

	return nil
}
