package model

import "time"

// Konstanta status kupon distribusi paket daging
const (
	DistributionStatusAllocated = "allocated"
	DistributionStatusRedeemed  = "redeemed"
	DistributionStatusCancelled = "cancelled"
)

// Distribution merepresentasikan alokasi kupon distribusi paket daging kepada penerima.
type Distribution struct {
	ID            int        `json:"id"`
	CouponCode    string     `json:"coupon_code"`
	AnimalID      int        `json:"animal_id"`
	CoordinatorID int        `json:"coordinator_id"`
	RecipientNIK  string     `json:"recipient_nik"`
	RecipientName string     `json:"recipient_name"`
	RecipientRT   string     `json:"recipient_rt"`
	Status        string     `json:"status"`
	RedeemedAt    *time.Time `json:"redeemed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// DistributionCreateRequest adalah DTO request pembuatan alokasi distribusi baru.
type DistributionCreateRequest struct {
	AnimalID      int    `json:"animal_id"`
	CoordinatorID int    `json:"coordinator_id"`
	RecipientNIK  string `json:"recipient_nik"`
	RecipientName string `json:"recipient_name"`
	RecipientRT   string `json:"recipient_rt"`
}
