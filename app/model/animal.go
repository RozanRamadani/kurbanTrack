package model

import "time"

// Konstanta spesies hewan qurban yang didukung
const (
	SpeciesSapi    = "sapi"
	SpeciesKambing = "kambing"
	SpeciesDomba   = "domba"
)

// Konstanta status siklus hidup hewan qurban
const (
	AnimalStatusRegistered  = "registered"
	AnimalStatusSlaughtered = "slaughtered"
	AnimalStatusDistributed = "distributed"
)

// Animal merepresentasikan entitas hewan qurban dalam sistem KurbanTrack.
// Tipe float64 digunakan untuk weight_kg yang sesuai dengan tipe PostgreSQL NUMERIC(6,2).
type Animal struct {
	ID                int       `json:"id"`
	Code              string    `json:"code"`
	Species           string    `json:"species"`
	SohibulName       string    `json:"sohibul_name"`
	WeightKg          float64   `json:"weight_kg"`
	EstimatedPackages int       `json:"estimated_packages"`
	AllocatedPackages int       `json:"allocated_packages"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
}

// AnimalCreateRequest adalah DTO request pendaftaran hewan qurban baru.
type AnimalCreateRequest struct {
	Code              string  `json:"code"`
	Species           string  `json:"species"`
	SohibulName       string  `json:"sohibul_name"`
	WeightKg          float64 `json:"weight_kg"`
	EstimatedPackages int     `json:"estimated_packages"`
}

// AnimalUpdateRequest adalah DTO request pembaruan data hewan qurban.
type AnimalUpdateRequest struct {
	Code              string  `json:"code"`
	Species           string  `json:"species"`
	SohibulName       string  `json:"sohibul_name"`
	WeightKg          float64 `json:"weight_kg"`
	EstimatedPackages int     `json:"estimated_packages"`
	Status            string  `json:"status"`
}
