package model_test

import (
	"encoding/json"
	"testing"
	"time"

	"kurbantrack-backend/app/model"
)

func TestModel_Constants(t *testing.T) {
	// Memastikan konstanta role sesuai batasan SRS & migrasi
	if model.RoleAdmin != "admin" ||
		model.RoleKoordinatorWilayah != "koordinator_wilayah" ||
		model.RolePanitiaDistribusi != "panitia_distribusi" {
		t.Errorf("Konstanta Role tidak sesuai")
	}

	// Memastikan konstanta spesies hewan
	if model.SpeciesSapi != "sapi" ||
		model.SpeciesKambing != "kambing" ||
		model.SpeciesDomba != "domba" {
		t.Errorf("Konstanta Species tidak sesuai")
	}

	// Memastikan konstanta status hewan
	if model.AnimalStatusRegistered != "registered" ||
		model.AnimalStatusSlaughtered != "slaughtered" ||
		model.AnimalStatusDistributed != "distributed" {
		t.Errorf("Konstanta Status Hewan tidak sesuai")
	}

	// Memastikan konstanta status distribusi
	if model.DistributionStatusAllocated != "allocated" ||
		model.DistributionStatusRedeemed != "redeemed" ||
		model.DistributionStatusCancelled != "cancelled" {
		t.Errorf("Konstanta Status Distribusi tidak sesuai")
	}
}

func TestModel_Animal_JSONSerialization(t *testing.T) {
	now := time.Now()
	animal := model.Animal{
		ID:                1,
		Code:              "SP-001",
		Species:           model.SpeciesSapi,
		SohibulName:       "Ahmad",
		WeightKg:          350.50,
		EstimatedPackages: 100,
		AllocatedPackages: 25,
		Status:            model.AnimalStatusRegistered,
		CreatedAt:         now,
	}

	bytes, err := json.Marshal(animal)
	if err != nil {
		t.Fatalf("Gagal serialize Animal ke JSON: %v", err)
	}

	var unmarshaled model.Animal
	if err := json.Unmarshal(bytes, &unmarshaled); err != nil {
		t.Fatalf("Gagal deserialize JSON ke Animal: %v", err)
	}

	if unmarshaled.Code != "SP-001" || unmarshaled.WeightKg != 350.50 {
		t.Errorf("Data hasil deserialisasi tidak sesuai: %+v", unmarshaled)
	}
}

func TestModel_User_PasswordHash_ExcludedFromJSON(t *testing.T) {
	user := model.User{
		ID:           1,
		Name:         "Admin Kurban",
		Email:        "admin@kurbantrack.local",
		PasswordHash: "secret_hash_value",
		Role:         model.RoleAdmin,
		CreatedAt:    time.Now(),
	}

	bytes, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Gagal serialize User ke JSON: %v", err)
	}

	jsonString := string(bytes)
	if jsonString == "" {
		t.Fatal("Hasil JSON kosong")
	}

	// PasswordHash harus memiliki tag `json:"-"` sehingga tidak bocor ke JSON
	var parsedMap map[string]interface{}
	_ = json.Unmarshal(bytes, &parsedMap)

	if _, exists := parsedMap["password_hash"]; exists {
		t.Errorf("PasswordHash bocor ke dalam JSON: %s", jsonString)
	}
	if _, exists := parsedMap["PasswordHash"]; exists {
		t.Errorf("PasswordHash bocor ke dalam JSON: %s", jsonString)
	}
}
